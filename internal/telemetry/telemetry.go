package telemetry

// Provider is an observability seam reserved for OpenTelemetry wiring.
type Provider interface{ Shutdown() error }
type NoopProvider struct{}

func (NoopProvider) Shutdown() error { return nil }
