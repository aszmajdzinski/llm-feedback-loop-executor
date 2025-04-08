package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	loggerutils "example.com/web-app-creator/logger_utils"
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type OpenAIProvider struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient HTTPClient
}

func (o *OpenAIProvider) GetCompletion(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	logger := loggerutils.GetLogger(ctx)
	startTime := time.Now()

	requestBody, err := json.Marshal(newOpenAIRequest(req, o.model))
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

	var resp *http.Response
	retryCount := 3
	for i := range retryCount {
		resp, err = o.httpClient.Do(httpReq)
		if err == nil {
			break
		}
		logger.Warn("Request failed, retrying...", "attempt", i+1, "error", err)
		time.Sleep(2 * time.Second)
	}
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
		Usage openAITokenUsage `json:"usage"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return ChatResponse{}, fmt.Errorf("error parsing response: %w", err)
	}

	response := ChatResponse{
		Response: result.Choices[0].Message.Content,
		TokenUsage: TokenUsage{
			InputTokens:  result.Usage.PromptTokens,
			OutputTokens: result.Usage.CompletionTokens,
			TotalTokens:  result.Usage.TotalTokens,
		},
		TimeTaken: time.Since(startTime),
	}

	return response, nil
}

func NewOpenAIProvider(apiKey, model string, baseURL string) *OpenAIProvider {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	return &OpenAIProvider{
		apiKey:     apiKey,
		baseURL:    baseURL,
		model:      model,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

type openAIChatRequest struct {
	Messages  []openAIChatMessage `json:"messages"`
	Model     string              `json:"model"`
	MaxTokens int                 `json:"max_tokens,omitempty"`
}

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAITokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

func newOpenAIRequest(chat ChatRequest, model string) openAIChatRequest {
	messages := make([]openAIChatMessage, len(chat.Messages))
	for i, msg := range chat.Messages {
		messages[i] = openAIChatMessage{
			Role:    msg.Role,
			Content: msg.Message,
		}
	}

	return openAIChatRequest{
		Messages:  messages,
		Model:     model,
		MaxTokens: chat.MaxTokens,
	}
}
