package eval

import (
	"context"
	"testing"

	"github.com/tommyxie2026-tech/aicloud/model/mock"
	"github.com/tommyxie2026-tech/aicloud/model/provider"
	"github.com/tommyxie2026-tech/aicloud/model/schema"
)

func TestRunnerRunsDefaultDevScaleOutCase(t *testing.T) {
	runner := NewRunner(mock.NewProvider(), schema.NewBasicValidator())

	report, err := runner.Run(context.Background(), []EvalCase{DefaultDevScaleOutCase()})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if report == nil {
		t.Fatalf("expected report, got nil")
	}
	if report.Total != 1 {
		t.Fatalf("expected total 1, got %d", report.Total)
	}
	if report.Passed != 1 {
		t.Fatalf("expected passed 1, got %d", report.Passed)
	}
	if report.Failed != 0 {
		t.Fatalf("expected failed 0, got %d", report.Failed)
	}
	if report.AverageScore < 85 {
		t.Fatalf("expected average score >= 85, got %d", report.AverageScore)
	}
	if report.SafetyFailures != 0 {
		t.Fatalf("expected no safety failures, got %d", report.SafetyFailures)
	}
	if report.SchemaFailures != 0 {
		t.Fatalf("expected no schema failures, got %d", report.SchemaFailures)
	}
	if len(report.Cases) != 1 {
		t.Fatalf("expected one case result, got %d", len(report.Cases))
	}
	if !report.Cases[0].Passed {
		t.Fatalf("expected case to pass, failures: %#v", report.Cases[0].Failures)
	}
}

func TestRecommendationAllowsPlanningForHighScore(t *testing.T) {
	runner := NewRunner(mock.NewProvider(), schema.NewBasicValidator())

	report, err := runner.Run(context.Background(), []EvalCase{DefaultDevScaleOutCase()})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !containsTask(report.Recommendation.AllowedTasks, provider.TaskGeneratePlan) {
		t.Fatalf("expected recommendation to allow GeneratePlan, got %#v", report.Recommendation.AllowedTasks)
	}
}

func containsTask(tasks []provider.TaskType, target provider.TaskType) bool {
	for _, task := range tasks {
		if task == target {
			return true
		}
	}
	return false
}
