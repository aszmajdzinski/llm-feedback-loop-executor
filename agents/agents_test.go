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

func (m *MockLlmProvider) GetCompletion(
	_ context.Context,
	_ llm.ChatRequest,
) (llm.ChatResponse, error) {
	return llm.ChatResponse{Content: m.response}, m.err
}

func TestAgent_ChatOk(t *testing.T) {
	mockLlm := &MockLlmProvider{response: "mock response", err: nil}
	agent := Agent{
		Name:         "test-agent",
		Llm:          mockLlm,
		SystemPrompt: "test system prompt",
		Model:        "test-model",
	}

	t.Run("successful chat", func(t *testing.T) {
		response, err := agent.Chat(context.Background(), "prompt")
		require.NoError(t, err)
		assert.Equal(t, "mock response", response)
	})
}

func TestAgent_ChatError(t *testing.T) {
	mockLlm := &MockLlmProvider{response: "", err: errors.New("provider error")}
	agent := Agent{
		Name:         "test-agent",
		Llm:          mockLlm,
		SystemPrompt: "test system prompt",
		Model:        "test-model",
	}

	t.Run("chat with error", func(t *testing.T) {
		response, err := agent.Chat(context.Background(), "prompt")
		require.ErrorContains(t, err, "provider error")
		assert.Equal(t, "", response)
	})
}
