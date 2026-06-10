package openai

import "testing"

func TestBuildChatCompletionRequest(t *testing.T) {
	req := CompatibleRequest{
		Model:           "gpt-test",
		SystemPrompt:    "system prompt",
		Instruction:     "generate a plan",
		ContextText:     "userIntent: scale cluster",
		OutputSchema:    "ChangePlan",
		MaxOutputTokens: 1024,
		Temperature:     0.2,
	}

	body, err := BuildChatCompletionRequest(req)
	if err != nil {
		t.Fatalf("BuildChatCompletionRequest returned error: %v", err)
	}
	if body.Model != "gpt-test" {
		t.Fatalf("unexpected model: %s", body.Model)
	}
	if len(body.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(body.Messages))
	}
	if body.Messages[0].Role != "system" {
		t.Fatalf("expected first message to be system")
	}
	if body.Messages[1].Role != "user" {
		t.Fatalf("expected second message to be user")
	}
	if body.ResponseFormat.Type != "json_object" {
		t.Fatalf("expected json_object response format, got %s", body.ResponseFormat.Type)
	}
	if body.Stream {
		t.Fatalf("expected stream=false")
	}
	if body.MaxTokens != 1024 {
		t.Fatalf("expected max_tokens 1024, got %d", body.MaxTokens)
	}
}

func TestBuildChatCompletionRequestWithoutSchema(t *testing.T) {
	body, err := BuildChatCompletionRequest(CompatibleRequest{Model: "m", Instruction: "do it"})
	if err != nil {
		t.Fatalf("BuildChatCompletionRequest returned error: %v", err)
	}
	if body.ResponseFormat.Type != "" {
		t.Fatalf("expected empty response format without schema")
	}
}

func TestBuildChatCompletionRequestRejectsMissingModel(t *testing.T) {
	_, err := BuildChatCompletionRequest(CompatibleRequest{Instruction: "do it"})
	if err == nil {
		t.Fatalf("expected missing model error")
	}
}

func TestBuildChatCompletionRequestRejectsMissingInstruction(t *testing.T) {
	_, err := BuildChatCompletionRequest(CompatibleRequest{Model: "m"})
	if err == nil {
		t.Fatalf("expected missing instruction error")
	}
}

func TestBuildChatCompletionsURL(t *testing.T) {
	url, err := BuildChatCompletionsURL("https://api.example.com/v1/")
	if err != nil {
		t.Fatalf("BuildChatCompletionsURL returned error: %v", err)
	}
	if url != "https://api.example.com/v1/chat/completions" {
		t.Fatalf("unexpected url: %s", url)
	}
}

func TestBuildChatCompletionsURLRejectsMissingEndpoint(t *testing.T) {
	_, err := BuildChatCompletionsURL("")
	if err == nil {
		t.Fatalf("expected missing endpoint error")
	}
}
