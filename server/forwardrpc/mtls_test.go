package forwardrpc

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"net"
	"testing"
	"time"

	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestVerifyPeerCertificateRejectsUnexpectedFingerprint(t *testing.T) {
	configs.UseTestPaths(t, t.TempDir())
	if err := certutils.GenerateRootCert(); err != nil {
		t.Fatalf("GenerateRootCert failed: %v", err)
	}
	auth, expectedFingerprint, err := certutils.GenerateListenerCert("127.0.0.1", "listener-forward-fp", 5005)
	if err != nil {
		t.Fatalf("GenerateListenerCert failed: %v", err)
	}
	caCert, _, err := certutils.GetCertificateAuthority()
	if err != nil {
		t.Fatalf("GetCertificateAuthority failed: %v", err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(caCert)
	leaf := parseLeafCertificateForTest(t, []byte(auth.Certificate))

	if err := verifyPeerCertificate(roots, [][]byte{leaf.Raw}, x509.ExtKeyUsageServerAuth, expectedFingerprint); err != nil {
		t.Fatalf("verifyPeerCertificate with expected fingerprint failed: %v", err)
	}

	wrongHash := sha256.Sum256([]byte("wrong-forward-listener"))
	wrongFingerprint := hex.EncodeToString(wrongHash[:])
	if err := verifyPeerCertificate(roots, [][]byte{leaf.Raw}, x509.ExtKeyUsageServerAuth, wrongFingerprint); err == nil {
		t.Fatal("verifyPeerCertificate with wrong fingerprint succeeded")
	}
}

func TestForwardClientTLSUsesFingerprintInsteadOfServerName(t *testing.T) {
	configs.UseTestPaths(t, t.TempDir())
	if err := certutils.GenerateRootCert(); err != nil {
		t.Fatalf("GenerateRootCert failed: %v", err)
	}
	auth, expectedFingerprint, err := certutils.GenerateListenerCert("127.0.0.1", "listener-forward-fp", 5005)
	if err != nil {
		t.Fatalf("GenerateListenerCert failed: %v", err)
	}
	serverTLS, err := ServerTLSConfig(auth)
	if err != nil {
		t.Fatalf("ServerTLSConfig failed: %v", err)
	}
	grpcServer := grpc.NewServer(grpc.Creds(credentials.NewTLS(serverTLS)))
	defer grpcServer.Stop()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	go func() { _ = grpcServer.Serve(ln) }()
	t.Cleanup(func() { _ = ln.Close() })

	dialOptions, err := DialOptions("listener.example.invalid", expectedFingerprint)
	if err != nil {
		t.Fatalf("DialOptions failed: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, ln.Addr().String(), dialOptions...)
	if err != nil {
		t.Fatalf("DialContext failed with mismatched DNS name: %v", err)
	}
	_ = conn.Close()
}

func parseLeafCertificateForTest(t testing.TB, certPEM []byte) *x509.Certificate {
	t.Helper()
	cert, err := parseLeafCertificate(certPEM)
	if err != nil {
		t.Fatalf("parseLeafCertificate failed: %v", err)
	}
	return cert
}
