package kubevirt

import (
	"fmt"
	"strings"

	"github.com/tommyxie2026-tech/aicloud/infra/api"
)

type Mapper struct{}

func NewMapper() Mapper {
	return Mapper{}
}

func (m Mapper) MapManagedCluster(cluster api.ManagedCluster, classes []api.MachineClass) (MappingResult, error) {
	if err := api.ValidateManagedCluster(cluster); err != nil {
		return MappingResult{}, NewMappingError("InvalidManagedCluster", err.Error())
	}
	classIndex := indexMachineClasses(classes)
	result := MappingResult{}
	for _, worker := range cluster.Spec.Workers {
		class, ok := classIndex[worker.MachineClassRef.Name]
		if !ok {
			return MappingResult{}, NewMappingError("MachineClassNotFound", "machine class not found: "+worker.MachineClassRef.Name)
		}
		for ordinal := 1; ordinal <= worker.Replicas; ordinal++ {
			result.VirtualMachines = append(result.VirtualMachines, DesiredVirtualMachine{
				Name:             virtualMachineName(cluster.Metadata.Name, worker.Name, ordinal),
				Namespace:        cluster.Metadata.Namespace,
				ClusterName:      cluster.Metadata.Name,
				WorkerGroupName:  worker.Name,
				Ordinal:          ordinal,
				MachineClassName: class.Metadata.Name,
				CPU:              class.Spec.CPU,
				Memory:           class.Spec.Memory,
				GPUProfile:       class.Spec.GPU,
				StorageProfile:   class.Spec.Storage,
				Labels:           baseLabels(cluster.Metadata.Name, cluster.Spec.Environment, worker.Name, class.Metadata.Name),
			})
		}
	}
	return result, nil
}

func indexMachineClasses(classes []api.MachineClass) map[string]api.MachineClass {
	index := map[string]api.MachineClass{}
	for _, class := range classes {
		index[class.Metadata.Name] = class
	}
	return index
}

func virtualMachineName(clusterName string, workerGroupName string, ordinal int) string {
	return fmt.Sprintf("%s-%s-%04d", sanitizeName(clusterName), sanitizeName(workerGroupName), ordinal)
}

func baseLabels(clusterName string, environment string, workerGroupName string, machineClassName string) map[string]string {
	return map[string]string{
		"aicloud.dev/managed-by":          "aicloud",
		"aicloud.dev/managedcluster-name": clusterName,
		"aicloud.dev/environment":         environment,
		"aicloud.dev/worker-group":        workerGroupName,
		"aicloud.dev/machine-class":       machineClassName,
	}
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
