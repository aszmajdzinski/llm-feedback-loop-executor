package llm

import (
	"context"
	"time"
)

type LLMProvider interface {
	GetCompletion(context.Context, ChatRequest) (ChatResponse, error)
}

type StructuredLLMProvider interface {
	LLMProvider
	GetResponse(context.Context, StructuredChatRequest) (ChatResponse, error)
}

type BaseChatRequest struct {
	Messages  []ChatMessage
	MaxTokens int
}

type ChatRequest struct {
	BaseChatRequest
}

type StructuredChatRequest struct {
	BaseChatRequest
	Schema any
	Name   string
}

type ChatMessage struct {
	Role    string
	Content string
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
