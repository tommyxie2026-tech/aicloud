package metal3

type DesiredBareMetalHostClaim struct {
	Name             string
	Namespace        string
	ClusterName      string
	WorkerGroupName  string
	Ordinal          int
	MachineClassName string
	CPU              string
	Memory           string
	GPUProfile       string
	StorageProfile   string
	Labels           map[string]string
}

type MappingResult struct {
	HostClaims []DesiredBareMetalHostClaim
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
