package agents

import (
	"fmt"
	"sync"

	"example.com/web-app-creator/llm"
)

type Agent struct {
	name         string
	llm          llm.LlmProvider
	systemPrompt string
}

func (a Agent) Chat(msg string) (string, error) {
	ans, err := a.llm.GetCompletion(msg)
	if err != nil {
		return "", err
	}
	return ans, nil
}

type ExpertsTeam struct {
	Experts []Agent
}

type ExpertAnswer struct {
	Answer string
	Error  error
}

func (e *ExpertsTeam) Ask(msg string) []ExpertAnswer {
	type result struct {
		index  int
		answer string
		error  error
	}

	ch := make(chan result, len(e.Experts))
	var wg sync.WaitGroup

	for i, a := range e.Experts {
		wg.Add(1)

		go func(index int, agent Agent) {
			defer wg.Done()
			ans, err := agent.Chat(msg)
			if err != nil {
				ch <- result{index: index, error: fmt.Errorf("cannot get response from agent %s: %w", agent.name, err)}
				return
			}

			ch <- result{index: index, answer: ans}
		}(i, a)
	}

	wg.Wait()
	close(ch)

	answers := make([]ExpertAnswer, len(e.Experts))

	for res := range ch {
		answers[res.index] = ExpertAnswer{
			Answer: res.answer,
			Error:  res.error,
		}
	}

	return answers
}
