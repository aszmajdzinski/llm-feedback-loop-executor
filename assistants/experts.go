package assistants

import (
	"context"
	"fmt"
	"sync"
)

type ExpertsTeamInterface interface {
	Ask(ctx context.Context, prompt string) []ExpertAnswer
}

type ExpertsTeam struct {
	Experts []Assistant
}

type ExpertAnswer struct {
	Answer string
	Error  error
}

func (et *ExpertsTeam) Ask(ctx context.Context, prompt string) []ExpertAnswer {
	type result struct {
		index  int
		answer string
		error  error
	}

	ch := make(chan result, len(et.Experts))
	var wg sync.WaitGroup

	for i, a := range et.Experts {
		wg.Add(1)

		go func(index int, assistant Assistant) {
			defer wg.Done()
			ans, err := assistant.Chat(ctx, prompt)
			if err != nil {
				ch <- result{
					index: index,
					error: fmt.Errorf("cannot get response from chat %s: %w", assistant.Name, err),
				}
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
