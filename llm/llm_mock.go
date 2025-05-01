package llm

import (
	"context"
)

// MockLLMProvider is a mock implementation of the LLMProvider interface.
type MockLLMProvider struct {
	GetCompletionFunc func(ctx context.Context, req ChatRequest) (ChatResponse, error)
}

func (m *MockLLMProvider) GetCompletion(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	if m.GetCompletionFunc != nil {
		return m.GetCompletionFunc(ctx, req)
	}
	return ChatResponse{}, nil
}

// MockStructuredLLMProvider is a mock implementation of the StructuredLLMProvider interface.
type MockStructuredLLMProvider struct {
	MockLLMProvider
	GetResponseFunc func(ctx context.Context, req StructuredChatRequest) (ChatResponse, error)
}

func (m *MockStructuredLLMProvider) GetResponse(ctx context.Context, req StructuredChatRequest) (ChatResponse, error) {
	if m.GetResponseFunc != nil {
		return m.GetResponseFunc(ctx, req)
	}
	return ChatResponse{}, nil
}
