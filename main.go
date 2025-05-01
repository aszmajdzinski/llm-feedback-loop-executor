package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"example.com/web-app-creator/agents"
	fileutils "example.com/web-app-creator/file_utils"
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
	Name        string `yaml:"name"`
	Iterations  int    `yaml:"iterations"`
	FilesOutput bool   `yaml:"filesOutput"`
	Worker      struct {
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
	providers := map[string]llm.LLMProvider{
		"openai": llm.NewOpenAIWithStructuredOutputProvider(openAIAPIKey, "gpt-4o-mini", ""),
	}

	previousBlockOutput := ""
	for bn, b := range appSetup.Blocks {
		logger.Info("Running block", "name", b.Name)

		ans, err := RunBlock(ctx, b, previousBlockOutput, providers)
		if err != nil {
			logger.Error("error running block", "block", b.Name, "error", err)
			os.Exit(1)
		}
		logger.Debug(
			"Finished running block", "block name", b.Name)

		partialOutputsDir := filepath.Join(
			os.Getenv("OUTPUT_DIRECTORY"),
			"conversations",
			fileutils.ToKebabCase(fmt.Sprintf("%03d %s", bn, b.Name)),
		)

		err = SaveBlockAnswer(ctx, partialOutputsDir, b, ans)
		if err != nil {
			logger.Error("Error saving block answer", "block", b.Name, "error", err)
			os.Exit(1)
		}

		previousBlockOutput = ans.FinalAnswer

		blockFinalAnswerDir := filepath.Join(
			os.Getenv("OUTPUT_DIRECTORY"),
			"answers",
			fileutils.ToKebabCase(fmt.Sprintf("%03d %s", bn, b.Name)),
		)

		if b.FilesOutput {
			err = os.MkdirAll(blockFinalAnswerDir, 0o755)
			if err != nil {
				logger.Error("Error creating block answer directory", "error", err)
				os.Exit(1)
			}
			fileutils.SaveFilesFromJson(blockFinalAnswerDir, []byte(ans.FinalAnswer))
		}
	}

	fmt.Println(previousBlockOutput)
}

func RunBlock(
	ctx context.Context,
	blockData Block,
	additionalData string,
	providers map[string]llm.LLMProvider,
) (thinkingblock.ThinkingBlockOutput, error) {
	provider := providers["openai"]

	worker := agents.Agent{
		Name:         blockData.Worker.Name,
		SystemPrompt: blockData.Worker.System,
		Llm:          provider,
	}

	var experts []agents.Agent
	for _, a := range blockData.Experts {
		experts = append(experts, agents.Agent{
			Name:         a.Name,
			SystemPrompt: a.System,
			Llm:          provider,
		},
		)
	}

	oracle := agents.Agent{
		Name: blockData.Oracle.Name,
		Llm:  provider,
	}

	thinkingBlock := thinkingblock.ThinkingBlock{
		Worker:      worker,
		ExpertsTeam: &agents.ExpertsTeam{Experts: experts},
		Oracle:      oracle,
	}

	out, err := thinkingBlock.Run(
		ctx,
		string(blockData.Worker.Prompt),
		additionalData,
		blockData.FilesOutput,
		blockData.Iterations,
	)
	if err != nil {
		return thinkingblock.ThinkingBlockOutput{}, fmt.Errorf(
			"error running thinking block: %v",
			err.Error(),
		)
	}

	return out, nil
}

func SaveBlockAnswer(
	ctx context.Context,
	outputDir string,
	blockData Block,
	answer thinkingblock.ThinkingBlockOutput,
) error {
	logger := loggerutils.GetLogger(ctx)
	logger.Info("Saving answers", "block name", blockData.Name)

	err := os.MkdirAll(outputDir, 0o755)
	if err != nil {
		return err
	}

	for paIdx, pa := range answer.PartAnswers {
		ansFileName := fileutils.CreateTxtFilename(
			outputDir,
			paIdx,
			"1-"+blockData.Worker.Name,
			"response",
		)
		err := fileutils.WriteToFile(ansFileName, pa.WorkerSolution)
		if err != nil {
			logger.Error("error writing to file", "error", err)
		}

		for ean, ea := range pa.ExpertAnswers {
			ansFileName = fileutils.CreateTxtFilename(
				outputDir,
				paIdx,
				"2-"+blockData.Experts[ean].Name,
				"response",
			)
			err = fileutils.WriteToFile(ansFileName, ea)
			if err != nil {
				logger.Error("error writing to file", "error", err)
			}
		}

		ansFileName = fileutils.CreateTxtFilename(outputDir, paIdx, "3-"+blockData.Oracle.Name, "response")
		err = fileutils.WriteToFile(ansFileName, pa.OracleSummary)
		if err != nil {
			logger.Error("error writing to file", "error", err)
		}
	}

	for pIdx, p := range answer.Prompts {
		promptFileName := fileutils.CreateTxtFilename(
			outputDir,
			pIdx,
			"1-"+blockData.Worker.Name,
			"prompt",
		)
		err := fileutils.WriteToFile(promptFileName, p.WorkerPrompt)
		if err != nil {
			logger.Error("error writing to file", "error", err)
		}

		promptFileName = fileutils.CreateTxtFilename(outputDir, pIdx, "2-"+"experts", "prompt")
		err = fileutils.WriteToFile(promptFileName, p.ExpertsPrompt)
		if err != nil {
			logger.Error("error writing to file", "error", err)
		}

		promptFileName = fileutils.CreateTxtFilename(
			outputDir,
			pIdx,
			"3-"+blockData.Oracle.Name,
			"prompt",
		)
		err = fileutils.WriteToFile(promptFileName, p.OraclePrompt)
		if err != nil {
			logger.Error("error writing to file", "error", err)
		}
	}

	return nil
}
