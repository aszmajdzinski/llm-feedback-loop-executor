package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	loggerutils "github.com/aszmajdzinski/llm-feedback-loop-executor/logger_utils"
)

const (
	defaultBaseURL = "https://api.openai.com/v1"
	defaultTimeout = 60 * time.Second
	maxRetries     = 3
	retryDelay     = 2 * time.Second
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
	openAIReq := newOpenAIRequest(req, o.model)
	return o.executeRequest(ctx, "/chat/completions", openAIReq, func(body []byte) (ChatResponse, error) {
		var result openAIResponse
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
			TimeTaken: 0, // Time taken is handled in executeRequest
		}
		return response, nil
	})
}

func NewOpenAIProvider(apiKey, model string, baseURL string) *OpenAIProvider {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	return &OpenAIProvider{
		apiKey:     apiKey,
		baseURL:    baseURL,
		model:      model,
		httpClient: &http.Client{Timeout: defaultTimeout},
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

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage openAITokenUsage `json:"usage"`
}

func newOpenAIRequest(chat ChatRequest, model string) openAIChatRequest {
	messages := make([]openAIChatMessage, len(chat.Messages))
	for i, msg := range chat.Messages {
		messages[i] = openAIChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	return openAIChatRequest{
		Messages:  messages,
		Model:     model,
		MaxTokens: chat.MaxTokens,
	}
}

type OpenAIProviderWithStructuredOutput struct {
	*OpenAIProvider
}

func (o *OpenAIProviderWithStructuredOutput) GetResponse(ctx context.Context, req StructuredChatRequest) (ChatResponse, error) {
	openAIReq := newOpenAIWithStructuredOutputProviderRequest(req, o.model)

	return o.executeRequest(ctx, "/responses", openAIReq, func(body []byte) (ChatResponse, error) {
		var result structuredOpenAIResponse
		if err := json.Unmarshal(body, &result); err != nil {
			return ChatResponse{}, fmt.Errorf("error parsing response: %w", err)
		}

		response := ChatResponse{
			Response: result.Output[0].Content[0].Text,
			TokenUsage: TokenUsage{
				InputTokens:  result.Usage.PromptTokens,
				OutputTokens: result.Usage.CompletionTokens,
				TotalTokens:  result.Usage.TotalTokens,
			},
			TimeTaken: 0, // Time taken is handled in executeRequest
		}
		return response, nil
	})
}

func NewOpenAIWithStructuredOutputProvider(apiKey, model string, baseURL string) *OpenAIProviderWithStructuredOutput {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	return &OpenAIProviderWithStructuredOutput{
		&OpenAIProvider{
			apiKey:     apiKey,
			baseURL:    baseURL,
			model:      model,
			httpClient: &http.Client{Timeout: defaultTimeout},
		}}
}

type openAIWithStructuredOutputProviderChatRequest struct {
	Model     string                                          `json:"model"`
	MaxTokens int                                             `json:"max_output_tokens,omitempty"`
	Input     []openAIWithStructuredOutputProviderChatMessage `json:"input"`
	Text      TextFormat                                      `json:"text"`
}

type openAIWithStructuredOutputProviderChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type TextFormat struct {
	Format FormatDetail `json:"format"`
}

type FormatDetail struct {
	Type   string `json:"type"`
	Name   string `json:"name"`
	Schema any    `json:"schema"`
	Strict bool   `json:"strict"`
}

type openAIWithStructuredOutputProviderTokenUsage struct {
	PromptTokens     int `json:"input_tokens"`
	CompletionTokens int `json:"output_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type structuredOpenAIResponse struct {
	Output []struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
	Usage openAIWithStructuredOutputProviderTokenUsage `json:"usage"`
}

func newOpenAIWithStructuredOutputProviderRequest(chat StructuredChatRequest, model string) openAIWithStructuredOutputProviderChatRequest {
	messages := make([]openAIWithStructuredOutputProviderChatMessage, len(chat.Messages))
	for i, msg := range chat.Messages {
		messages[i] = openAIWithStructuredOutputProviderChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	return openAIWithStructuredOutputProviderChatRequest{
		Model:     model,
		MaxTokens: chat.MaxTokens,
		Input:     messages,
		Text: TextFormat{
			Format: FormatDetail{
				Type:   "json_schema",
				Name:   chat.Name,
				Schema: chat.Schema,
				Strict: true,
			},
		},
	}
}

type responseParser func([]byte) (ChatResponse, error)

func (o *OpenAIProvider) executeRequest(ctx context.Context, endpoint string, requestBodyData any, parseResponse responseParser) (ChatResponse, error) {
	logger := loggerutils.GetLogger(ctx)
	startTime := time.Now()

	requestBody, err := json.Marshal(requestBodyData)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("error marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx,
		"POST",
		o.baseURL+endpoint,
		bytes.NewReader(requestBody),
	)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("error creating request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)

	var resp *http.Response
	for i := range maxRetries {
		resp, err = o.httpClient.Do(httpReq)
		if err == nil {
			break
		}
		logger.Warn("Request failed, retrying...", "attempt", i+1, "error", err)
		time.Sleep(retryDelay)
	}
	if err != nil {
		return ChatResponse{}, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return ChatResponse{}, fmt.Errorf("error reading response body: %w", err)
		}
		return ChatResponse{}, fmt.Errorf(
			"non-200 status code: %d; body: %s",
			resp.StatusCode,
			string(body),
		)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("error reading response body: %w", err)
	}

	chatResponse, err := parseResponse(body)
	if err != nil {
		return ChatResponse{}, err
	}

	chatResponse.TimeTaken = time.Since(startTime)
	return chatResponse, nil
}
