package workflow

import (
	"crypto/tls"
	"testing"
	"time"
)

func TestTemporalClientOptionsPropagateTLS(t *testing.T) {
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12, ServerName: "temporal.internal"}
	config := WorkerConfig{
		Enabled:     true,
		Address:     " temporal.internal:7233 ",
		Namespace:   " production ",
		TaskQueue:   "aicloud-task-v1",
		StopTimeout: 30 * time.Second,
		TLSConfig:   tlsConfig,
	}
	options := temporalClientOptions(config)
	if options.HostPort != "temporal.internal:7233" || options.Namespace != "production" {
		t.Fatalf("unexpected client options host=%q namespace=%q", options.HostPort, options.Namespace)
	}
	if options.ConnectionOptions.TLS != tlsConfig {
		t.Fatal("Temporal TLS config was not propagated to the SDK client")
	}
}
