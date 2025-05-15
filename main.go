package main

import (
	"context"
	"flag"
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

	appSetupFile := flag.String("config", "", "Path to the app setup file")
	flag.Parse()

	if *appSetupFile == "" {
		log.Fatalf("usage: %s -config <app setup file>", os.Args[0])
	}

	appSetup, err := getAppData(*appSetupFile)
	if err != nil {
		log.Fatalf("failed loading app setup file: %v", err)
	}

	providers := createProviders()

	err = RunApp(ctx, appSetup, providers)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func RunApp(ctx context.Context, appSetup AppSetup, providers map[string]llm.LLMProvider) error {

	previousBlockOutput := ""
	for bn, b := range appSetup.Blocks {
		logger := loggerutils.GetLogger(ctx)

		logger.Info("Running block", "name", b.Name)

		ans, err := RunBlock(ctx, b, previousBlockOutput, providers)
		if err != nil {
			return fmt.Errorf("error running block %s: %s", b.Name, err.Error())
		}

		previousBlockOutput = ans.FinalAnswer

		partialOutputsDir := filepath.Join(
			os.Getenv("OUTPUT_DIRECTORY"),
			"conversations",
			fileutils.ToKebabCase(fmt.Sprintf("%03d %s", bn, b.Name)),
		)

		err = SaveBlockAnswer(ctx, partialOutputsDir, b, ans)
		if err != nil {
			return fmt.Errorf("error saving block %s: %s", b.Name, err.Error())
		}

		blockFinalAnswerDir := filepath.Join(
			os.Getenv("OUTPUT_DIRECTORY"),
			"answers",
			fileutils.ToKebabCase(fmt.Sprintf("%03d %s", bn, b.Name)),
		)

		if b.FilesOutput {
			err = os.MkdirAll(blockFinalAnswerDir, 0o755)
			if err != nil {
				return fmt.Errorf("error creating block answer directory %s: %s", b.Name, err.Error())

			}
			fileutils.SaveFilesFromJson(blockFinalAnswerDir, []byte(ans.FinalAnswer))
		}
	}
	return nil
}

func RunBlock(
	ctx context.Context,
	blockData Block,
	additionalData string,
	providers map[string]llm.LLMProvider,
) (thinkingblock.ThinkingBlockOutput, error) {
	provider := providers["openai"]

	worker, experts, oracle := createAgents(blockData, provider)

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

func createProviders() map[string]llm.LLMProvider {
	openAIAPIKey := os.Getenv("OPENAI_API_KEY")
	return map[string]llm.LLMProvider{
		"openai": llm.NewOpenAIWithStructuredOutputProvider(openAIAPIKey, "gpt-4o-mini", ""),
	}
}

func createAgents(blockData Block, provider llm.LLMProvider) (worker agents.Agent, experts []agents.Agent, oracle agents.Agent) {
	worker = agents.Agent{
		Name:         blockData.Worker.Name,
		SystemPrompt: blockData.Worker.System,
		Llm:          provider,
	}

	for _, a := range blockData.Experts {
		experts = append(experts, agents.Agent{
			Name:         a.Name,
			SystemPrompt: a.System,
			Llm:          provider,
		},
		)
	}

	oracle = agents.Agent{
		Name: blockData.Oracle.Name,
		Llm:  provider,
	}

	return
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

	// save all prompts
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

	// save all responses
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

	return nil
}

func getAppData(file string) (AppSetup, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return AppSetup{}, fmt.Errorf("error reading file %s: %v", file, err)
	}

	var appSetup AppSetup
	err = yaml.Unmarshal(data, &appSetup)
	if err != nil {
		return AppSetup{}, fmt.Errorf("failed unmarshaling yaml: %v", err)
	}

	return appSetup, nil
}
