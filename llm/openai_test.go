package llm

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

func TestGetCompletion_Success(t *testing.T) {
	mockResponse := `{
		"choices": [{
			"message": {
				"content": "Test response"
			}
		}],
		"usage": {
			"prompt_tokens": 5,
			"completion_tokens": 7,
			"total_tokens": 12
		}
	}`

	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader([]byte(mockResponse))),
			}, nil
		},
	}

	provider := &OpenAIProvider{
		apiKey:     "test-api-key",
		baseURL:    "https://api.openai.com/v1",
		httpClient: mockClient,
	}

	req := ChatRequest{
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	ctx := context.Background()
	resp, err := provider.GetCompletion(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, "Test response", resp.Content)
	assert.Equal(t, 5, resp.TokenUsage.PromptTokens)
	assert.Equal(t, 7, resp.TokenUsage.CompletionTokens)
	assert.Equal(t, 12, resp.TokenUsage.TotalTokens)
}

func TestGetCompletion_Error(t *testing.T) {
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("network error")
		},
	}

	provider := &OpenAIProvider{
		apiKey:     "test-api-key",
		baseURL:    "https://api.openai.com/v1",
		httpClient: mockClient,
	}

	req := ChatRequest{
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	ctx := context.Background()
	_, err := provider.GetCompletion(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, "error sending request: network error", err.Error())
}

func TestGetCompletion_Non200StatusCode(t *testing.T) {
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Body:       io.NopCloser(bytes.NewReader([]byte("Bad Request"))),
			}, nil
		},
	}

	provider := &OpenAIProvider{
		apiKey:     "test-api-key",
		baseURL:    "https://api.openai.com/v1",
		httpClient: mockClient,
	}

	req := ChatRequest{
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	ctx := context.Background()
	_, err := provider.GetCompletion(ctx, req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "non-200 status code: 400; body: Bad Request")
}

func TestNewOpenAIProvider_DefaultBaseURL(t *testing.T) {
	provider := NewOpenAIProvider("test-api-key", "")

	assert.Equal(t, "https://api.openai.com/v1", provider.baseURL)
	assert.Equal(t, "test-api-key", provider.apiKey)
	assert.NotNil(t, provider.httpClient)
}

func TestNewOpenAIProvider_CustomBaseURL(t *testing.T) {
	provider := NewOpenAIProvider("test-api-key", "https://custom-url.com")

	assert.Equal(t, "https://custom-url.com", provider.baseURL)
	assert.Equal(t, "test-api-key", provider.apiKey)
	assert.NotNil(t, provider.httpClient)
}
