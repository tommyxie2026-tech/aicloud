package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type SecretResolver interface {
	ResolveSecret(ctx context.Context, secretRef string) (string, error)
}

type HTTPClient struct {
	config      Config
	doer        HTTPDoer
	resolver    SecretResolver
	retryPolicy RetryPolicy
}

func NewHTTPClient(config Config, doer HTTPDoer, resolver SecretResolver) (*HTTPClient, error) {
	if err := ValidateConfig(config); err != nil {
		return nil, err
	}
	if doer == nil {
		doer = &http.Client{Timeout: time.Duration(config.TimeoutSeconds) * time.Second}
	}
	if resolver == nil {
		return nil, NewHTTPClientError("MissingSecretResolver", "secret resolver is required")
	}
	return &HTTPClient{config: config, doer: doer, resolver: resolver, retryPolicy: NewRetryPolicy(config.MaxRetries)}, nil
}

func (c *HTTPClient) Health(ctx context.Context) error {
	return nil
}

func (c *HTTPClient) Generate(ctx context.Context, req CompatibleRequest) (*CompatibleResponse, error) {
	body, err := BuildChatCompletionRequest(req)
	if err != nil {
		return nil, err
	}
	url, err := BuildChatCompletionsURL(c.config.Endpoint)
	if err != nil {
		return nil, err
	}
	apiKey, err := c.resolver.ResolveSecret(ctx, c.config.SecretRef)
	if err != nil {
		return nil, NewHTTPClientError("SecretResolveFailed", err.Error())
	}
	if apiKey == "" {
		return nil, NewHTTPClientError("EmptySecret", "resolved secret is empty")
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, NewHTTPClientError("RequestMarshalFailed", err.Error())
	}

	var lastErr error
	for attempt := 0; ; attempt++ {
		response, statusCode, err := c.doOnce(ctx, url, apiKey, payload)
		if err == nil {
			return response, nil
		}
		lastErr = err
		decision := c.retryPolicy.ShouldRetry(attempt, statusCode, err)
		if !decision.Retry {
			return nil, lastErr
		}
	}
}

func (c *HTTPClient) doOnce(ctx context.Context, url string, apiKey string, payload []byte) (*CompatibleResponse, int, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, 0, NewHTTPClientError("RequestBuildFailed", err.Error())
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := c.doer.Do(httpReq)
	if err != nil {
		return nil, 0, NewHTTPClientError("RequestFailed", err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return nil, resp.StatusCode, NewHTTPClientError("Non2xxResponse", fmt.Sprintf("status=%d body=%s", resp.StatusCode, string(data)))
	}

	var parsed chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, resp.StatusCode, NewHTTPClientError("ResponseDecodeFailed", err.Error())
	}
	if len(parsed.Choices) == 0 {
		return nil, resp.StatusCode, NewHTTPClientError("MissingChoices", "response choices are empty")
	}
	return &CompatibleResponse{
		OutputText:   parsed.Choices[0].Message.Content,
		FinishReason: parsed.Choices[0].FinishReason,
		InputTokens:  parsed.Usage.PromptTokens,
		OutputTokens: parsed.Usage.CompletionTokens,
	}, resp.StatusCode, nil
}

type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

type HTTPClientError struct {
	Code    string
	Message string
}

func NewHTTPClientError(code string, message string) *HTTPClientError {
	return &HTTPClientError{Code: code, Message: message}
}

func (e *HTTPClientError) Error() string {
	return e.Code + ": " + e.Message
}
