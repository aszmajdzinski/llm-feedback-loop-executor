package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"example.com/web-app-creator/agents"
	"example.com/web-app-creator/llm"
	loggerutils "example.com/web-app-creator/logger_utils"
	thinkingblock "example.com/web-app-creator/thinking_block"
	_ "github.com/joho/godotenv/autoload"
	"gopkg.in/yaml.v3"
)

type AppSetup struct {
	Blocks []Block `yaml:"blocks"`
}

type Block struct {
	Name       string `yaml:"name"`
	Iterations int    `yaml:"iterations"`
	Worker     struct {
		Name   string `yaml:"name"`
		System string `yaml:"system"`
		Prompt string `yaml:"prompt"`
	} `yaml:"worker"`
	Experts []struct {
		Name   string `yaml:"name"`
		System string `yaml:"system"`
	} `yaml:"experts"`
	Oracle struct {
		Name   string `yaml:"name"`
		System string `yaml:"system"`
	} `yaml:"oracle"`
}

func main() {
	logger := loggerutils.SetupLogger()
	ctx := context.TODO()
	ctx = loggerutils.WithLogger(ctx, logger)

	data, err := os.ReadFile("system_prompts.yaml")
	if err != nil {
		log.Fatalf("error reading prompts file: %v", err)
	}

	var appSetup AppSetup
	err = yaml.Unmarshal(data, &appSetup)
	if err != nil {
		log.Fatalf("failed unmarshaling yaml: %v", err)
	}

	openAIAPIKey := os.Getenv("OPENAI_API_KEY")
	providers := map[string]llm.LlmProvider{
		"openai": llm.NewOpenAIProvider(openAIAPIKey, ""),
	}

	for bn, b := range appSetup.Blocks {
		logger.Info("Running block", " name", b.Name)
		ans, err := RunBlock(ctx, b, providers)
		if err != nil {
			logger.Error("error running block", "block", b.Name, "error", err)
			os.Exit(1)
		}
		logger.Debug(
			"Finished running block",
			"block name",
			b.Name,
			"part answers count",
			len(ans.PartAnswers),
			"final answer lenght",
			len(ans.FinalAnswer),
		)

		o := filepath.Join(
			os.Getenv("OUTPUT_DIRECTORY"),
			toKebabCase(fmt.Sprintf("%03d %s", bn, b.Name)),
		)
		SaveBlockAnswer(ctx, o, b, ans)
	}
}

func RunBlock(
	ctx context.Context,
	blockData Block,
	providers map[string]llm.LlmProvider,
) (thinkingblock.ThinkingBlockAnswer, error) {
	provider := providers["openai"]

	worker := agents.Agent{
		Name:         blockData.Worker.Name,
		SystemPrompt: blockData.Worker.System,
		Model:        "gpt-4o-mini",
		Llm:          provider,
	}

	var experts []agents.Agent
	for _, a := range blockData.Experts {
		experts = append(experts, agents.Agent{
			Name:         a.Name,
			SystemPrompt: a.System,
			Model:        "gpt-4o-mini",
			Llm:          provider,
		},
		)
	}

	oracle := agents.Agent{
		Name:  blockData.Oracle.Name,
		Model: "gpt-4o-mini",
		Llm:   provider,
	}

	thinkingBlock := thinkingblock.ThinkingBlock{
		Worker:      worker,
		ExpertsTeam: agents.ExpertsTeam{Experts: experts},
		Oracle:      oracle,
	}

	ans, err := thinkingBlock.Run(ctx, string(blockData.Worker.Prompt), blockData.Iterations)
	if err != nil {
		return thinkingblock.ThinkingBlockAnswer{}, fmt.Errorf(
			"error running thinking block: %v",
			err.Error(),
		)
	}

	return ans, nil
}

func SaveBlockAnswer(
	ctx context.Context,
	outputDir string,
	blockData Block,
	answer thinkingblock.ThinkingBlockAnswer,
) {
	logger := loggerutils.GetLogger(ctx)
	logger.Info("Saving answers", "block name", blockData.Name)

	err := os.MkdirAll(outputDir, 0o755)
	if err != nil {
		logger.Error("error creating output directory", "error", err)
		os.Exit(1)
	}

	for paIdx, pa := range answer.PartAnswers {
		fileName := createTxtFilename(outputDir, paIdx, blockData.Worker.Name)
		logger.Debug("RunBlock: saving answer to", "file", fileName)

		err := writeToFile(fileName, pa.WorkerSolution)
		if err != nil {
			logger.Error("error writing to file", "error", err)
		}

		for ean, ea := range pa.ExpertAnswers {
			fileName = createTxtFilename(outputDir, paIdx, blockData.Experts[ean].Name)
			logger.Debug("RunBlock: saving answer to", "file", fileName)
			err = writeToFile(fileName, ea)
			if err != nil {
				logger.Error("error writing to file", "error", err)
			}
		}

		fileName = createTxtFilename(outputDir, paIdx, blockData.Oracle.Name)
		logger.Debug("RunBlock: saving answer to", "file", fileName)
		err = writeToFile(fileName, pa.OracleSummary)
		if err != nil {
			logger.Error("error writing to file", "error", err)
		}
	}
}

func toKebabCase(input string) string {
	var builder strings.Builder
	for _, r := range input {
		if r == ' ' {
			builder.WriteRune('-')
		} else {
			builder.WriteRune(unicode.ToLower(r))
		}
	}

	return builder.String()
}

func writeToFile(fileName string, content string) error {
	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		return err
	}

	return nil
}

func createTxtFilename(directory string, iteration int, name string) string {
	return filepath.Join(directory, toKebabCase(fmt.Sprintf("%03d %s.txt", iteration, name)))
}
