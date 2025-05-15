package thinkingblock

import (
	"context"
	"fmt"

	"example.com/web-app-creator/assistants"
	loggerutils "example.com/web-app-creator/logger_utils"
)

const initialWorkerPrompt string = "You will be given a TASK. " +
	"Your job is to provide a solution to the TASK. " +
	"Ensure that your solution is as accurate and complete as possible."

const initialWorkerPromptWithData string = "You will be given a TASK and some DATA. " +
	"Your job is to provide a solution to the TASK using the provided DATA. " +
	"Ensure that your solution is as accurate and complete as possible."

const workerPromptWithSummary string = "You will be given a TASK, a SOLUTION and a SUMMARY of feedback from experts. " +
	"Your job is to refine the SOLUTION based on the feedback provided. " +
	"Ensure that the final solution is accurate, complete, and incorporates all the improvements suggested by the experts."

const workerPromptWithSummaryAndData string = "You will be given a TASK, some DATA, a SOLUTION, and a SUMMARY of feedback from experts. " +
	"Your job is to refine the SOLUTION based on the feedback provided and the DATA. " +
	"Ensure that the final solution is accurate, complete, and incorporates all the improvements suggested by the experts while utilizing the provided DATA."

const expertPrompt string = "You will be given a TASK and a SOLUTION. " +
	"Your job is to review the SOLUTION and provide feedback on its accuracy, " +
	"completeness, and any improvements that can be made. " +
	"Remember that you are an expert with all the needed knowledge and experience."

const expertPromptWithData string = "You will be given a TASK, some DATA, and a SOLUTION. " +
	"Your job is to review the SOLUTION using the provided TASK and DATA and provide feedback on its accuracy, " +
	"completeness, and any improvements that can be made. " +
	"Remember that you are an expert with all the needed knowledge and experience."

const oraclePrompt string = "You will be given a SOLUTION and its REVIEWS. " +
	"Your job is to summarize the key points from the reviews, " +
	"highlighting strengths, weaknesses, and suggestions for improvement. " +
	"Provide a concise and clear summary. Do not overthink, if you see that those reviews are positive enough and nothing more should be added to a SOLUTION, then simply answer \"OK\", without any other characters." +
	"Review will start with <REVIEW number>."

const oraclePromptWithData string = "You will be given a SOLUTION, its REVIEWS, and some DATA. " +
	"Your job is to summarize the key points from the reviews, " +
	"highlighting strengths, weaknesses, and suggestions for improvement while considering the provided DATA. " +
	"Provide a concise and clear summary. Do not overthink, if you see that those reviews are positive enough and nothing more should be added to a SOLUTION, then simply answer \"OK\", without any other characters." +
	"Review will start with <REVIEW number>."

type ThinkingBlockOutput struct {
	Prompts     []Prompts
	PartAnswers []PartialAnswer
	FinalAnswer string
}

type PartialAnswer struct {
	WorkerSolution string
	ExpertAnswers  []string
	OracleSummary  string
}

type Prompts struct {
	WorkerPrompt  string
	ExpertsPrompt string
	OraclePrompt  string
}

type ThinkingBlock struct {
	Worker      assistants.Assistant
	ExpertsTeam assistants.ExpertsTeamInterface
	Oracle      assistants.Assistant
}

func (tb *ThinkingBlock) Run(
	ctx context.Context,
	taskDescription string,
	data string,
	saveOutputFiles bool,
	iterations int,
) (ThinkingBlockOutput, error) {
	logger := loggerutils.GetLogger(ctx)
	blockOutput := ThinkingBlockOutput{}

	// a different prompt is used depending on whether additional data is provided
	// worker prompts
	var wPrompt, wPromptSummary string
	if data != "" {
		wPrompt = initialWorkerPromptWithData
		wPromptSummary = workerPromptWithSummaryAndData
	} else {
		wPrompt = initialWorkerPrompt
		wPromptSummary = workerPromptWithSummary
	}

	// experts prompts
	// a different prompt is used depending on whether additional data is provided
	var ePrompt string
	if data != "" {
		ePrompt = expertPromptWithData
	} else {
		ePrompt = expertPrompt
	}

	// oracle prompts
	var oPrompt string
	if data != "" {
		oPrompt = oraclePromptWithData
	} else {
		oPrompt = oraclePrompt
	}

	// use json schema for structured output or don't care about output format
	var s *map[string]any
	if saveOutputFiles {
		s = &schema
	}

	for i := range iterations {
		logger.Debug("Thinking block: iteration", "number", i)
		currentIterationAnswer := PartialAnswer{}
		currentIterationPrompts := Prompts{}

		// 1. Prepare worker prompt depending on whether it's a first attempt to complete a task
		// or it is making corrections according to review
		var wP string
		if i == 0 {
			// 1a. If it is a first iteration, ask worker to provide solution
			if data != "" {
				// using provided data
				wP = fmt.Sprintf("%s\nTASK: %s\nDATA: %s\n", wPrompt, taskDescription, data)
			} else {
				// no data provided, just a task
				wP = fmt.Sprintf("%s\nTASK: %s\n", wPrompt, taskDescription)
			}
		} else {
			// 1b. Else ask worker to refine a solution
			if data != "" {
				// using provided data
				wP = fmt.Sprintf(
					"%s\nTASK: %s\nSOLUTION: %s\nSUMMARY: %s\n DATA: %s\n",
					wPromptSummary,
					taskDescription,
					blockOutput.PartAnswers[i-1].WorkerSolution,
					blockOutput.PartAnswers[i-1].OracleSummary,
					data,
				)
			} else {
				// no data provided, just a task and a solution
				wP = fmt.Sprintf(
					"%s\nTASK: %s\nSOLUTION: %s\nSUMMARY: %s",
					wPromptSummary,
					taskDescription,
					blockOutput.PartAnswers[i-1].WorkerSolution,
					blockOutput.PartAnswers[i-1].OracleSummary,
				)
			}
		}

		// 2. Chat with worker and get solution proposal
		currentIterationPrompts.WorkerPrompt = wP
		solution, err := chat(ctx, tb.Worker, wP, s)

		if err != nil {
			return ThinkingBlockOutput{}, fmt.Errorf("error chatting with worker: %w", err)
		}
		currentIterationAnswer.WorkerSolution = solution

		// 3. Ask experts to review the proposal
		var eP string
		if data != "" {
			// using provided data
			eP = fmt.Sprintf(
				"%s\nTASK: %s\nDATA: %s\nSOLUTION %s",
				ePrompt,
				taskDescription,
				data,
				solution,
			)
		} else {
			// no data provided, just a task and a solution
			eP = fmt.Sprintf("%s\nTASK: %s\nSOLUTION %s", ePrompt, taskDescription, solution)
		}

		currentIterationPrompts.ExpertsPrompt = eP
		expertsAnswers := tb.ExpertsTeam.Ask(
			ctx,
			eP,
		)

		var reviews string
		for i, ea := range expertsAnswers {
			if ea.Error != nil {
				logger.Error("error chatting with expert", "error", ea.Error)
				continue
			}

			reviews += fmt.Sprintf("<REVIEW %d> %s\n", i, ea.Answer)
			currentIterationAnswer.ExpertAnswers = append(
				currentIterationAnswer.ExpertAnswers,
				ea.Answer,
			)
		}

		// 4. Provide those reviews to Oracle to sum up
		var oP string
		if data != "" {
			// using provided data
			oP = fmt.Sprintf(
				"%s\nSOLUTION: %s\nDATA: %s\nREVIEWS: %s\n",
				oPrompt,
				solution,
				data,
				reviews,
			)
		} else {
			// no data provided, just a solution and reviews
			oP = fmt.Sprintf("%s\nSOLUTION: %s\nREVIEWS: %s\n", oPrompt, solution, reviews)
		}
		summary, err := tb.Oracle.Chat(
			ctx,
			oP,
		)
		if err != nil {
			return ThinkingBlockOutput{}, fmt.Errorf("error chatting with oracle %w", err)
		}
		currentIterationAnswer.OracleSummary = summary
		currentIterationPrompts.OraclePrompt = oP

		blockOutput.PartAnswers = append(blockOutput.PartAnswers, currentIterationAnswer)
		blockOutput.Prompts = append(blockOutput.Prompts, currentIterationPrompts)

		if summary == "OK" {
			logger.Debug("Thinking block: Oracle told OK")
			break
		}

	}
	blockOutput.FinalAnswer = blockOutput.PartAnswers[len(blockOutput.PartAnswers)-1].WorkerSolution

	return blockOutput, nil
}

var schema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"files": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"fileName": map[string]any{
						"type":        "string",
						"description": "File name",
					},
					"fileContent": map[string]any{
						"type":        "string",
						"description": "File content",
					},
				},
				"required":             []string{"fileName", "fileContent"},
				"additionalProperties": false,
			},
			"description": "Lista plików",
		},
	},
	"required":             []string{"files"},
	"additionalProperties": false,
}

func chat(ctx context.Context, assistant assistants.Assistant, msg string, schema *map[string]any) (string, error) {
	if schema != nil {
		return assistant.StructuredChat(ctx, msg, assistant.Name, *schema)
	} else {
		return assistant.Chat(ctx, msg)
	}
}
