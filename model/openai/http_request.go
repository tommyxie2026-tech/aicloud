package openai

import "strings"

const defaultChatCompletionsPath = "/chat/completions"

// ChatCompletionRequest is the OpenAI-compatible request body used by many hosted,
// private, and self-hosted model servers.
type ChatCompletionRequest struct {
	Model          string                  `json:"model"`
	Messages       []ChatCompletionMessage `json:"messages"`
	Temperature    float32                 `json:"temperature,omitempty"`
	MaxTokens      int                     `json:"max_tokens,omitempty"`
	Stream         bool                    `json:"stream"`
	ResponseFormat ResponseFormat          `json:"response_format,omitempty"`
}

type ChatCompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ResponseFormat struct {
	Type string `json:"type,omitempty"`
}

// BuildChatCompletionRequest converts an internal CompatibleRequest into an
// OpenAI-compatible chat completions request body.
//
// It does not send HTTP requests and does not attach credentials.
func BuildChatCompletionRequest(req CompatibleRequest) (ChatCompletionRequest, error) {
	if strings.TrimSpace(req.Model) == "" {
		return ChatCompletionRequest{}, NewHTTPBuildError("MissingModel", "model is required")
	}
	if strings.TrimSpace(req.Instruction) == "" {
		return ChatCompletionRequest{}, NewHTTPBuildError("MissingInstruction", "instruction is required")
	}

	messages := make([]ChatCompletionMessage, 0, 2)
	if strings.TrimSpace(req.SystemPrompt) != "" {
		messages = append(messages, ChatCompletionMessage{Role: "system", Content: strings.TrimSpace(req.SystemPrompt)})
	}
	messages = append(messages, ChatCompletionMessage{Role: "user", Content: buildUserMessage(req)})

	body := ChatCompletionRequest{
		Model:       strings.TrimSpace(req.Model),
		Messages:    messages,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxOutputTokens,
		Stream:      false,
	}
	if strings.TrimSpace(req.OutputSchema) != "" {
		body.ResponseFormat = ResponseFormat{Type: "json_object"}
	}
	return body, nil
}

func BuildChatCompletionsURL(endpoint string) (string, error) {
	endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if endpoint == "" {
		return "", NewHTTPBuildError("MissingEndpoint", "endpoint is required")
	}
	return endpoint + defaultChatCompletionsPath, nil
}

func buildUserMessage(req CompatibleRequest) string {
	parts := []string{"instruction:\n" + strings.TrimSpace(req.Instruction)}
	if strings.TrimSpace(req.ContextText) != "" {
		parts = append(parts, "context:\n"+strings.TrimSpace(req.ContextText))
	}
	if strings.TrimSpace(req.OutputSchema) != "" {
		parts = append(parts, "outputSchema:\n"+strings.TrimSpace(req.OutputSchema))
	}
	parts = append(parts, "Return only raw JSON. Do not wrap output in markdown fences.")
	return strings.Join(parts, "\n\n")
}

type HTTPBuildError struct {
	Code    string
	Message string
}

func NewHTTPBuildError(code string, message string) *HTTPBuildError {
	return &HTTPBuildError{Code: code, Message: message}
}

func (e *HTTPBuildError) Error() string {
	return e.Code + ": " + e.Message
}
