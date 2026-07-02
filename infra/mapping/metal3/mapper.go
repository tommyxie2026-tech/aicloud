package metal3

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
		for ordinal := int32(1); ordinal <= worker.Replicas; ordinal++ {
			result.HostClaims = append(result.HostClaims, DesiredBareMetalHostClaim{
				Name:             hostClaimName(cluster.Name, worker.Name, ordinal),
				Namespace:        cluster.Namespace,
				ClusterName:      cluster.Name,
				WorkerGroupName:  worker.Name,
				Ordinal:          ordinal,
				MachineClassName: class.Name,
				CPU:              class.Spec.CPU,
				Memory:           class.Spec.Memory,
				GPUProfile:       gpuProfile(class.Spec.GPU),
				Labels:           baseLabels(cluster.Name, cluster.Spec.Environment, worker.Name, class.Name),
			})
		}
	}
	return result, nil
}

func indexMachineClasses(classes []api.MachineClass) map[string]api.MachineClass {
	index := map[string]api.MachineClass{}
	for _, class := range classes {
		index[class.Name] = class
	}
	return index
}

func gpuProfile(gpu *api.GPUSpec) string {
	if gpu == nil || gpu.Count == 0 {
		return ""
	}
	return fmt.Sprintf("%dx%s", gpu.Count, gpu.Type)
}

func hostClaimName(clusterName string, workerGroupName string, ordinal int32) string {
	return fmt.Sprintf("%s-%s-host-%04d", sanitizeName(clusterName), sanitizeName(workerGroupName), ordinal)
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
