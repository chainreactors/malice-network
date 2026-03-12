package configs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/malice-network/helper/implanttypes"
)

func TestTlsConfigReadCertNil(t *testing.T) {
	var cfg *TlsConfig

	tls, err := cfg.ReadCert()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tls == nil {
		t.Fatal("expected non-nil tls config")
	}
	if tls.Enable {
		t.Fatal("expected disabled tls config for nil input")
	}
	if tls.Cert != nil || tls.CA != nil || tls.Subject != nil {
		t.Fatal("expected empty tls config for nil input")
	}
}

func TestTlsConfigReadCertLoadsFilesAndSubject(t *testing.T) {
	dir := t.TempDir()
	certPath := writeTempFile(t, dir, "cert.pem", "cert-data")
	keyPath := writeTempFile(t, dir, "key.pem", "key-data")
	caPath := writeTempFile(t, dir, "ca.pem", "ca-data")

	cfg := &TlsConfig{
		Enable:   true,
		CertFile: certPath,
		KeyFile:  keyPath,
		CAFile:   caPath,
		CN:       "example-cn",
		O:        "example-org",
		C:        "CN",
		L:        "Shanghai",
		OU:       "Security",
		ST:       "Shanghai",
	}

	tls, err := cfg.ReadCert()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !tls.Enable {
		t.Fatal("expected tls to be enabled")
	}
	if tls.Subject == nil {
		t.Fatal("expected subject to be populated")
	}
	if tls.Subject.CommonName != "example-cn" {
		t.Fatalf("unexpected CN: %q", tls.Subject.CommonName)
	}
	if tls.Cert == nil || tls.Cert.Cert != "cert-data" || tls.Cert.Key != "key-data" {
		t.Fatalf("unexpected cert payload: %#v", tls.Cert)
	}
	if tls.CA == nil || tls.CA.Cert != "ca-data" {
		t.Fatalf("unexpected ca payload: %#v", tls.CA)
	}
}

func TestHttpPipelineConfigToProtobufSerializesParams(t *testing.T) {
	dir := t.TempDir()
	errorPage := writeTempFile(t, dir, "error.html", "<html>fail</html>")

	cfg := &HttpPipelineConfig{
		Enable:     true,
		Name:       "http-main",
		Host:       "127.0.0.1",
		Port:       8080,
		Parser:     consts.ImplantMalefic,
		Headers:    map[string][]string{"X-Test": []string{"a", "b"}},
		ErrorPage:  errorPage,
		BodyPrefix: "pre",
		BodySuffix: "suf",
		EncryptionConfig: implanttypes.EncryptionsConfig{
			&implanttypes.EncryptionConfig{Type: consts.CryptorAES, Key: "secret"},
		},
		SecureConfig: &implanttypes.SecureConfig{Enable: true},
	}

	pb, err := cfg.ToProtobuf("listener-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pb.Type != consts.HTTPPipeline {
		t.Fatalf("unexpected pipeline type: %q", pb.Type)
	}
	if pb.Tls == nil || pb.Tls.Enable {
		t.Fatalf("expected disabled tls protobuf, got %#v", pb.Tls)
	}
	if len(pb.Encryption) != 1 || pb.Encryption[0].Key != "secret" {
		t.Fatalf("unexpected encryption protobuf: %#v", pb.Encryption)
	}

	params, err := implanttypes.UnmarshalPipelineParams(pb.GetHttp().Params)
	if err != nil {
		t.Fatalf("failed to unmarshal params: %v", err)
	}
	if params.ErrorPage != "<html>fail</html>" {
		t.Fatalf("unexpected error page: %q", params.ErrorPage)
	}
	if params.BodyPrefix != "pre" || params.BodySuffix != "suf" {
		t.Fatalf("unexpected body wrappers: %#v", params)
	}
	if len(params.Headers["X-Test"]) != 2 {
		t.Fatalf("unexpected headers: %#v", params.Headers)
	}
}

func TestTcpPipelineConfigToProtobufIncludesTLSAndSecure(t *testing.T) {
	dir := t.TempDir()
	certPath := writeTempFile(t, dir, "cert.pem", "cert-data")
	keyPath := writeTempFile(t, dir, "key.pem", "key-data")

	cfg := &TcpPipelineConfig{
		Enable: true,
		Name:   "tcp-main",
		Host:   "0.0.0.0",
		Port:   5001,
		Parser: consts.ImplantMalefic,
		TlsConfig: &TlsConfig{
			Enable:   true,
			CertFile: certPath,
			KeyFile:  keyPath,
		},
		EncryptionConfig: implanttypes.EncryptionsConfig{
			&implanttypes.EncryptionConfig{Type: consts.CryptorXOR, Key: "key123"},
		},
		SecureConfig: &implanttypes.SecureConfig{
			Enable:           true,
			ServerPublicKey:  "spub",
			ServerPrivateKey: "spri",
		},
	}

	pb, err := cfg.ToProtobuf("listener-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pb.Tls == nil || !pb.Tls.Enable {
		t.Fatalf("expected enabled tls protobuf, got %#v", pb.Tls)
	}
	if pb.Tls.Cert == nil || pb.Tls.Cert.Cert != "cert-data" || pb.Tls.Cert.Key != "key-data" {
		t.Fatalf("unexpected tls cert payload: %#v", pb.Tls)
	}
	if pb.Secure == nil || !pb.Secure.Enable {
		t.Fatalf("unexpected secure protobuf: %#v", pb.Secure)
	}
	if len(pb.Encryption) != 1 || pb.Encryption[0].Type != consts.CryptorXOR {
		t.Fatalf("unexpected encryption protobuf: %#v", pb.Encryption)
	}
}

func writeTempFile(t *testing.T, dir, name, content string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write temp file %s: %v", name, err)
	}
	return path
}
