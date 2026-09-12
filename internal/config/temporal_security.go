package config

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"
)

type TemporalSecurityConfig struct {
	TLSEnabled bool
	ServerName string
	CAFile     string
	CertFile   string
	KeyFile    string
}

type WorkflowDispatchConfig struct {
	Enabled bool
}

func LoadTemporalSecurity() TemporalSecurityConfig {
	return TemporalSecurityConfig{
		TLSEnabled: envBool("AICLOUD_TEMPORAL_TLS_ENABLED", false),
		ServerName: env("AICLOUD_TEMPORAL_TLS_SERVER_NAME", ""),
		CAFile:     env("AICLOUD_TEMPORAL_TLS_CA_FILE", ""),
		CertFile:   env("AICLOUD_TEMPORAL_TLS_CERT_FILE", ""),
		KeyFile:    env("AICLOUD_TEMPORAL_TLS_KEY_FILE", ""),
	}
}

func LoadWorkflowDispatch() WorkflowDispatchConfig {
	return WorkflowDispatchConfig{Enabled: envBool("AICLOUD_WORKFLOW_DISPATCH_ENABLED", false)}
}

func (c TemporalSecurityConfig) Validate() error {
	certFile := strings.TrimSpace(c.CertFile)
	keyFile := strings.TrimSpace(c.KeyFile)
	if !c.TLSEnabled {
		if strings.TrimSpace(c.ServerName) != "" || strings.TrimSpace(c.CAFile) != "" || certFile != "" || keyFile != "" {
			return fmt.Errorf("Temporal TLS files/server name are configured while TLS is disabled")
		}
		return nil
	}
	if (certFile == "") != (keyFile == "") {
		return fmt.Errorf("Temporal mTLS certificate and key must be configured together")
	}
	return nil
}

// ValidateAuthenticated requires an authenticated client identity suitable for
// production workflow dispatch. Server-auth TLS without a client certificate is
// useful for some development environments but is not sufficient to activate
// AI Cloud's production Task dispatcher in S3D.
func (c TemporalSecurityConfig) ValidateAuthenticated() error {
	if err := c.Validate(); err != nil {
		return err
	}
	if !c.TLSEnabled {
		return fmt.Errorf("authenticated Temporal transport requires TLS")
	}
	if strings.TrimSpace(c.CertFile) == "" || strings.TrimSpace(c.KeyFile) == "" {
		return fmt.Errorf("authenticated Temporal transport requires an mTLS client certificate and key")
	}
	return nil
}

func (c WorkflowDispatchConfig) Validate(security TemporalSecurityConfig) error {
	if !c.Enabled {
		return nil
	}
	if err := security.ValidateAuthenticated(); err != nil {
		return fmt.Errorf("workflow dispatch activation rejected: %w", err)
	}
	return nil
}

func (c TemporalSecurityConfig) TLSConfig() (*tls.Config, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	if !c.TLSEnabled {
		return nil, nil
	}

	rootCAs, err := x509.SystemCertPool()
	if err != nil || rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}
	if caFile := strings.TrimSpace(c.CAFile); caFile != "" {
		pem, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("read Temporal CA file: %w", err)
		}
		if ok := rootCAs.AppendCertsFromPEM(pem); !ok {
			return nil, fmt.Errorf("Temporal CA file contains no valid certificates")
		}
	}

	result := &tls.Config{
		MinVersion: tls.VersionTLS12,
		ServerName: strings.TrimSpace(c.ServerName),
		RootCAs:    rootCAs,
	}
	if certFile := strings.TrimSpace(c.CertFile); certFile != "" {
		certificate, err := tls.LoadX509KeyPair(certFile, strings.TrimSpace(c.KeyFile))
		if err != nil {
			return nil, fmt.Errorf("load Temporal mTLS client certificate: %w", err)
		}
		result.Certificates = []tls.Certificate{certificate}
	}
	return result, nil
}
