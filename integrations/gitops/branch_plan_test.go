package gitops

import "testing"

func TestBuildBranchPlan(t *testing.T) {
	writer := NewDryRunManifestWriter()
	cluster := validManagedCluster(3)
	plan := validPatchPlan(3, 6)
	plan.PR = PRMetadata{
		BranchName:    "aicloud/request-001/scaleout/dev-gpu-cluster",
		CommitMessage: "aicloud: ScaleOut ManagedCluster/dev-gpu-cluster",
		Title:         "ScaleOut ManagedCluster/dev-gpu-cluster",
		Draft:         false,
	}
	result, err := writer.WriteManagedCluster(plan, cluster)
	if err != nil {
		t.Fatalf("WriteManagedCluster returned error: %v", err)
	}

	branchPlan, err := BuildBranchPlan("main", plan, *result)
	if err != nil {
		t.Fatalf("BuildBranchPlan returned error: %v", err)
	}
	if branchPlan.BaseBranch != "main" {
		t.Fatalf("expected base branch main, got %s", branchPlan.BaseBranch)
	}
	if branchPlan.HeadBranch != plan.PR.BranchName {
		t.Fatalf("unexpected head branch: %s", branchPlan.HeadBranch)
	}
	if branchPlan.Commit.Message != plan.PR.CommitMessage {
		t.Fatalf("unexpected commit message: %s", branchPlan.Commit.Message)
	}
	if len(branchPlan.Commit.Files) != 1 {
		t.Fatalf("expected one file change, got %d", len(branchPlan.Commit.Files))
	}
	if branchPlan.Commit.Files[0].Path != result.OutputPath {
		t.Fatalf("unexpected file path: %s", branchPlan.Commit.Files[0].Path)
	}
	if branchPlan.PullRequest.Title != plan.PR.Title {
		t.Fatalf("unexpected PR title: %s", branchPlan.PullRequest.Title)
	}
	if branchPlan.PullRequest.Draft {
		t.Fatalf("expected non-draft PR plan")
	}
	if branchPlan.PullRequest.Body == "" {
		t.Fatalf("expected PR body")
	}
}

func TestBuildBranchPlanPreservesDraftFlag(t *testing.T) {
	writer := NewDryRunManifestWriter()
	cluster := validManagedCluster(3)
	plan := validPatchPlan(3, 6)
	plan.PR = PRMetadata{BranchName: "aicloud/request-001/scaleout/dev-gpu-cluster", CommitMessage: "msg", Title: "title", Draft: true}
	result, err := writer.WriteManagedCluster(plan, cluster)
	if err != nil {
		t.Fatalf("WriteManagedCluster returned error: %v", err)
	}

	branchPlan, err := BuildBranchPlan("main", plan, *result)
	if err != nil {
		t.Fatalf("BuildBranchPlan returned error: %v", err)
	}
	if !branchPlan.PullRequest.Draft {
		t.Fatalf("expected draft PR plan")
	}
}

func TestBuildBranchPlanRequiresBaseBranch(t *testing.T) {
	_, err := BuildBranchPlan("", ManifestPatchPlan{}, WriteResult{OutputPath: "x"})
	if err == nil {
		t.Fatalf("expected missing base branch error")
	}
}

func TestBuildBranchPlanRequiresHeadBranch(t *testing.T) {
	_, err := BuildBranchPlan("main", ManifestPatchPlan{}, WriteResult{OutputPath: "x"})
	if err == nil {
		t.Fatalf("expected missing head branch error")
	}
}

func TestBuildBranchPlanRequiresOutputPath(t *testing.T) {
	plan := ManifestPatchPlan{PR: PRMetadata{BranchName: "aicloud/request-001/change/target"}}
	_, err := BuildBranchPlan("main", plan, WriteResult{})
	if err == nil {
		t.Fatalf("expected missing output path error")
	}
}
