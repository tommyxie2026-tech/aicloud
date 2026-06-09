package gitops

// BranchPlan is a dry-run representation of a future Git branch operation.
type BranchPlan struct {
	BaseBranch string
	HeadBranch string
	Commit     CommitPlan
	PullRequest PullRequestPlan
}

// CommitPlan is a dry-run representation of a future Git commit.
type CommitPlan struct {
	Message string
	Files   []FileChangePlan
}

// FileChangePlan is a dry-run representation of a file-level manifest change.
type FileChangePlan struct {
	Path    string
	Summary string
}

// PullRequestPlan is a dry-run representation of a future pull request.
type PullRequestPlan struct {
	Title string
	Body  string
	Draft bool
}

// BuildBranchPlan converts a ManifestPatchPlan and WriteResult into dry-run branch/commit/PR metadata.
// It does not create branches, commits, or pull requests.
func BuildBranchPlan(baseBranch string, plan ManifestPatchPlan, result WriteResult) (*BranchPlan, error) {
	if baseBranch == "" {
		return nil, NewGitOpsError("MissingBaseBranch", "base branch is required")
	}
	if plan.PR.BranchName == "" {
		return nil, NewGitOpsError("MissingHeadBranch", "patch plan PR branch name is required")
	}
	if result.OutputPath == "" {
		return nil, NewGitOpsError("MissingOutputPath", "write result output path is required")
	}
	return &BranchPlan{
		BaseBranch: baseBranch,
		HeadBranch: plan.PR.BranchName,
		Commit: CommitPlan{
			Message: plan.PR.CommitMessage,
			Files: []FileChangePlan{{Path: result.OutputPath, Summary: result.Summary}},
		},
		PullRequest: PullRequestPlan{
			Title: plan.PR.Title,
			Body:  buildPullRequestBody(plan, result),
			Draft: plan.PR.Draft,
		},
	}, nil
}

func buildPullRequestBody(plan ManifestPatchPlan, result WriteResult) string {
	body := "## Intent\n\n"
	body += plan.PR.Title + "\n\n"
	body += "## Manifest\n\n"
	body += result.OutputPath + "\n\n"
	body += "## Changes\n\n"
	body += result.Summary + "\n\n"
	body += "## Safety Boundary\n\n"
	body += "This plan is dry-run only. It does not apply live infrastructure changes.\n"
	return body
}
