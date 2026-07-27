package openai

import (
	"context"
	"testing"
)

func TestHTTPClientHealthUsesModelsEndpointAndAuthentication(t *testing.T) {
	doer := &fakeHTTPDoer{statusCode: 200, responseBody: `{"data":[]}`}
	client, err := NewHTTPClient(validHTTPConfig(), doer, fakeSecretResolver{value: "test-key"})
	if err != nil {
		t.Fatalf("NewHTTPClient returned error: %v", err)
	}
	if err := client.Health(context.Background()); err != nil {
		t.Fatalf("Health returned error: %v", err)
	}
	if doer.request == nil {
		t.Fatal("expected health request")
	}
	if doer.request.URL.String() != "https://api.example.com/v1/models" {
		t.Fatalf("health URL = %s", doer.request.URL.String())
	}
	if doer.request.Header.Get("Authorization") != "Bearer test-key" {
		t.Fatal("missing health authorization header")
	}
}

func TestHTTPClientHealthRejectsNon2xx(t *testing.T) {
	client, err := NewHTTPClient(validHTTPConfig(), &fakeHTTPDoer{statusCode: 503, responseBody: `unavailable`}, fakeSecretResolver{value: "test-key"})
	if err != nil {
		t.Fatalf("NewHTTPClient returned error: %v", err)
	}
	if err := client.Health(context.Background()); err == nil {
		t.Fatal("expected health error")
	}
}
