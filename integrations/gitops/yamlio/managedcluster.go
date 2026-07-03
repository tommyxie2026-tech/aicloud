package yamlio

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/tommyxie2026-tech/aicloud/infra/api"
)

type ManagedClusterYAML struct {
	APIVersion string
	Kind       string
	Metadata   ObjectMetaYAML
	Spec       ManagedClusterSpecYAML
}

type ObjectMetaYAML struct {
	Name      string
	Namespace string
	Labels    map[string]string
}

type ManagedClusterSpecYAML struct {
	Environment string
	Workers     []WorkerGroupYAML
}

type WorkerGroupYAML struct {
	Name            string
	Replicas        int32
	MachineClassRef LocalObjectReferenceYAML
}

type LocalObjectReferenceYAML struct {
	Name string
}

type YAMLIOError struct {
	Code    string
	Message string
}

func NewYAMLIOError(code string, message string) *YAMLIOError {
	return &YAMLIOError{Code: code, Message: message}
}

func (e *YAMLIOError) Error() string {
	return e.Code + ": " + e.Message
}

func ReadManagedCluster(data []byte) (api.ManagedCluster, error) {
	if len(data) == 0 {
		return api.ManagedCluster{}, NewYAMLIOError("EmptyInput", "managed cluster YAML is empty")
	}
	doc, err := parseManagedClusterYAML(string(data))
	if err != nil {
		return api.ManagedCluster{}, err
	}
	cluster := fromManagedClusterYAML(doc)
	if err := api.ValidateManagedCluster(cluster); err != nil {
		return api.ManagedCluster{}, NewYAMLIOError("InvalidManagedCluster", err.Error())
	}
	return cluster, nil
}

func WriteManagedCluster(cluster api.ManagedCluster) ([]byte, error) {
	if err := api.ValidateManagedCluster(cluster); err != nil {
		return nil, NewYAMLIOError("InvalidManagedCluster", err.Error())
	}
	return formatManagedClusterYAML(toManagedClusterYAML(cluster)), nil
}

func fromManagedClusterYAML(doc ManagedClusterYAML) api.ManagedCluster {
	cluster := api.NewManagedCluster(doc.Metadata.Name, doc.Metadata.Namespace, doc.Spec.Environment)
	cluster.APIVersion = doc.APIVersion
	cluster.Kind = doc.Kind
	cluster.Labels = doc.Metadata.Labels
	for _, worker := range doc.Spec.Workers {
		cluster.Spec.Workers = append(cluster.Spec.Workers, api.WorkerGroupSpec{
			Name:     worker.Name,
			Replicas: worker.Replicas,
			MachineClassRef: api.LocalObjectReference{
				Name: worker.MachineClassRef.Name,
			},
		})
	}
	return cluster
}

func toManagedClusterYAML(cluster api.ManagedCluster) ManagedClusterYAML {
	doc := ManagedClusterYAML{
		APIVersion: cluster.APIVersion,
		Kind:       cluster.Kind,
		Metadata: ObjectMetaYAML{
			Name:      cluster.Name,
			Namespace: cluster.Namespace,
			Labels:    cluster.Labels,
		},
		Spec: ManagedClusterSpecYAML{
			Environment: cluster.Spec.Environment,
		},
	}
	for _, worker := range cluster.Spec.Workers {
		doc.Spec.Workers = append(doc.Spec.Workers, WorkerGroupYAML{
			Name:     worker.Name,
			Replicas: worker.Replicas,
			MachineClassRef: LocalObjectReferenceYAML{
				Name: worker.MachineClassRef.Name,
			},
		})
	}
	return doc
}

func parseManagedClusterYAML(raw string) (ManagedClusterYAML, error) {
	if strings.Contains(raw, "[") || strings.Contains(raw, "]") {
		return ManagedClusterYAML{}, NewYAMLIOError("InvalidYAML", "unsupported YAML sequence syntax")
	}
	doc := ManagedClusterYAML{Metadata: ObjectMetaYAML{Labels: map[string]string{}}}
	lines := strings.Split(raw, "\n")
	section := ""
	inLabels := false
	inWorkers := false
	inMachineClassRef := false
	var currentWorker *WorkerGroupYAML
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasSuffix(trimmed, ":") && !strings.HasPrefix(trimmed, "-") {
			key := strings.TrimSuffix(trimmed, ":")
			switch key {
			case "metadata", "spec":
				section = key
				inLabels = false
				inWorkers = false
				inMachineClassRef = false
			case "labels":
				inLabels = true
				inWorkers = false
				inMachineClassRef = false
			case "workers":
				inWorkers = true
				inLabels = false
				inMachineClassRef = false
			case "machineClassRef":
				inMachineClassRef = true
			default:
				return ManagedClusterYAML{}, NewYAMLIOError("InvalidYAML", "unsupported section: "+key)
			}
			continue
		}
		if strings.HasPrefix(trimmed, "- ") {
			if !inWorkers {
				return ManagedClusterYAML{}, NewYAMLIOError("InvalidYAML", "list item outside workers")
			}
			worker := WorkerGroupYAML{}
			currentWorker = &worker
			doc.Spec.Workers = append(doc.Spec.Workers, worker)
			currentWorker = &doc.Spec.Workers[len(doc.Spec.Workers)-1]
			inMachineClassRef = false
			trimmed = strings.TrimPrefix(trimmed, "- ")
			if trimmed == "" {
				continue
			}
		}
		key, value, ok := splitYAMLKeyValue(trimmed)
		if !ok {
			return ManagedClusterYAML{}, NewYAMLIOError("InvalidYAML", "expected key-value line: "+trimmed)
		}
		if inWorkers && currentWorker != nil {
			if inMachineClassRef {
				if key != "name" {
					return ManagedClusterYAML{}, NewYAMLIOError("InvalidYAML", "unsupported machineClassRef field: "+key)
				}
				currentWorker.MachineClassRef.Name = value
				continue
			}
			switch key {
			case "name":
				currentWorker.Name = value
			case "replicas":
				replicas, err := strconv.ParseInt(value, 10, 32)
				if err != nil {
					return ManagedClusterYAML{}, NewYAMLIOError("InvalidYAML", "invalid replicas")
				}
				currentWorker.Replicas = int32(replicas)
			default:
				return ManagedClusterYAML{}, NewYAMLIOError("InvalidYAML", "unsupported worker field: "+key)
			}
			continue
		}
		if inLabels {
			doc.Metadata.Labels[key] = value
			continue
		}
		switch section {
		case "":
			if key == "apiVersion" {
				doc.APIVersion = value
			} else if key == "kind" {
				doc.Kind = value
			} else {
				return ManagedClusterYAML{}, NewYAMLIOError("InvalidYAML", "unsupported top-level field: "+key)
			}
		case "metadata":
			if key == "name" {
				doc.Metadata.Name = value
			} else if key == "namespace" {
				doc.Metadata.Namespace = value
			} else {
				return ManagedClusterYAML{}, NewYAMLIOError("InvalidYAML", "unsupported metadata field: "+key)
			}
		case "spec":
			if key == "environment" {
				doc.Spec.Environment = value
			} else {
				return ManagedClusterYAML{}, NewYAMLIOError("InvalidYAML", "unsupported spec field: "+key)
			}
		}
	}
	return doc, nil
}

func splitYAMLKeyValue(line string) (string, string, bool) {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	key := strings.TrimSpace(parts[0])
	value := strings.Trim(strings.TrimSpace(parts[1]), "\"")
	if key == "" {
		return "", "", false
	}
	return key, value, true
}

func formatManagedClusterYAML(doc ManagedClusterYAML) []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "apiVersion: %s\n", doc.APIVersion)
	fmt.Fprintf(&buf, "kind: %s\n", doc.Kind)
	buf.WriteString("metadata:\n")
	fmt.Fprintf(&buf, "  name: %s\n", doc.Metadata.Name)
	fmt.Fprintf(&buf, "  namespace: %s\n", doc.Metadata.Namespace)
	if len(doc.Metadata.Labels) > 0 {
		buf.WriteString("  labels:\n")
		for key, value := range doc.Metadata.Labels {
			fmt.Fprintf(&buf, "    %s: %s\n", key, value)
		}
	}
	buf.WriteString("spec:\n")
	fmt.Fprintf(&buf, "  environment: %s\n", doc.Spec.Environment)
	buf.WriteString("  workers:\n")
	for _, worker := range doc.Spec.Workers {
		fmt.Fprintf(&buf, "    - name: %s\n", worker.Name)
		fmt.Fprintf(&buf, "      replicas: %d\n", worker.Replicas)
		buf.WriteString("      machineClassRef:\n")
		fmt.Fprintf(&buf, "        name: %s\n", worker.MachineClassRef.Name)
	}
	return buf.Bytes()
}
