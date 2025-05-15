package thinkingblock

import (
	"context"
	"testing"

	"example.com/web-app-creator/assistants"
	"example.com/web-app-creator/llm"
)

func TestThinkingBlock_Run(t *testing.T) {
	// Mock Worker Assistant
	mockWorker := &llm.MockLLMProvider{
		GetCompletionFunc: func(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
			return llm.ChatResponse{Response: "Worker solution"}, nil
		},
	}

	// Mock ExpertsTeam
	mockExpertsTeam := assistants.MockExpertsTeam{
		AskFunc: func(ctx context.Context, prompt string) []assistants.ExpertAnswer {
			return []assistants.ExpertAnswer{
				{Answer: "Expert review 1", Error: nil},
				{Answer: "Expert review 2", Error: nil},
			}
		},
	}

	// Mock Oracle Assistant
	mockOracle := &llm.MockLLMProvider{
		GetCompletionFunc: func(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
			if req.BaseChatRequest.Messages[0].Content == "OK" {
				return llm.ChatResponse{Response: "OK"}, nil
			}
			return llm.ChatResponse{Response: "Oracle summary"}, nil
		},
	}

	// Create ThinkingBlock with mocks
	tb := ThinkingBlock{
		Worker: assistants.Assistant{
			Name:         "WorkerAssistant",
			SystemPrompt: "Worker prompt",
			Llm:          mockWorker,
		},
		ExpertsTeam: &mockExpertsTeam,
		Oracle: assistants.Assistant{
			Name:         "OracleAssistant",
			SystemPrompt: "Oracle prompt",
			Llm:          mockOracle,
		},
	}

	// Run the ThinkingBlock
	output, err := tb.Run(context.Background(), "Test task", "Test data", false, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Validate the output
	if len(output.PartAnswers) == 0 {
		t.Fatalf("expected at least one partial answer, got none")
	}

	if output.FinalAnswer != "Worker solution" {
		t.Errorf("expected final answer to be 'Worker solution', got '%s'", output.FinalAnswer)
	}

	if output.PartAnswers[0].WorkerSolution != "Worker solution" {
		t.Errorf("expected worker solution to be 'Worker solution', got '%s'", output.PartAnswers[0].WorkerSolution)
	}

	if len(output.PartAnswers[0].ExpertAnswers) != 2 {
		t.Errorf("expected 2 expert answers, got %d", len(output.PartAnswers[0].ExpertAnswers))
	}

	if output.PartAnswers[0].OracleSummary != "Oracle summary" {
		t.Errorf("expected oracle summary to be 'Oracle summary', got '%s'", output.PartAnswers[0].OracleSummary)
	}
}
