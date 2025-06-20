package assistants

import (
	"context"
	"errors"

	"github.com/aszmajdzinski/llm-feedback-loop-executor/llm"
)

type Assistant struct {
	Name         string
	SystemPrompt string
	Llm          llm.LLMProvider
}

func (a Assistant) Chat(ctx context.Context, msg string) (string, error) {
	s := llm.ChatMessage{Role: "developer", Content: a.SystemPrompt}
	m := llm.ChatMessage{Role: "user", Content: msg}

	ans, err := a.Llm.GetCompletion(
		ctx,
		llm.ChatRequest{BaseChatRequest: llm.BaseChatRequest{
			Messages: []llm.ChatMessage{s, m}},
		},
	)
	if err != nil {
		return "", err
	}

	return ans.Response, nil
}

func (a Assistant) StructuredChat(ctx context.Context, msg string, name string, schema map[string]any) (string, error) {
	l, ok := a.Llm.(llm.StructuredLLMProvider)
	if !ok {
		return "", errors.New("selected model does not support structured responses")
	}

	s := llm.ChatMessage{Role: "developer", Content: a.SystemPrompt}
	m := llm.ChatMessage{Role: "user", Content: msg}

	ans, err := l.GetResponse(
		ctx,
		llm.StructuredChatRequest{BaseChatRequest: llm.BaseChatRequest{
			Messages: []llm.ChatMessage{s, m}},
			Schema: schema,
			Name:   name,
		},
	)
	if err != nil {
		return "", err
	}

	return ans.Response, nil
}
