package cert_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/command/testsupport"
)

func TestCertCommandConformance(t *testing.T) {
	testsupport.RunClientCases(t, []testsupport.CommandCase{
		{
			Name: "list propagates server errors",
			Argv: []string{consts.CommandCert},
			Setup: func(t testing.TB, h *testsupport.Harness) {
				h.Recorder.OnCerts("GetAllCertificates", func(_ context.Context, _ any) (*clientpb.Certs, error) {
					return nil, errors.New("list failed")
				})
			},
			WantErr: "list failed",
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				testsupport.MustSingleCall[*clientpb.Empty](t, h, "GetAllCertificates")
			},
		},
		{
			Name:    "delete requires cert name",
			Argv:    []string{consts.CommandCert, consts.CommandCertDelete},
			WantErr: "accepts 1 arg(s), received 0",
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				testsupport.RequireNoPrimaryCalls(t, h)
			},
		},
		{
			Name:    "update requires cert name",
			Argv:    []string{consts.CommandCert, consts.CommandCertUpdate},
			WantErr: "accepts 1 arg(s), received 0",
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				testsupport.RequireNoPrimaryCalls(t, h)
			},
		},
		{
			Name:    "download requires cert name",
			Argv:    []string{consts.CommandCert, consts.CommandCertDownload},
			WantErr: "accepts 1 arg(s), received 0",
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				testsupport.RequireNoPrimaryCalls(t, h)
			},
		},
		{
			Name: "download propagates server errors",
			Argv: []string{consts.CommandCert, consts.CommandCertDownload, "demo-cert"},
			Setup: func(t testing.TB, h *testsupport.Harness) {
				h.Recorder.OnTLS("DownloadCertificate", func(_ context.Context, _ any) (*clientpb.TLS, error) {
					return nil, errors.New("download failed")
				})
			},
			WantErr: "download failed",
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, _ := testsupport.MustSingleCall[*clientpb.Cert](t, h, "DownloadCertificate")
				if req.Name != "demo-cert" {
					t.Fatalf("download request name = %q, want demo-cert", req.Name)
				}
			},
		},
		{
			Name: "self_signed forwards subject fields",
			Argv: []string{
				consts.CommandCert, consts.CommandCertSelfSigned,
				"--CN", "demo.example",
				"--O", "Example Org",
				"--C", "US",
				"--L", "SF",
				"--OU", "Ops",
				"--ST", "CA",
				"--validity", "730",
			},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, _ := testsupport.MustSingleCall[*clientpb.Pipeline](t, h, "GenerateSelfCert")
				if req.Tls == nil || req.Tls.CertSubject == nil {
					t.Fatalf("self_signed tls request = %#v", req)
				}
				subject := req.Tls.CertSubject
				if subject.Cn != "demo.example" || subject.O != "Example Org" || subject.C != "US" ||
					subject.L != "SF" || subject.Ou != "Ops" || subject.St != "CA" || subject.Validity != "730" {
					t.Fatalf("self_signed subject = %#v", subject)
				}
				if req.Tls.Acme {
					t.Fatalf("self_signed acme = true, want false")
				}
			},
		},
	})
}

func TestCertUpdateLoadsKeyPairWithoutCACert(t *testing.T) {
	h := testsupport.NewClientHarness(t)
	certPath, keyPath := writePEMFixture(t)

	err := h.ExecuteClient(
		consts.CommandCert, consts.CommandCertUpdate, "demo-cert",
		"--cert", certPath,
		"--key", keyPath,
		"--type", "imported",
	)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	req, _ := testsupport.MustSingleCall[*clientpb.TLS](t, h, "UpdateCertificate")
	if req.Cert == nil {
		t.Fatalf("update certificate request cert = nil")
	}
	if req.Cert.Name != "demo-cert" || req.Cert.Type != "imported" {
		t.Fatalf("update certificate metadata = %#v", req.Cert)
	}
	if strings.TrimSpace(req.Cert.Cert) == "" || strings.TrimSpace(req.Cert.Key) == "" {
		t.Fatalf("update certificate payload missing cert/key: %#v", req.Cert)
	}
	if req.Ca != nil && strings.TrimSpace(req.Ca.Cert) != "" {
		t.Fatalf("update certificate CA = %#v, want empty", req.Ca)
	}
}

func TestCertUpdateRejectsPartialKeyPair(t *testing.T) {
	h := testsupport.NewClientHarness(t)
	certPath, _ := writePEMFixture(t)

	err := h.ExecuteClient(
		consts.CommandCert, consts.CommandCertUpdate, "demo-cert",
		"--cert", certPath,
	)
	if err == nil || !strings.Contains(err.Error(), "cert and key must be provided together") {
		t.Fatalf("update error = %v, want partial key-pair validation", err)
	}
	testsupport.RequireNoPrimaryCalls(t, h)
}

func TestAcmeConfigCmdMergesExistingState(t *testing.T) {
	h := testsupport.NewClientHarness(t)
	getCount := 0
	h.Recorder.OnAcmeConfig("GetAcmeConfig", func(_ context.Context, _ any) (*clientpb.AcmeConfig, error) {
		getCount++
		if getCount == 1 {
			return &clientpb.AcmeConfig{
				Email:       "old@example.com",
				CaUrl:       "https://old-ca",
				Provider:    "cloudflare",
				Credentials: map[string]string{"api_token": "old-token"},
			}, nil
		}
		return &clientpb.AcmeConfig{
			Email:       "new@example.com",
			CaUrl:       "https://old-ca",
			Provider:    "cloudflare",
			Credentials: map[string]string{"api_token": "old-token"},
		}, nil
	})

	err := h.ExecuteClient(
		consts.CommandCert, consts.CommandCertAcmeConfig,
		"--email", "new@example.com",
	)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	calls := h.Recorder.Calls()
	if len(calls) != 3 {
		t.Fatalf("call count = %d, want 3", len(calls))
	}
	if calls[0].Method != "GetAcmeConfig" || calls[1].Method != "UpdateAcmeConfig" || calls[2].Method != "GetAcmeConfig" {
		t.Fatalf("call methods = %#v", calls)
	}
	update, ok := calls[1].Request.(*clientpb.AcmeConfig)
	if !ok {
		t.Fatalf("update request type = %T, want *clientpb.AcmeConfig", calls[1].Request)
	}
	if update.Email != "new@example.com" || update.CaUrl != "https://old-ca" || update.Provider != "cloudflare" {
		t.Fatalf("merged acme config = %#v", update)
	}
	if update.Credentials["api_token"] != "old-token" {
		t.Fatalf("merged credentials = %#v", update.Credentials)
	}
}

func writePEMFixture(t testing.TB) (string, string) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   "demo.example",
			Organization: []string{"Example Org"},
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}

	dir := t.TempDir()
	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")

	if err := os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}), 0o600); err != nil {
		t.Fatalf("write cert: %v", err)
	}
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}), 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}

	return certPath, keyPath
}
