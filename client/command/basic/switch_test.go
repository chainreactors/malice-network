package basic

import (
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/implanttypes"
)

func TestBuildSwitchRequestHTTPPipelineIncludesRuntimeTargetConfig(t *testing.T) {
	pipeline := &clientpb.Pipeline{
		Name: "http-a",
		Type: consts.HTTPPipeline,
		Tls: &clientpb.TLS{
			Enable: true,
			Domain: "listener.example",
		},
		Encryption: []*clientpb.Encryption{
			{Type: consts.CryptorAES, Key: "http-secret"},
		},
		Body: &clientpb.Pipeline_Http{
			Http: &clientpb.HTTPPipeline{
				Host:  "127.0.0.1",
				Port:  8443,
				Proxy: "socks5://user:pass@127.0.0.2:1080",
				Params: (&implanttypes.PipelineParams{
					Headers: map[string][]string{
						"X-Test": {"ok"},
					},
				}).String(),
			},
		},
	}

	req, err := buildSwitchRequest(pipeline)
	if err != nil {
		t.Fatalf("buildSwitchRequest(http) failed: %v", err)
	}
	if req.Action != implantpb.SwitchAction_REPLACE {
		t.Fatalf("switch action = %v, want REPLACE", req.Action)
	}
	if string(req.Key) != "http-secret" {
		t.Fatalf("switch key = %q, want http-secret", string(req.Key))
	}
	if len(req.Targets) != 1 {
		t.Fatalf("switch targets len = %d, want 1", len(req.Targets))
	}

	target := req.Targets[0]
	if target.GetProtocol() != "http" || target.GetAddress() != "127.0.0.1:8443" {
		t.Fatalf("http target = %#v, want protocol http address 127.0.0.1:8443", target)
	}
	if target.GetHttpConfig() == nil || target.GetHttpConfig().GetHeaders()["X-Test"] != "ok" {
		t.Fatalf("http config = %#v, want X-Test header", target.GetHttpConfig())
	}
	if target.GetHttpConfig().GetMethod() != "POST" || target.GetHttpConfig().GetPath() != "/" || target.GetHttpConfig().GetVersion() != "1.1" {
		t.Fatalf("http config = %#v, want POST / HTTP/1.1 defaults", target.GetHttpConfig())
	}
	if target.GetTlsConfig() == nil || !target.GetTlsConfig().GetEnable() || target.GetTlsConfig().GetSni() != "listener.example" || !target.GetTlsConfig().GetSkipVerify() {
		t.Fatalf("tls config = %#v, want enabled tls with listener.example sni", target.GetTlsConfig())
	}
	if target.GetProxyConfig() == nil {
		t.Fatal("proxy config is nil, want parsed socks5 proxy")
	}
	if target.GetProxyConfig().GetType() != "socks5" || target.GetProxyConfig().GetHost() != "127.0.0.2" || target.GetProxyConfig().GetPort() != 1080 {
		t.Fatalf("proxy config = %#v, want socks5 127.0.0.2:1080", target.GetProxyConfig())
	}
	if target.GetProxyConfig().GetUsername() != "user" || target.GetProxyConfig().GetPassword() != "pass" {
		t.Fatalf("proxy credentials = %#v, want user/pass", target.GetProxyConfig())
	}
}

func TestBuildSwitchRequestREMPipelineIncludesLink(t *testing.T) {
	pipeline := &clientpb.Pipeline{
		Name: "rem-a",
		Type: consts.RemPipeline,
		Body: &clientpb.Pipeline_Rem{
			Rem: &clientpb.REM{
				Host: "127.0.0.1",
				Port: 7443,
				Link: "grpc://127.0.0.1:34996",
			},
		},
		Encryption: []*clientpb.Encryption{
			{Type: consts.CryptorRAW, Key: "ignored"},
		},
	}

	req, err := buildSwitchRequest(pipeline)
	if err != nil {
		t.Fatalf("buildSwitchRequest(rem) failed: %v", err)
	}
	if len(req.Key) != 0 {
		t.Fatalf("switch key = %q, want empty key for raw pipeline", string(req.Key))
	}
	if len(req.Targets) != 1 {
		t.Fatalf("switch targets len = %d, want 1", len(req.Targets))
	}

	target := req.Targets[0]
	if target.GetProtocol() != "rem" || target.GetAddress() != "127.0.0.1:7443" {
		t.Fatalf("rem target = %#v, want protocol rem address 127.0.0.1:7443", target)
	}
	if target.GetRemConfig() == nil || target.GetRemConfig().GetLink() != "grpc://127.0.0.1:34996" {
		t.Fatalf("rem config = %#v, want grpc link", target.GetRemConfig())
	}
}

func TestBuildSwitchRequestRejectsUnsupportedPipeline(t *testing.T) {
	pipeline := &clientpb.Pipeline{
		Name: "bind-a",
		Type: consts.BindPipeline,
		Body: &clientpb.Pipeline_Bind{
			Bind: &clientpb.BindPipeline{Name: "bind-a"},
		},
	}

	if _, err := buildSwitchRequest(pipeline); err == nil {
		t.Fatal("buildSwitchRequest(bind) succeeded, want unsupported pipeline error")
	}
}
