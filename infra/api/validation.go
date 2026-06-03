package api

import "fmt"

// ValidateManagedCluster validates the static API invariants for ManagedCluster.
func ValidateManagedCluster(cluster ManagedCluster) error {
	var errs ValidationErrors
	if cluster.Kind != "" && cluster.Kind != KindManagedCluster {
		errs = append(errs, ValidationError{Field: "kind", Message: "kind must be ManagedCluster"})
	}
	if cluster.Name == "" {
		errs = append(errs, ValidationError{Field: "metadata.name", Message: "metadata.name is required"})
	}
	if cluster.Spec.Environment == "" {
		errs = append(errs, ValidationError{Field: "spec.environment", Message: "spec.environment is required"})
	}

	seenWorkers := map[string]bool{}
	for i, worker := range cluster.Spec.Workers {
		fieldPrefix := fmt.Sprintf("spec.workers[%d]", i)
		if worker.Name == "" {
			errs = append(errs, ValidationError{Field: fieldPrefix + ".name", Message: "worker group name is required"})
		} else if seenWorkers[worker.Name] {
			errs = append(errs, ValidationError{Field: fieldPrefix + ".name", Message: "worker group name must be unique"})
		} else {
			seenWorkers[worker.Name] = true
		}
		if worker.Replicas < 0 {
			errs = append(errs, ValidationError{Field: fieldPrefix + ".replicas", Message: "replicas must be >= 0"})
		}
		if worker.MachineClassRef.Name == "" {
			errs = append(errs, ValidationError{Field: fieldPrefix + ".machineClassRef.name", Message: "machineClassRef.name is required"})
		}
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}

// ValidateMachineClass validates the static API invariants for MachineClass.
func ValidateMachineClass(machineClass MachineClass) error {
	var errs ValidationErrors
	if machineClass.Kind != "" && machineClass.Kind != KindMachineClass {
		errs = append(errs, ValidationError{Field: "kind", Message: "kind must be MachineClass"})
	}
	if machineClass.Name == "" {
		errs = append(errs, ValidationError{Field: "metadata.name", Message: "metadata.name is required"})
	}
	if machineClass.Spec.Provider == "" {
		errs = append(errs, ValidationError{Field: "spec.provider", Message: "spec.provider is required"})
	}
	if machineClass.Spec.GPU != nil {
		if machineClass.Spec.GPU.Count < 0 {
			errs = append(errs, ValidationError{Field: "spec.gpu.count", Message: "gpu.count must be >= 0"})
		}
		if machineClass.Spec.GPU.Count > 0 && machineClass.Spec.GPU.Type == "" {
			errs = append(errs, ValidationError{Field: "spec.gpu.type", Message: "gpu.type is required when gpu.count > 0"})
		}
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}

type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	message := "validation failed"
	for _, err := range e {
		message += "; " + err.Error()
	}
	return message
}
