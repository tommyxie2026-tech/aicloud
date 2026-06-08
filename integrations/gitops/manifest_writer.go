package gitops

import (
	"fmt"
	"strings"

	infraapi "github.com/tommyxie2026-tech/aicloud/infra/api"
)

// ManifestWriter converts a patch plan and an input object into an updated object.
// It does not define live apply behavior.
type ManifestWriter interface {
	WriteManagedCluster(plan ManifestPatchPlan, cluster infraapi.ManagedCluster) (*WriteResult, error)
}

// WriteResult is the dry-run result of manifest generation.
type WriteResult struct {
	SourcePath string
	OutputPath string
	Summary    string
	Updated    infraapi.ManagedCluster
	Changes    []ManifestFieldChange
	Rollback   []ManifestFieldChange
}

// DryRunManifestWriter applies object-level patches in memory and returns a write result.
// It does not read files, write files, create commits, create PRs, or call Kubernetes.
type DryRunManifestWriter struct{}

func NewDryRunManifestWriter() *DryRunManifestWriter {
	return &DryRunManifestWriter{}
}

func (w *DryRunManifestWriter) WriteManagedCluster(plan ManifestPatchPlan, cluster infraapi.ManagedCluster) (*WriteResult, error) {
	updated, err := ApplyManagedClusterPatch(plan, cluster)
	if err != nil {
		return nil, err
	}
	return &WriteResult{
		SourcePath: plan.SourcePath,
		OutputPath: plan.OutputPath,
		Summary:    summarizeChanges(plan),
		Updated:    updated,
		Changes:    append([]ManifestFieldChange{}, plan.Changes...),
		Rollback:   append([]ManifestFieldChange{}, plan.Rollback...),
	}, nil
}

func summarizeChanges(plan ManifestPatchPlan) string {
	if len(plan.Changes) == 0 {
		return "no manifest changes"
	}
	parts := make([]string, 0, len(plan.Changes))
	for _, change := range plan.Changes {
		parts = append(parts, fmt.Sprintf("%s: %v -> %v", change.Field, change.From, change.To))
	}
	return strings.Join(parts, "; ")
}
