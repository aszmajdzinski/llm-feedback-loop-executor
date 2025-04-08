package agents

import (
	"context"

	"example.com/web-app-creator/llm"
)

type Agent struct {
	Name         string
	SystemPrompt string
	Llm          llm.LlmProvider
}

func (a Agent) Chat(ctx context.Context, msg string) (string, error) {
	s := llm.ChatMessage{Role: "developer", Message: a.SystemPrompt}
	m := llm.ChatMessage{Role: "user", Message: msg}

	ans, err := a.Llm.GetCompletion(
		ctx,
		llm.ChatRequest{Messages: []llm.ChatMessage{s, m}},
	)
	if err != nil {
		return "", err
	}

	return ans.Response, nil
}
