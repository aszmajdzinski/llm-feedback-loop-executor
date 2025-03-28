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
	provider := llm.NewOpenAIProvider(openAIAPIKey, "")

	for bn, b := range appSetup.Blocks {
		worker := agents.Agent{
			Name:         b.Worker.Name,
			SystemPrompt: b.Worker.System,
			Model:        "gpt-4o-mini",
			Llm:          provider,
		}

		var experts []agents.Agent
		for _, a := range b.Experts {
			experts = append(experts, agents.Agent{
				Name:         a.Name,
				SystemPrompt: a.System,
				Model:        "gpt-4o-mini",
				Llm:          provider,
			},
			)
		}

		oracle := agents.Agent{
			Name:  b.Oracle.Name,
			Model: "gpt-4o-mini",
			Llm:   provider,
		}

		thinkingBlock := thinkingblock.ThinkingBlock{
			Worker:      worker,
			ExpertsTeam: agents.ExpertsTeam{Experts: experts},
			Oracle:      oracle,
		}

		ans, err := thinkingBlock.Run(ctx, string(b.Worker.Prompt), b.Iterations)
		if err != nil {
			logger.Error(fmt.Sprintf("error running thinking block: %v", err.Error()))
		}

		out := os.Getenv("OUTPUT_DIRECTORY")
		createOutputDirectory(out)
		for paIdx, pa := range ans.PartAnswers {
			fileName := createTxtFilename(out, bn, b.Name, paIdx, b.Worker.Name)
			_ = writeToFile(fileName, pa.WorkerSolution)

			for ean, ea := range pa.ExpertAnswers {
				fileName := createTxtFilename(out, bn, b.Name, paIdx, b.Experts[ean].Name)
				_ = writeToFile(fileName, ea)
			}

			fileName = createTxtFilename(out, bn, b.Name, paIdx, b.Oracle.Name)
			_ = writeToFile(fileName, pa.OracleSummary)
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

func createOutputDirectory(path string) error {
	outputDir := path
	if _, err := os.Stat(outputDir); err == nil {
		err = os.RemoveAll(outputDir)
		if err != nil {
			return err
		}
	}

	return os.Mkdir(outputDir, 0o755)
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

func createTxtFilename(
	directory string,
	blockNumber int,
	blockName string,
	iteration int,
	name string,
) string {
	return filepath.Join(
		directory,
		toKebabCase(fmt.Sprintf("%03d %s %03d %s.txt", blockNumber, blockName, iteration, name)),
	)
}
