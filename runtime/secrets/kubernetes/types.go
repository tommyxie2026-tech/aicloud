package kubernetes

type SecretData struct {
	Namespace string
	Name      string
	Data      map[string]string
}

type ResolverConfig struct {
	AllowedNamespaces []string
}

type ResolverError struct {
	Code    string
	Message string
}

func NewResolverError(code string, message string) *ResolverError {
	return &ResolverError{Code: code, Message: message}
}

func (e *ResolverError) Error() string {
	return e.Code + ": " + e.Message
}
