package clusterapi

type DesiredCluster struct {
	Name        string
	Namespace   string
	Environment string
	Labels      map[string]string
}

type DesiredMachineDeployment struct {
	Name             string
	Namespace        string
	ClusterName      string
	WorkerGroupName  string
	Replicas         int
	MachineClassName string
	Labels           map[string]string
}

type MappingResult struct {
	Cluster            DesiredCluster
	MachineDeployments []DesiredMachineDeployment
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
