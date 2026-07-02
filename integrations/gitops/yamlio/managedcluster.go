package yamlio

import (
	"github.com/tommyxie2026-tech/aicloud/infra/api"
	"gopkg.in/yaml.v3"
)

type ManagedClusterYAML struct {
	APIVersion string                 `yaml:"apiVersion"`
	Kind       string                 `yaml:"kind"`
	Metadata   ObjectMetaYAML         `yaml:"metadata"`
	Spec       ManagedClusterSpecYAML `yaml:"spec"`
}

type ObjectMetaYAML struct {
	Name      string            `yaml:"name"`
	Namespace string            `yaml:"namespace"`
	Labels    map[string]string `yaml:"labels,omitempty"`
}

type ManagedClusterSpecYAML struct {
	Environment string            `yaml:"environment"`
	Workers     []WorkerGroupYAML `yaml:"workers"`
}

type WorkerGroupYAML struct {
	Name            string                   `yaml:"name"`
	Replicas        int32                    `yaml:"replicas"`
	MachineClassRef LocalObjectReferenceYAML `yaml:"machineClassRef"`
}

type LocalObjectReferenceYAML struct {
	Name string `yaml:"name"`
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
	var doc ManagedClusterYAML
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return api.ManagedCluster{}, NewYAMLIOError("InvalidYAML", err.Error())
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
	data, err := yaml.Marshal(toManagedClusterYAML(cluster))
	if err != nil {
		return nil, NewYAMLIOError("MarshalFailed", err.Error())
	}
	return data, nil
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
