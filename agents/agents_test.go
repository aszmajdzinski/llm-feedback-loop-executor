package agents

import (
	"context"
	"errors"
	"testing"

	"example.com/web-app-creator/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockLlmProvider struct {
	response string
	err      error
}

func (m *MockLlmProvider) GetCompletion(_ context.Context, _ llm.ChatRequest) (llm.ChatResponse, error) {
	return llm.ChatResponse{Content: m.response}, m.err
}

func TestAgent_ChatOk(t *testing.T) {
	mockLlm := &MockLlmProvider{response: "mock response", err: nil}
	agent := Agent{Name: "test-agent", Llm: mockLlm, SystemPrompt: "test system prompt", Model: "test-model"}

	t.Run("successful chat", func(t *testing.T) {
		response, err := agent.Chat(context.Background(), "prompt")
		require.NoError(t, err)
		assert.Equal(t, "mock response", response)
	})
}

func TestAgent_ChatError(t *testing.T) {
	mockLlm := &MockLlmProvider{response: "", err: errors.New("provider error")}
	agent := Agent{Name: "test-agent", Llm: mockLlm, SystemPrompt: "test system prompt", Model: "test-model"}

	t.Run("chat with error", func(t *testing.T) {
		response, err := agent.Chat(context.Background(), "prompt")
		require.ErrorContains(t, err, "provider error")
		assert.Equal(t, "", response)
	})
}

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
