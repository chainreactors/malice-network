package implanttypes

import (
	"testing"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
)

func TestFromTlsPreservesCertificateSubject(t *testing.T) {
	cfg := FromTls(&clientpb.TLS{
		Enable: true,
		Acme:   true,
		Domain: "example.com",
		CertSubject: &clientpb.CertificateSubject{
			Cn: "example-cn",
			O:  "example-org",
			C:  "CN",
			L:  "Shanghai",
			Ou: "Security",
			St: "Shanghai",
		},
	})

	if cfg.Subject == nil {
		t.Fatal("expected subject to be preserved")
	}
	if cfg.Subject.CommonName != "example-cn" {
		t.Fatalf("unexpected common name: %q", cfg.Subject.CommonName)
	}
	if got := cfg.Subject.Organization; len(got) != 1 || got[0] != "example-org" {
		t.Fatalf("unexpected organization: %#v", got)
	}
	if cfg.Domain != "example.com" || !cfg.Acme || !cfg.Enable {
		t.Fatalf("unexpected tls config: %#v", cfg)
	}
}
