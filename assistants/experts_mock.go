package assistants

import "context"

type MockExpertsTeam struct {
	AskFunc func(ctx context.Context, prompt string) []ExpertAnswer
}

func (m MockExpertsTeam) Ask(ctx context.Context, prompt string) []ExpertAnswer {
	if m.AskFunc != nil {
		return m.AskFunc(ctx, prompt)
	}
	return nil
}
