package kubevirt

type DesiredVirtualMachine struct {
	Name             string
	Namespace        string
	ClusterName      string
	WorkerGroupName  string
	Ordinal          int32
	MachineClassName string
	CPU              string
	Memory           string
	GPUProfile       string
	Labels           map[string]string
}

type MappingResult struct {
	VirtualMachines []DesiredVirtualMachine
}

type MappingError struct {
	Code    string
	Message string
}

func NewMappingError(code string, message string) *MappingError {
	return &MappingError{Code: code, Message: message}
}

func (e *MappingError) Error() string {
	return e.Code + ": " + e.Message
}
