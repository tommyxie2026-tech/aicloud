package openai

import "testing"

func TestBuildChatCompletionRequest(t *testing.T) {
	body, err := BuildChatCompletionRequest(CompatibleRequest{
		Model:           "gpt-test",
		SystemPrompt:    "return JSON only",
		ContextText:     "cluster=dev-gpu-cluster",
		Instruction:     "scale gpu workers from 3 to 6",
		OutputSchema:    "ChangePlan",
		MaxOutputTokens: 1024,
		Temperature:     0.2,
	})
	if err != nil {
		t.Fatalf("BuildChatCompletionRequest returned error: %v", err)
	}
	if body.Model != "gpt-test" {
		t.Fatalf("unexpected model: %s", body.Model)
	}
	if len(body.Messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(body.Messages))
	}
	if body.Messages[0].Role != "system" {
		t.Fatalf("expected first message to be system, got %s", body.Messages[0].Role)
	}
	if body.Messages[1].Role != "user" {
		t.Fatalf("expected second message to be user, got %s", body.Messages[1].Role)
	}
	if body.Messages[2].Content != "scale gpu workers from 3 to 6" {
		t.Fatalf("unexpected instruction message: %s", body.Messages[2].Content)
	}
	if body.ResponseFormat.Type != "json_object" {
		t.Fatalf("expected json_object response format, got %s", body.ResponseFormat.Type)
	}
	if body.MaxTokens != 1024 {
		t.Fatalf("expected max tokens 1024, got %d", body.MaxTokens)
	}
	if body.Temperature != 0.2 {
		t.Fatalf("expected temperature 0.2, got %f", body.Temperature)
	}
}

func TestBuildChatCompletionRequestWithoutOptionalMessages(t *testing.T) {
	body, err := BuildChatCompletionRequest(CompatibleRequest{Model: "gpt-test", Instruction: "return status"})
	if err != nil {
		t.Fatalf("BuildChatCompletionRequest returned error: %v", err)
	}
	if len(body.Messages) != 1 {
		t.Fatalf("expected one message, got %d", len(body.Messages))
	}
	if body.Messages[0].Role != "user" {
		t.Fatalf("expected user message, got %s", body.Messages[0].Role)
	}
	if body.ResponseFormat.Type != "" {
		t.Fatalf("expected empty response format without output schema")
	}
}

func TestBuildChatCompletionRequestRequiresModel(t *testing.T) {
	_, err := BuildChatCompletionRequest(CompatibleRequest{Instruction: "return status"})
	if err == nil {
		t.Fatalf("expected missing model error")
	}
}

func TestBuildChatCompletionRequestRequiresInstruction(t *testing.T) {
	_, err := BuildChatCompletionRequest(CompatibleRequest{Model: "gpt-test"})
	if err == nil {
		t.Fatalf("expected missing instruction error")
	}
}
