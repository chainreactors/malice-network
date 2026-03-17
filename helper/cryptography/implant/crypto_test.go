package cryptography

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"strings"
	"testing"
	"time"
)

func TestRootOnlyVerifyCertificateRejectsInvalidCAPEM(t *testing.T) {
	err := RootOnlyVerifyCertificate("not-a-cert", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "failed to parse root certificate") {
		t.Fatalf("RootOnlyVerifyCertificate(invalid ca) error = %v, want parse failure", err)
	}
}

func TestRootOnlyVerifyCertificateRejectsEmptyPeerChain(t *testing.T) {
	caPEM, _ := generateRootAndLeafPEM(t)

	err := RootOnlyVerifyCertificate(caPEM, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "no peer certificate") {
		t.Fatalf("RootOnlyVerifyCertificate(empty chain) error = %v, want empty chain failure", err)
	}
}

func TestRootOnlyVerifyCertificateAcceptsCertificateSignedByRoot(t *testing.T) {
	caPEM, leafDER := generateRootAndLeafPEM(t)

	if err := RootOnlyVerifyCertificate(caPEM, [][]byte{leafDER}, nil); err != nil {
		t.Fatalf("RootOnlyVerifyCertificate(valid chain) failed: %v", err)
	}
}

func generateRootAndLeafPEM(t *testing.T) (string, []byte) {
	t.Helper()

	now := time.Now()
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey(ca) failed: %v", err)
	}
	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "malice-test-ca",
		},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("CreateCertificate(ca) failed: %v", err)
	}
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})

	leafKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey(leaf) failed: %v", err)
	}
	leafTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			CommonName: "malice-test-leaf",
		},
		NotBefore:   now.Add(-time.Hour),
		NotAfter:    now.Add(time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTemplate, caTemplate, &leafKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("CreateCertificate(leaf) failed: %v", err)
	}

	return string(caPEM), leafDER
}
