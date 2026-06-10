package openai

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestHTTPClientGenerate(t *testing.T) {
	doer := &fakeHTTPDoer{responseBody: `{"choices":[{"message":{"content":"{\"ok\":true}"},"finish_reason":"stop"}],"usage":{"prompt_tokens":11,"completion_tokens":7}}`, statusCode: 200}
	client, err := NewHTTPClient(validHTTPConfig(), doer, fakeSecretResolver{value: "test-key"})
	if err != nil {
		t.Fatalf("NewHTTPClient returned error: %v", err)
	}

	resp, err := client.Generate(context.Background(), CompatibleRequest{Model: "gpt-test", Instruction: "return json", OutputSchema: "ChangePlan"})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if resp.OutputText != `{"ok":true}` {
		t.Fatalf("unexpected output text: %s", resp.OutputText)
	}
	if resp.FinishReason != "stop" {
		t.Fatalf("unexpected finish reason: %s", resp.FinishReason)
	}
	if resp.InputTokens != 11 || resp.OutputTokens != 7 {
		t.Fatalf("unexpected token usage: %d/%d", resp.InputTokens, resp.OutputTokens)
	}
	if doer.request == nil {
		t.Fatalf("expected request")
	}
	if doer.request.Header.Get("Authorization") != "Bearer test-key" {
		t.Fatalf("missing authorization header")
	}
	if doer.request.URL.String() != "https://api.example.com/v1/chat/completions" {
		t.Fatalf("unexpected request url: %s", doer.request.URL.String())
	}
}

func TestNewHTTPClientRequiresSecretResolver(t *testing.T) {
	_, err := NewHTTPClient(validHTTPConfig(), &fakeHTTPDoer{}, nil)
	if err == nil {
		t.Fatalf("expected missing secret resolver error")
	}
}

func TestHTTPClientGenerateRejectsEmptySecret(t *testing.T) {
	client, err := NewHTTPClient(validHTTPConfig(), &fakeHTTPDoer{}, fakeSecretResolver{})
	if err != nil {
		t.Fatalf("NewHTTPClient returned error: %v", err)
	}
	_, err = client.Generate(context.Background(), CompatibleRequest{Model: "gpt-test", Instruction: "return json"})
	if err == nil {
		t.Fatalf("expected empty secret error")
	}
}

func TestHTTPClientGenerateHandlesNon2xx(t *testing.T) {
	doer := &fakeHTTPDoer{statusCode: 500, responseBody: `server error`}
	client, err := NewHTTPClient(validHTTPConfig(), doer, fakeSecretResolver{value: "test-key"})
	if err != nil {
		t.Fatalf("NewHTTPClient returned error: %v", err)
	}
	_, err = client.Generate(context.Background(), CompatibleRequest{Model: "gpt-test", Instruction: "return json"})
	if err == nil {
		t.Fatalf("expected non-2xx error")
	}
}

func TestHTTPClientGenerateRejectsMissingChoices(t *testing.T) {
	doer := &fakeHTTPDoer{statusCode: 200, responseBody: `{"choices":[]}`}
	client, err := NewHTTPClient(validHTTPConfig(), doer, fakeSecretResolver{value: "test-key"})
	if err != nil {
		t.Fatalf("NewHTTPClient returned error: %v", err)
	}
	_, err = client.Generate(context.Background(), CompatibleRequest{Model: "gpt-test", Instruction: "return json"})
	if err == nil {
		t.Fatalf("expected missing choices error")
	}
}

func validHTTPConfig() Config {
	return Config{Name: "openai-compatible", Endpoint: "https://api.example.com/v1", SecretRef: "secret/openai", DefaultModel: "gpt-test", TimeoutSeconds: 30, MaxRetries: 2, MaxInputTokens: 32768, MaxOutputTokens: 2048}
}

type fakeSecretResolver struct{ value string }

func (r fakeSecretResolver) ResolveSecret(ctx context.Context, secretRef string) (string, error) {
	return r.value, nil
}

type fakeHTTPDoer struct {
	request      *http.Request
	statusCode   int
	responseBody string
}

func (d *fakeHTTPDoer) Do(req *http.Request) (*http.Response, error) {
	d.request = req
	status := d.statusCode
	if status == 0 {
		status = 200
	}
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(d.responseBody)), Header: make(http.Header)}, nil
}
