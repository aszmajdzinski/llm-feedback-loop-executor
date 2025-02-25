package llm

import (
	"context"
	"time"
)

type LlmProvider interface {
	GetCompletion(context.Context, ChatRequest) (ChatResponse, error)
}

type ChatRequest struct {
	Model     string        `json:"model"`
	Messages  []ChatMessage `json:"messages"`
	MaxTokens int           `json:"max_tokens,omitempty"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatResponse struct {
	Content    string
	TokenUsage TokenUsage
	TimeTaken  time.Duration
}

type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
