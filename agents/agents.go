package agents

import (
	"context"
	"fmt"
	"sync"

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

	ans, err := a.Llm.GetCompletion(ctx, llm.ChatRequest{Model: a.Model, Messages: []llm.ChatMessage{s, m}})
	if err != nil {
		return "", err
	}
	// TODO: add logger

	return ans.Content, nil
}

type ExpertsTeam struct {
	Experts []Agent
}

type ExpertAnswer struct {
	Answer string
	Error  error
}

func (et *ExpertsTeam) Ask(ctx context.Context, msg string) []ExpertAnswer {
	type result struct {
		index  int
		answer string
		error  error
	}

	ch := make(chan result, len(et.Experts))
	var wg sync.WaitGroup

	for i, a := range et.Experts {
		wg.Add(1)

		go func(index int, agent Agent) {
			defer wg.Done()
			ans, err := agent.Chat(ctx, msg)
			if err != nil {
				ch <- result{index: index, error: fmt.Errorf("cannot get response from agent %s: %w", agent.Name, err)}
				return
			}

			ch <- result{index: index, answer: ans}
		}(i, a)
	}

	wg.Wait()
	close(ch)

	answers := make([]ExpertAnswer, len(et.Experts))

	for res := range ch {
		answers[res.index] = ExpertAnswer{
			Answer: res.answer,
			Error:  res.error,
		}
	}

	return answers
}
