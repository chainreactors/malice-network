package certutils

import (
	"crypto/x509"
	"encoding/pem"
	"path/filepath"
	"testing"

	"github.com/chainreactors/malice-network/server/internal/configs"
)

func TestGenerateListenerCertIncludesClientAndServerAuth(t *testing.T) {
	configs.UseTestPaths(t, filepath.Join(t.TempDir(), "malice"))
	if err := GenerateRootCert(); err != nil {
		t.Fatalf("GenerateRootCert failed: %v", err)
	}
	auth, _, err := GenerateListenerCert("127.0.0.1", "listener-dual-use", 5004)
	if err != nil {
		t.Fatalf("GenerateListenerCert failed: %v", err)
	}
	cert := parseTestCert(t, []byte(auth.Certificate))
	if !testHasExtKeyUsage(cert.ExtKeyUsage, x509.ExtKeyUsageClientAuth) {
		t.Fatalf("listener cert missing clientAuth: %#v", cert.ExtKeyUsage)
	}
	if !testHasExtKeyUsage(cert.ExtKeyUsage, x509.ExtKeyUsageServerAuth) {
		t.Fatalf("listener cert missing serverAuth: %#v", cert.ExtKeyUsage)
	}
}

func TestGetOrCreateForwardClientCertUsesClientAuthOnly(t *testing.T) {
	configs.UseTestPaths(t, filepath.Join(t.TempDir(), "malice"))
	if err := GenerateRootCert(); err != nil {
		t.Fatalf("GenerateRootCert failed: %v", err)
	}
	certPEM, _, err := GetOrCreateForwardClientCert("server-forward-client")
	if err != nil {
		t.Fatalf("GetOrCreateForwardClientCert failed: %v", err)
	}
	cert := parseTestCert(t, certPEM)
	if !testHasExtKeyUsage(cert.ExtKeyUsage, x509.ExtKeyUsageClientAuth) {
		t.Fatalf("forward client cert missing clientAuth: %#v", cert.ExtKeyUsage)
	}
	if testHasExtKeyUsage(cert.ExtKeyUsage, x509.ExtKeyUsageServerAuth) {
		t.Fatalf("forward client cert unexpectedly has serverAuth: %#v", cert.ExtKeyUsage)
	}
}

func parseTestCert(t testing.TB, certPEM []byte) *x509.Certificate {
	t.Helper()
	block, _ := pem.Decode(certPEM)
	if block == nil {
		t.Fatal("failed to decode cert PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("ParseCertificate failed: %v", err)
	}
	return cert
}

func testHasExtKeyUsage(usages []x509.ExtKeyUsage, want x509.ExtKeyUsage) bool {
	for _, usage := range usages {
		if usage == want {
			return true
		}
	}
	return false
}
