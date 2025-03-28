package agents

import (
	"context"

	"example.com/web-app-creator/llm"
)

type Agent struct {
	Name         string
	SystemPrompt string
	Model        string
	Llm          llm.LlmProvider
}

func (a Agent) Chat(ctx context.Context, msg string) (string, error) {
	s := llm.ChatMessage{Role: "developer", Content: a.SystemPrompt}
	m := llm.ChatMessage{Role: "user", Content: msg}

	ans, err := a.Llm.GetCompletion(
		ctx,
		llm.ChatRequest{Model: a.Model, Messages: []llm.ChatMessage{s, m}},
	)
	if err != nil {
		return "", err
	}

	return ans.Content, nil
}
