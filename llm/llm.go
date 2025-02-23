package llm

type LlmProvider interface {
	GetCompletion(msg string) (string, error)
}

type DummyLlmProvider struct{}

func (DummyLlmProvider) GetCompletion(_ string) (string, error) {
	return "This is a dummy response", nil
}
