package listener

import (
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/malice-network/helper/implanttypes"
	"github.com/chainreactors/malice-network/server/internal/core"
)

func TestHTTPPipelineToProtobufPreservesParamsAndSecure(t *testing.T) {
	pipeline := &HTTPPipeline{
		Name:       "http-1",
		Port:       8080,
		Host:       "0.0.0.0",
		Enable:     true,
		CertName:   "cert-a",
		Headers:    map[string][]string{"X-Test": []string{"a", "b"}},
		ErrorPage:  []byte("err"),
		BodyPrefix: []byte("pre"),
		BodySuffix: []byte("suf"),
		PipelineConfig: &core.PipelineConfig{
			ListenerID: "listener-1",
			Parser:     "auto",
			TLSConfig:  &implanttypes.TlsConfig{Enable: true, Domain: "example.com"},
			Encryption: implanttypes.EncryptionsConfig{
				&implanttypes.EncryptionConfig{Type: consts.CryptorAES, Key: "aes-key"},
			},
			SecureConfig: &implanttypes.SecureConfig{
				Enable:           true,
				ServerPublicKey:  "spub",
				ServerPrivateKey: "spriv",
			},
		},
	}

	pb := pipeline.ToProtobuf()
	params, err := implanttypes.UnmarshalPipelineParams(pb.GetHttp().Params)
	if err != nil {
		t.Fatalf("failed to unmarshal http params: %v", err)
	}
	if params.ErrorPage != "err" || params.BodyPrefix != "pre" || params.BodySuffix != "suf" {
		t.Fatalf("http params not preserved: %#v", params)
	}
	if len(params.Headers["X-Test"]) != 2 {
		t.Fatalf("http headers not preserved: %#v", params.Headers)
	}
	if pb.Secure == nil || !pb.Secure.Enable || pb.Secure.ServerKeypair.PrivateKey != "spriv" {
		t.Fatalf("secure config not preserved: %#v", pb.Secure)
	}
}

func TestWebsiteToProtobufPreservesCommonFields(t *testing.T) {
	website := &Website{
		Name:     "website-1",
		Enable:   true,
		CertName: "cert-a",
		port:     8081,
		rootPath: "/static",
		PipelineConfig: &core.PipelineConfig{
			ListenerID: "listener-1",
			Parser:     "auto",
			TLSConfig:  &implanttypes.TlsConfig{Enable: true},
			Encryption: implanttypes.EncryptionsConfig{
				&implanttypes.EncryptionConfig{Type: consts.CryptorXOR, Key: "xor-key"},
			},
			SecureConfig: &implanttypes.SecureConfig{
				Enable: true,
			},
		},
	}

	pb := website.ToProtobuf()
	if pb.Parser != "auto" {
		t.Fatalf("parser not preserved: %#v", pb)
	}
	if len(pb.Encryption) != 1 || pb.Encryption[0].Key != "xor-key" {
		t.Fatalf("encryption not preserved: %#v", pb.Encryption)
	}
	if pb.Secure == nil || !pb.Secure.Enable {
		t.Fatalf("secure config not preserved: %#v", pb.Secure)
	}
}
