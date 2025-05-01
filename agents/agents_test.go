package agents

import (
	"context"
	"testing"

	"example.com/web-app-creator/llm"
)

func TestAgentChat(t *testing.T) {
	mockLLM := &llm.MockLLMProvider{
		GetCompletionFunc: func(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
			return llm.ChatResponse{Response: "Mocked response"}, nil
		},
	}

	agent := Agent{
		Name:         "TestAgent",
		SystemPrompt: "You are a helpful assistant.",
		Llm:          mockLLM,
	}

	resp, err := agent.Chat(context.Background(), "Hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp != "Mocked response" {
		t.Errorf("expected 'Mocked response', got '%s'", resp)
	}
}

func TestAgentStructuredChat(t *testing.T) {
	mockStructuredLLM := &llm.MockStructuredLLMProvider{
		GetResponseFunc: func(ctx context.Context, req llm.StructuredChatRequest) (llm.ChatResponse, error) {
			return llm.ChatResponse{Response: "Mocked structured response"}, nil
		},
	}

	agent := Agent{
		Name:         "TestAgent",
		SystemPrompt: "You are a helpful assistant.",
		Llm:          mockStructuredLLM,
	}

	resp, err := agent.StructuredChat(context.Background(), "Hello", "TestName", map[string]any{"key": "value"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp != "Mocked structured response" {
		t.Errorf("expected 'Mocked structured response', got '%s'", resp)
	}
}
