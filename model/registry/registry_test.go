package registry

import (
	"context"
	"testing"

	"github.com/tommyxie2026-tech/aicloud/model/mock"
)

func TestMemoryRegistryRegisterGetAndList(t *testing.T) {
	r := NewMemoryRegistry()
	p := mock.NewProvider()

	if err := r.Register(p); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	got, ok := r.Get("mock")
	if !ok {
		t.Fatalf("expected provider mock to exist")
	}
	if got.Name() != "mock" {
		t.Fatalf("expected provider name mock, got %s", got.Name())
	}

	list := r.List()
	if len(list) != 1 {
		t.Fatalf("expected 1 provider info, got %d", len(list))
	}
	if list[0].Name != "mock" {
		t.Fatalf("expected mock provider info, got %s", list[0].Name)
	}
	if !list[0].SupportsStructuredOutput {
		t.Fatalf("expected mock to support structured output")
	}
}

func TestMemoryRegistryRejectsDuplicateProvider(t *testing.T) {
	r := NewMemoryRegistry()

	if err := r.Register(mock.NewProvider()); err != nil {
		t.Fatalf("first Register returned error: %v", err)
	}
	if err := r.Register(mock.NewProvider()); err == nil {
		t.Fatalf("expected duplicate provider error")
	}
}

func TestMemoryRegistryHealth(t *testing.T) {
	r := NewMemoryRegistry()
	if err := r.Register(mock.NewProvider()); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	health := r.Health(context.Background())
	if len(health) != 1 {
		t.Fatalf("expected 1 health result, got %d", len(health))
	}
	if health[0].Name != "mock" {
		t.Fatalf("expected mock health, got %s", health[0].Name)
	}
	if !health[0].Available {
		t.Fatalf("expected mock provider to be available")
	}
}

func TestMemoryRegistryMissingProvider(t *testing.T) {
	r := NewMemoryRegistry()

	if _, ok := r.Get("missing"); ok {
		t.Fatalf("expected missing provider lookup to return false")
	}
}
