package thinkingblock

import (
	"context"
	"fmt"

	"example.com/web-app-creator/agents"
	loggerutils "example.com/web-app-creator/logger_utils"
)

const initialWorkerPrompt string = "You will be given a TASK. " +
	"Your job is to provide a solution to the TASK. " +
	"Ensure that your solution is as accurate and complete as possible."

const expertPrompt string = "You will be given a TASK and a SOLUTION. " +
	"Your job is to review the SOLUTION and provide feedback on its accuracy, " +
	"completeness, and any improvements that can be made. " +
	"Remember that you are an expert with all the needed knowledge and experience."

const oraclePrompt string = "You will be given a SOLUTION and its REVIEWS. " +
	"Your job is to summarize the key points from the reviews, " +
	"highlighting strengths, weaknesses, and suggestions for improvement. " +
	"Provide a concise and clear summary. Do not overthink, if you see that" +
	"those reviews are enough, then simply answer OK, without any other characters." +
	"Review will start with <REVIEW number>."

const workerPromptWithSummary string = "You will be given a TASK, a SOLUTION and a SUMMARY of feedback from experts. " +
	"Your job is to refine the SOLUTION based on the feedback provided. " +
	"Ensure that the final solution is accurate, complete, and incorporates all the improvements suggested by the experts."

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
	Worker      agents.Agent
	ExpertsTeam agents.ExpertsTeam
	Oracle      agents.Agent
}

func (tb *ThinkingBlock) Run(
	ctx context.Context,
	taskDescription string,
	iterations int,
) (ThinkingBlockOutput, error) {
	logger := loggerutils.GetLogger(ctx)
	blockOutput := ThinkingBlockOutput{}

	for i := range iterations {
		logger.Debug("Thinking block: iteration", "number", i)
		currentIterationAnswer := PartialAnswer{}
		currentIterationPrompts := Prompts{}

		var wPrompt string
		if i == 0 {
			// 1a. If it is a first iteration, ask worker to provide solution
			wPrompt = fmt.Sprintf("%s\nTASK: %s\n", initialWorkerPrompt, taskDescription)
		} else {
			// 1. Else ask worker to refine solution
			wPrompt = fmt.Sprintf(
				"%s\nTASK: %s\nSOLUTION: %s\nSUMMARY: %s",
				workerPromptWithSummary,
				taskDescription,
				blockOutput.PartAnswers[i-1].WorkerSolution,
				blockOutput.PartAnswers[i-1].OracleSummary,
			)
		}

		// 2. Get worker solution proposal
		currentIterationPrompts.WorkerPrompt = wPrompt
		solution, err := tb.Worker.Chat(ctx, wPrompt)
		if err != nil {
			return ThinkingBlockOutput{}, fmt.Errorf("error chatting with worker %w", err)
		}
		currentIterationAnswer.WorkerSolution = solution

		// 3. Ask experts to review the proposal
		eP := fmt.Sprintf("%s\nTASK: %s\nSOLUTION %s", expertPrompt, taskDescription, solution)
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
		oP := fmt.Sprintf("%s\nSOLUTION: %s\nREVIEWS: %s", oraclePrompt, solution, reviews)
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
