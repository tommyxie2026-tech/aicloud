package httpapi

import (
	"testing"

	"github.com/tommyxie2026-tech/aicloud/model/provider"
	"github.com/tommyxie2026-tech/aicloud/model/schema"
)

func TestModelExecutionRequestContract(t *testing.T) {
	valid := provider.ProviderRequest{
		TaskType:    provider.TaskGeneratePlan,
		Instruction: "produce a plan",
		OutputSchema: schema.Reference{Name: "plan", Version: "v1"},
	}
	if err := validateModelExecutionRequest(valid); err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}

	cases := []struct {
		name    string
		mutate  func(*provider.ProviderRequest)
	}{
		{name: "missing taskType", mutate: func(r *provider.ProviderRequest) { r.TaskType = "" }},
		{name: "unsupported taskType", mutate: func(r *provider.ProviderRequest) { r.TaskType = provider.TaskType("Unknown") }},
		{name: "missing instruction", mutate: func(r *provider.ProviderRequest) { r.Instruction = "" }},
		{name: "missing schema name", mutate: func(r *provider.ProviderRequest) { r.OutputSchema.Name = "" }},
		{name: "missing schema version", mutate: func(r *provider.ProviderRequest) { r.OutputSchema.Version = "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request := valid
			tc.mutate(&request)
			if err := validateModelExecutionRequest(request); err == nil {
				t.Fatal("invalid request was accepted")
			}
		})
	}
}
