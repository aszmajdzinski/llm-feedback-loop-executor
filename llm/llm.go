package llm

import (
	"context"
	"time"
)

type LlmProvider interface {
	GetCompletion(context.Context, ChatRequest) (ChatResponse, error)
}

type StructuredOutputCapableLlmProvider interface {
	LlmProvider
	GetResponse(context.Context, ChatRequest) (ChatResponse, error)
}

type ChatRequest struct {
	Messages   []ChatMessage
	JSONSchema string
	MaxTokens  int
}

type ChatMessage struct {
	Role    string
	Message string
}

type ChatResponse struct {
	Response   string
	TokenUsage TokenUsage
	TimeTaken  time.Duration
}

type TokenUsage struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
}
