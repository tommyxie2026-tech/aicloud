package openai

// ChatCompletionRequest is the OpenAI-compatible request body used by many hosted,
// private, and self-hosted model servers.
type ChatCompletionRequest struct {
	Model       string                `json:"model"`
	Messages    []ChatCompletionMessage `json:"messages"`
	Temperature float32               `json:"temperature,omitempty"`
	MaxTokens   int                   `json:"max_tokens,omitempty"`
	ResponseFormat ResponseFormat     `json:"response_format,omitempty"`
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
	if req.Model == "" {
		return ChatCompletionRequest{}, NewHTTPBuildError("MissingModel", "model is required")
	}
	if req.Instruction == "" {
		return ChatCompletionRequest{}, NewHTTPBuildError("MissingInstruction", "instruction is required")
	}

	messages := make([]ChatCompletionMessage, 0, 3)
	if req.SystemPrompt != "" {
		messages = append(messages, ChatCompletionMessage{Role: "system", Content: req.SystemPrompt})
	}
	if req.ContextText != "" {
		messages = append(messages, ChatCompletionMessage{Role: "user", Content: "Context:\n" + req.ContextText})
	}
	messages = append(messages, ChatCompletionMessage{Role: "user", Content: req.Instruction})

	body := ChatCompletionRequest{
		Model:       req.Model,
		Messages:    messages,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxOutputTokens,
	}
	if req.OutputSchema != "" {
		body.ResponseFormat = ResponseFormat{Type: "json_object"}
	}
	return body, nil
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
