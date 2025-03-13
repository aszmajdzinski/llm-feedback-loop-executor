package agents

import (
	"context"
	"fmt"
	"sync"
)

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
