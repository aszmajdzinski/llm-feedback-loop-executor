package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type OpenAIProvider struct {
	apiKey     string
	baseURL    string
	httpClient HTTPClient
}

func (o *OpenAIProvider) GetCompletion(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	startTime := time.Now()

	requestBody, err := json.Marshal(req)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("error marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx,
		"POST",
		o.baseURL+"/chat/completions",
		bytes.NewReader(requestBody),
	)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("error creating request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)

	resp, err := o.httpClient.Do(httpReq)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return ChatResponse{}, fmt.Errorf(
			"non-200 status code: %d; body: %s",
			resp.StatusCode,
			string(body),
		)
	}

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage TokenUsage `json:"usage"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return ChatResponse{}, fmt.Errorf("error parsing response: %w", err)
	}

	response := ChatResponse{
		Content:    result.Choices[0].Message.Content,
		TokenUsage: result.Usage,
		TimeTaken:  time.Since(startTime),
	}

	return response, nil
}

func NewOpenAIProvider(apiKey, baseURL string) *OpenAIProvider {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	return &OpenAIProvider{
		apiKey:     apiKey,
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}
