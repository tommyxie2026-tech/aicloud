package openai

import (
	"context"
	"testing"

	"github.com/tommyxie2026-tech/aicloud/runtime/secrets"
)

func TestHTTPClientAcceptsRuntimeMemorySecretResolver(t *testing.T) {
	secretRef := "secret/aicloud-system/openai-public:api-key"
	resolver := secrets.NewMemoryResolver(map[string]string{secretRef: "test-key"})
	config := validHTTPConfig()
	config.SecretRef = secretRef

	doer := &fakeHTTPDoer{responseBody: successResponseBody(), statusCode: 200}
	client, err := NewHTTPClient(config, doer, resolver)
	if err != nil {
		t.Fatalf("NewHTTPClient returned error: %v", err)
	}
	resp, err := client.Generate(context.Background(), CompatibleRequest{Model: "gpt-test", Instruction: "return json"})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if resp.OutputText != `{"ok":true}` {
		t.Fatalf("unexpected output text: %s", resp.OutputText)
	}
	if doer.request.Header.Get("Authorization") != "Bearer test-key" {
		t.Fatalf("expected Authorization header to use runtime resolver secret")
	}
}
