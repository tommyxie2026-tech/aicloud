package config

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWorkflowDispatchDisabledByDefault(t *testing.T) {
	t.Setenv("AICLOUD_WORKFLOW_DISPATCH_ENABLED", "")
	if LoadWorkflowDispatch().Enabled {
		t.Fatal("workflow dispatch must be disabled by default")
	}
}

func TestWorkflowDispatchRequiresAuthenticatedTemporalTransport(t *testing.T) {
	dispatch := WorkflowDispatchConfig{Enabled: true}
	if err := dispatch.Validate(TemporalSecurityConfig{}); err == nil {
		t.Fatal("dispatch accepted unauthenticated Temporal transport")
	}
	if err := dispatch.Validate(TemporalSecurityConfig{TLSEnabled: true}); err == nil {
		t.Fatal("dispatch accepted server-auth TLS without worker client identity")
	}
}

func TestTemporalSecurityRejectsPartialOrDisabledTLSMaterial(t *testing.T) {
	cases := []TemporalSecurityConfig{
		{TLSEnabled: false, CAFile: "ca.pem"},
		{TLSEnabled: true, CertFile: "client.pem"},
		{TLSEnabled: true, KeyFile: "client.key"},
	}
	for _, config := range cases {
		if err := config.Validate(); err == nil {
			t.Fatalf("invalid Temporal TLS config accepted: %+v", config)
		}
	}
}

func TestTemporalSecurityBuildsMTLSConfig(t *testing.T) {
	caFile, certFile, keyFile := writeTemporalTestCertificate(t)
	security := TemporalSecurityConfig{
		TLSEnabled: true,
		ServerName: "temporal.internal",
		CAFile:     caFile,
		CertFile:   certFile,
		KeyFile:    keyFile,
	}
	if err := security.ValidateAuthenticated(); err != nil {
		t.Fatalf("valid mTLS config rejected: %v", err)
	}
	tlsConfig, err := security.TLSConfig()
	if err != nil {
		t.Fatal(err)
	}
	if tlsConfig == nil || tlsConfig.ServerName != "temporal.internal" || len(tlsConfig.Certificates) != 1 || tlsConfig.RootCAs == nil {
		t.Fatalf("unexpected TLS config: %+v", tlsConfig)
	}
	if err := (WorkflowDispatchConfig{Enabled: true}).Validate(security); err != nil {
		t.Fatalf("authenticated dispatch config rejected: %v", err)
	}
}

func TestTemporalTLSConfigDisabledReturnsNil(t *testing.T) {
	config, err := (TemporalSecurityConfig{}).TLSConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config != nil {
		t.Fatalf("disabled TLS returned config: %+v", config)
	}
}

func writeTemporalTestCertificate(t *testing.T) (string, string, string) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "temporal.internal"},
		DNSNames:     []string{"temporal.internal"},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		IsCA:         true,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	caFile := filepath.Join(dir, "ca.pem")
	certFile := filepath.Join(dir, "client.pem")
	keyFile := filepath.Join(dir, "client.key")
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	if err := os.WriteFile(caFile, certPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(certFile, certPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyFile, keyPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	return caFile, certFile, keyFile
}
