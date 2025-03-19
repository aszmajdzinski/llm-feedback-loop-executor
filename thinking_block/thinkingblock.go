package thinkingblock

import (
	"context"
	"fmt"

	"example.com/web-app-creator/agents"
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
	"Solution will start with <SOLUTION number>."

const workerPromptWithSummary string = "You will be given a TASK, a SOLUTION and a SUMMARY of feedback from experts. " +
	"Your job is to refine the SOLUTION based on the feedback provided. " +
	"Ensure that the final solution is accurate, complete, and incorporates all the improvements suggested by the experts."

type ThinkingBlockAnswer struct {
	PartAnswers []PartAnswer
	FinalAnswer string
}

type PartAnswer struct {
	WorkerSolution string
	ExpertAnswers  []string
	OracleSummary  string
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
) (ThinkingBlockAnswer, error) {
	blockAnswer := ThinkingBlockAnswer{}

	for i := range iterations {
		currentIterationAnswer := PartAnswer{}

		if i == 0 {
			// 1a. If it is a first iteration, ask worker to provide solution
			taskDescription = fmt.Sprintf("%s TASK: %s", initialWorkerPrompt, taskDescription)
		} else {
			// 1. Else ask worker to refine solution
			taskDescription = fmt.Sprintf(
				"%s TASK: %s SOLUTION: %s SUMMARY: %s",
				workerPromptWithSummary,
				taskDescription,
				blockAnswer.PartAnswers[i-1].WorkerSolution,
				blockAnswer.PartAnswers[i-1].OracleSummary,
			)
		}

		// 2. Get worker solution proposal
		solution, err := tb.Worker.Chat(ctx, taskDescription)
		if err != nil {
			return ThinkingBlockAnswer{}, fmt.Errorf("error chatting with worker %w", err)
		}
		currentIterationAnswer.WorkerSolution = solution

		// 3. Ask experts to review the proposal
		expertsAnswers := tb.ExpertsTeam.Ask(
			ctx,
			fmt.Sprintf("%s TASK: %s SOLUTION %s", expertPrompt, taskDescription, solution),
		)

		var reviews string
		for i, ea := range expertsAnswers {
			if ea.Error != nil {
				return ThinkingBlockAnswer{}, fmt.Errorf("error chatting with expert %w", err)
			}

			reviews += fmt.Sprintf("<SOLUTION %d> %s \n", i, ea.Answer)
			currentIterationAnswer.ExpertAnswers = append(
				currentIterationAnswer.ExpertAnswers,
				ea.Answer,
			)
		}

		// 4. Provide those reviews to Oracle to sum up
		summary, err := tb.Oracle.Chat(
			ctx,
			fmt.Sprintf("%s SOLUTION: %s REVIEWS: %s", oraclePrompt, solution, reviews),
		)
		if err != nil {
			return ThinkingBlockAnswer{}, fmt.Errorf("error chatting with oracle %w", err)
		}

		currentIterationAnswer.OracleSummary = summary

		if summary == "OK" {
			break
		}

		blockAnswer.PartAnswers = append(blockAnswer.PartAnswers, currentIterationAnswer)
	}
	blockAnswer.FinalAnswer = blockAnswer.PartAnswers[len(blockAnswer.PartAnswers)-1].WorkerSolution

	return blockAnswer, nil
}
