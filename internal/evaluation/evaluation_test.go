package evaluation

import (
	"testing"
	"time"
)

func TestDigestConfigIsStableAcrossMapOrder(t *testing.T) {
	left := validConfig()
	left.ToolVersions = map[string]string{"shell": "v2", "github": "v1"}
	left.Parameters = map[string]string{"temperature": "0", "effort": "medium"}
	right := validConfig()
	right.ToolVersions = map[string]string{"github": "v1", "shell": "v2"}
	right.Parameters = map[string]string{"effort": "medium", "temperature": "0"}

	leftDigest, err := DigestConfig(left)
	if err != nil {
		t.Fatalf("DigestConfig(left): %v", err)
	}
	rightDigest, err := DigestConfig(right)
	if err != nil {
		t.Fatalf("DigestConfig(right): %v", err)
	}
	if leftDigest != rightDigest {
		t.Fatalf("digests differ: %s != %s", leftDigest, rightDigest)
	}
}

func TestEvaluationGateBlocksRegression(t *testing.T) {
	run, err := NewRun(
		"task-1",
		"trace-1",
		validConfig(),
		Metrics{
			QualityScore:          0.90,
			SafetyScore:           0.70,
			ReliabilityScore:      0.95,
			LatencyP95MS:          1000,
			CostPerSuccessfulTask: 0.20,
			HumanInterventionRate: 0.05,
			TaskSuccessRate:       0.92,
		},
		Thresholds{
			MinimumQuality:          0.85,
			MinimumSafety:           0.90,
			MinimumReliability:      0.90,
			MaximumLatencyP95MS:     2000,
			MaximumCostPerSuccess:   0.50,
			MaximumHumanIntervention: 0.10,
			MinimumTaskSuccessRate:  0.90,
		},
		[]byte("raw output"),
		time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("NewRun: %v", err)
	}
	if run.Gate.Passed {
		t.Fatal("expected safety regression to block the gate")
	}
	if len(run.Gate.Reasons) != 1 || run.Gate.Reasons[0] != "safety score below threshold" {
		t.Fatalf("unexpected gate reasons: %#v", run.Gate.Reasons)
	}
	if run.ConfigDigest == "" || run.RawOutputDigest == "" {
		t.Fatalf("missing evidence digests: %#v", run)
	}
}

func validConfig() Config {
	return Config{
		ModelID: "model-1", ModelVersion: "v1", Provider: "provider-1",
		PromptVersion: "prompt-v1", WorkflowVersion: "workflow-v1",
		TokenBudget: 10000, TimeBudgetMS: 60000, RetryPolicyVersion: "retry-v1",
		SandboxProfile: "sandbox-v1", DatasetID: "developer-agent",
		DatasetVersion: "dataset-v1", EvaluatorID: "quality-evaluator",
		EvaluatorVersion: "evaluator-v1",
	}
}
