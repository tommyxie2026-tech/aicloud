package clusterapi

import (
	"fmt"
	"strings"

	"github.com/tommyxie2026-tech/aicloud/infra/api"
)

type Mapper struct{}

func NewMapper() Mapper {
	return Mapper{}
}

func (m Mapper) MapManagedCluster(cluster api.ManagedCluster) (MappingResult, error) {
	if err := api.ValidateManagedCluster(cluster); err != nil {
		return MappingResult{}, NewMappingError("InvalidManagedCluster", err.Error())
	}
	result := MappingResult{
		Cluster: DesiredCluster{
			Name:        cluster.Name,
			Namespace:   cluster.Namespace,
			Environment: cluster.Spec.Environment,
			Labels:      baseLabels(cluster.Name, cluster.Spec.Environment, ""),
		},
	}
	for _, worker := range cluster.Spec.Workers {
		result.MachineDeployments = append(result.MachineDeployments, DesiredMachineDeployment{
			Name:             machineDeploymentName(cluster.Name, worker.Name),
			Namespace:        cluster.Namespace,
			ClusterName:      cluster.Name,
			WorkerGroupName:  worker.Name,
			Replicas:         worker.Replicas,
			MachineClassName: worker.MachineClassRef.Name,
			Labels:           baseLabels(cluster.Name, cluster.Spec.Environment, worker.Name),
		})
	}
	return result, nil
}

func machineDeploymentName(clusterName string, workerGroupName string) string {
	return sanitizeName(clusterName) + "-" + sanitizeName(workerGroupName)
}

func baseLabels(clusterName string, environment string, workerGroupName string) map[string]string {
	labels := map[string]string{
		"aicloud.dev/managed-by":          "aicloud",
		"aicloud.dev/managedcluster-name": clusterName,
		"aicloud.dev/environment":         environment,
	}
	if workerGroupName != "" {
		labels["aicloud.dev/worker-group"] = workerGroupName
	}
	return labels
}

func sanitizeName(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, "_", "-")
	value = strings.ReplaceAll(value, ".", "-")
	value = strings.ReplaceAll(value, "/", "-")
	value = strings.Trim(value, "-")
	if value == "" {
		return "unnamed"
	}
	return value
}

func MachineDeploymentPatchPath(clusterName string, workerGroupName string) string {
	return fmt.Sprintf("clusters/%s/machinedeployments/%s.yaml", sanitizeName(clusterName), machineDeploymentName(clusterName, workerGroupName))
}
