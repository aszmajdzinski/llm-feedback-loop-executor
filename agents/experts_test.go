package agents

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpertsTeam_Ask(t *testing.T) {
	mockLlm1 := &MockLlmProvider{response: "mock response 1", err: nil}
	mockLlm2 := &MockLlmProvider{response: "mock response 2", err: nil}
	agents := []Agent{
		{Name: "agent1", Llm: mockLlm1, SystemPrompt: "prompt1", Model: "model1"},
		{Name: "agent2", Llm: mockLlm2, SystemPrompt: "prompt2", Model: "model2"},
	}
	team := ExpertsTeam{Experts: agents}

	t.Run("successful ask", func(t *testing.T) {
		answers := team.Ask(context.Background(), "hello")
		assert.NoError(t, answers[0].Error)
		assert.NoError(t, answers[1].Error)
		assert.Equal(t, "mock response 1", answers[0].Answer)
		assert.Equal(t, "mock response 2", answers[1].Answer)
	})
}

func TestExpertsTeam_Error(t *testing.T) {
	err := errors.New("provider error")
	mockLlm1 := &MockLlmProvider{response: "mock response 1", err: nil}
	mockLlm2 := &MockLlmProvider{response: "", err: err}
	agents := []Agent{
		{Name: "agent1", Llm: mockLlm1, SystemPrompt: "prompt1", Model: "model1"},
		{Name: "agent2", Llm: mockLlm2, SystemPrompt: "prompt2", Model: "model2"},
	}
	team := ExpertsTeam{Experts: agents}

	t.Run("ask with error", func(t *testing.T) {
		answers := team.Ask(context.Background(), "error")
		assert.True(t, (errors.Is(answers[0].Error, err) && answers[1].Error == nil ||
			answers[0].Error == nil && errors.Is(answers[1].Error, err)))
		assert.True(t, answers[0].Answer == "mock response 1" && answers[1].Answer == "" ||
			answers[0].Answer == "" && answers[1].Answer == "mock response 1")
	})
}
