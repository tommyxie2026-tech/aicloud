package gitops

import "github.com/tommyxie2026-tech/aicloud/integrations/gitops/yamlio"

// ManagedClusterManifestBytesResult is the dry-run result of parsing,
// object-level patching, and rendering a ManagedCluster manifest.
// It does not imply any filesystem, GitHub, or Kubernetes side effect.
type ManagedClusterManifestBytesResult struct {
	WriteResult *WriteResult
	Manifest    []byte
}

// ManagedClusterManifestBytesWriter adapts an object-level ManifestWriter
// to a byte-oriented ManagedCluster manifest flow.
//
// Flow:
// input bytes -> yamlio.ReadManagedCluster -> ObjectWriter.WriteManagedCluster
// -> yamlio.WriteManagedCluster(result.Updated) -> result bytes.
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
	objectWriter := ManifestWriter(NewDryRunManifestWriter())
	if w != nil && w.ObjectWriter != nil {
		objectWriter = w.ObjectWriter
	}
	cluster, err := yamlio.ReadManagedCluster(input)
	if err != nil {
		return nil, NewGitOpsError("ReadManagedClusterManifestFailed", err.Error())
	}
	writeResult, err := objectWriter.WriteManagedCluster(plan, cluster)
	if err != nil {
		return nil, err
	}
	manifest, err := yamlio.WriteManagedCluster(writeResult.Updated)
	if err != nil {
		return nil, NewGitOpsError("RenderManagedClusterManifestFailed", err.Error())
	}
	return &ManagedClusterManifestBytesResult{WriteResult: writeResult, Manifest: manifest}, nil
}
