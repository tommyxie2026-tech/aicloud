package gitops

import (
	"github.com/tommyxie2026-tech/aicloud/integrations/gitops/yamlio"
)

// ManagedClusterManifestBytesResult is the dry-run result of reading manifest bytes,
// applying the existing object-level writer, and rendering updated manifest bytes.
// It does not imply filesystem, GitHub, Kubernetes, or live apply behavior.
type ManagedClusterManifestBytesResult struct {
	WriteResult *WriteResult
	Manifest    []byte
}

// ManagedClusterManifestBytesWriter adapts the existing object-level ManifestWriter
// to a byte-oriented dry-run path.
//
// The dependency direction is intentionally one-way:
// this wrapper depends on yamlio and ManifestWriter, while DryRunManifestWriter
// remains independent of yamlio.
type ManagedClusterManifestBytesWriter struct {
	ObjectWriter ManifestWriter
}

func NewManagedClusterManifestBytesWriter(objectWriter ManifestWriter) *ManagedClusterManifestBytesWriter {
	if objectWriter == nil {
		objectWriter = NewDryRunManifestWriter()
	}
	return &ManagedClusterManifestBytesWriter{ObjectWriter: objectWriter}
}

func (w *ManagedClusterManifestBytesWriter) WriteManagedClusterBytes(plan ManifestPatchPlan, input []byte) (*ManagedClusterManifestBytesResult, error) {
	if w == nil || w.ObjectWriter == nil {
		return nil, NewGitOpsError("MissingObjectWriter", "object writer is required")
	}
	cluster, err := yamlio.ReadManagedCluster(input)
	if err != nil {
		return nil, err
	}
	writeResult, err := w.ObjectWriter.WriteManagedCluster(plan, cluster)
	if err != nil {
		return nil, err
	}
	manifest, err := yamlio.WriteManagedCluster(writeResult.Updated)
	if err != nil {
		return nil, err
	}
	return &ManagedClusterManifestBytesResult{WriteResult: writeResult, Manifest: manifest}, nil
}
