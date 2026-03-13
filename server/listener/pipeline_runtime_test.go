package listener

import (
	"context"
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/helper/implanttypes"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"google.golang.org/grpc"
)

type failingBindRPCClient struct {
	err error
}

func (c *failingBindRPCClient) SpiteStream(context.Context, ...grpc.CallOption) (listenerrpc.ListenerRPC_SpiteStreamClient, error) {
	return nil, c.err
}

func (c *failingBindRPCClient) Register(context.Context, *clientpb.RegisterSession, ...grpc.CallOption) (*clientpb.Empty, error) {
	return &clientpb.Empty{}, nil
}

func (c *failingBindRPCClient) Checkin(context.Context, *implantpb.Ping, ...grpc.CallOption) (*clientpb.Empty, error) {
	return &clientpb.Empty{}, nil
}

func TestNewHTTPPipelinePreservesConfigFromProtobuf(t *testing.T) {
	pb := &clientpb.Pipeline{
		Name:       "http-main",
		ListenerId: "listener-1",
		Enable:     true,
		Parser:     "http-parser",
		Tls: &clientpb.TLS{
			Enable: true,
			Cert: &clientpb.Cert{
				Cert: "cert-data",
				Key:  "key-data",
			},
		},
		Encryption: []*clientpb.Encryption{
			{Type: consts.CryptorAES, Key: "aes-key"},
		},
		Secure: &clientpb.Secure{
			Enable: true,
			ServerKeypair: &clientpb.KeyPair{
				PublicKey:  "spub",
				PrivateKey: "spriv",
			},
		},
		Body: &clientpb.Pipeline_Http{
			Http: &clientpb.HTTPPipeline{
				Host: "0.0.0.0",
				Port: 8080,
				Params: (&implanttypes.PipelineParams{
					Headers:    map[string][]string{"X-Test": {"a", "b"}},
					ErrorPage:  "err-page",
					BodyPrefix: "prefix",
					BodySuffix: "suffix",
				}).String(),
			},
		},
	}

	pipeline, err := NewHttpPipeline(nil, pb)
	if err != nil {
		t.Fatalf("NewHttpPipeline failed: %v", err)
	}

	if pipeline.Name != "http-main" || pipeline.Host != "0.0.0.0" || pipeline.Port != 8080 {
		t.Fatalf("unexpected http runtime config: %#v", pipeline)
	}
	if pipeline.Parser != "http-parser" || pipeline.ListenerID != "listener-1" {
		t.Fatalf("unexpected pipeline metadata: %#v", pipeline.PipelineConfig)
	}
	if string(pipeline.ErrorPage) != "err-page" || string(pipeline.BodyPrefix) != "prefix" || string(pipeline.BodySuffix) != "suffix" {
		t.Fatalf("unexpected http body config: error=%q prefix=%q suffix=%q", pipeline.ErrorPage, pipeline.BodyPrefix, pipeline.BodySuffix)
	}
	if len(pipeline.Headers["X-Test"]) != 2 {
		t.Fatalf("unexpected headers: %#v", pipeline.Headers)
	}
	if pipeline.TLSConfig == nil || !pipeline.TLSConfig.Enable || pipeline.TLSConfig.Cert == nil || pipeline.TLSConfig.Cert.Cert != "cert-data" {
		t.Fatalf("unexpected TLS config: %#v", pipeline.TLSConfig)
	}
	if pipeline.SecureConfig == nil || !pipeline.SecureConfig.Enable || pipeline.SecureConfig.ServerPrivateKey != "spriv" {
		t.Fatalf("unexpected secure config: %#v", pipeline.SecureConfig)
	}
	if len(pipeline.Encryption) != 1 || pipeline.Encryption[0].Key != "aes-key" {
		t.Fatalf("unexpected encryption config: %#v", pipeline.Encryption)
	}
}

func TestNewTCPPipelinePreservesConfigFromProtobuf(t *testing.T) {
	pb := &clientpb.Pipeline{
		Name:       "tcp-main",
		ListenerId: "listener-1",
		Enable:     true,
		Parser:     "tcp-parser",
		Tls: &clientpb.TLS{
			Enable: true,
			Cert: &clientpb.Cert{
				Cert: "cert-data",
				Key:  "key-data",
			},
		},
		Encryption: []*clientpb.Encryption{
			{Type: consts.CryptorXOR, Key: "xor-key"},
		},
		Secure: &clientpb.Secure{
			Enable: true,
			ImplantKeypair: &clientpb.KeyPair{
				PublicKey:  "ipub",
				PrivateKey: "ipriv",
			},
		},
		Body: &clientpb.Pipeline_Tcp{
			Tcp: &clientpb.TCPPipeline{
				Host: "127.0.0.1",
				Port: 5001,
			},
		},
	}

	pipeline, err := NewTcpPipeline(nil, pb)
	if err != nil {
		t.Fatalf("NewTcpPipeline failed: %v", err)
	}

	if pipeline.Name != "tcp-main" || pipeline.Host != "127.0.0.1" || pipeline.Port != 5001 {
		t.Fatalf("unexpected tcp runtime config: %#v", pipeline)
	}
	if pipeline.Parser != "tcp-parser" || pipeline.ListenerID != "listener-1" {
		t.Fatalf("unexpected pipeline metadata: %#v", pipeline.PipelineConfig)
	}
	if pipeline.TLSConfig == nil || !pipeline.TLSConfig.Enable || pipeline.TLSConfig.Cert == nil || pipeline.TLSConfig.Cert.Key != "key-data" {
		t.Fatalf("unexpected TLS config: %#v", pipeline.TLSConfig)
	}
	if pipeline.SecureConfig == nil || !pipeline.SecureConfig.Enable || pipeline.SecureConfig.ImplantPrivateKey != "ipriv" {
		t.Fatalf("unexpected secure config: %#v", pipeline.SecureConfig)
	}
	if len(pipeline.Encryption) != 1 || pipeline.Encryption[0].Type != consts.CryptorXOR {
		t.Fatalf("unexpected encryption config: %#v", pipeline.Encryption)
	}
}

func TestNewBindPipelinePreservesEnableStateAndConfig(t *testing.T) {
	pb := &clientpb.Pipeline{
		Name:       "bind-main",
		ListenerId: "listener-1",
		Enable:     false,
		Parser:     consts.ImplantMalefic,
		Tls: &clientpb.TLS{
			Enable: true,
			Cert: &clientpb.Cert{
				Cert: "cert-data",
				Key:  "key-data",
			},
		},
		Encryption: []*clientpb.Encryption{
			{Type: consts.CryptorAES, Key: "bind-key"},
		},
		Body: &clientpb.Pipeline_Bind{
			Bind: &clientpb.BindPipeline{},
		},
	}

	pipeline, err := NewBindPipeline(nil, pb)
	if err != nil {
		t.Fatalf("NewBindPipeline failed: %v", err)
	}

	if pipeline.Enable {
		t.Fatalf("bind runtime enable state should follow protobuf, got %#v", pipeline)
	}
	if pipeline.TLSConfig == nil || !pipeline.TLSConfig.Enable || pipeline.TLSConfig.Cert == nil || pipeline.TLSConfig.Cert.Cert != "cert-data" {
		t.Fatalf("unexpected TLS config: %#v", pipeline.TLSConfig)
	}
	if len(pipeline.Encryption) != 1 || pipeline.Encryption[0].Key != "bind-key" {
		t.Fatalf("unexpected encryption config: %#v", pipeline.Encryption)
	}
}

func TestBindPipelineStartReturnsForwardCreationError(t *testing.T) {
	want := errors.New("forward stream unavailable")
	pipeline, err := NewBindPipeline(&failingBindRPCClient{err: want}, &clientpb.Pipeline{
		Name:       "bind-start-fail",
		ListenerId: "listener-1",
		Enable:     true,
		Body: &clientpb.Pipeline_Bind{
			Bind: &clientpb.BindPipeline{},
		},
	})
	if err != nil {
		t.Fatalf("NewBindPipeline failed: %v", err)
	}

	err = pipeline.Start()
	if !errors.Is(err, want) {
		t.Fatalf("Start error = %v, want %v", err, want)
	}
}

func TestRegisterAndStartSkipsDisabledPipeline(t *testing.T) {
	lns := &listener{
		Name: "listener-1",
		cfg:  &configs.ListenerConfig{},
	}

	err := lns.RegisterAndStart(&clientpb.Pipeline{
		Name:   "disabled-bind",
		Enable: false,
		Body: &clientpb.Pipeline_Bind{
			Bind: &clientpb.BindPipeline{},
		},
	})
	if err != nil {
		t.Fatalf("RegisterAndStart should skip disabled pipeline, got %v", err)
	}
}

func TestWebsiteAddContentAndHandlerServeConfiguredPath(t *testing.T) {
	root := filepath.Join(t.TempDir(), "website-root")
	oldWebsitePath := configs.WebsitePath
	configs.WebsitePath = root
	t.Cleanup(func() {
		configs.WebsitePath = oldWebsitePath
	})

	web := &Website{
		Name:     "site-1",
		rootPath: "/site",
		Content:  make(map[string]*clientpb.WebContent),
		Artifact: make(map[string]*clientpb.WebContent),
	}

	content := &clientpb.WebContent{
		Id:          "content-1",
		WebsiteId:   "site-1",
		Path:        "/index.html",
		ContentType: "text/html",
		Content:     []byte("<html>ok</html>"),
	}
	if err := web.AddContent(content); err != nil {
		t.Fatalf("AddContent failed: %v", err)
	}

	storedPath := filepath.Join(root, "site-1", "content-1")
	data, err := os.ReadFile(storedPath)
	if err != nil {
		t.Fatalf("failed to read stored content: %v", err)
	}
	if string(data) != "<html>ok</html>" {
		t.Fatalf("unexpected stored content: %q", data)
	}
	if got := web.Content["index.html"]; got == nil || got.ContentType != "text/html" {
		t.Fatalf("unexpected website content map: %#v", web.Content)
	}

	req := httptest.NewRequest("GET", "http://example.com/site/index.html", nil)
	resp := httptest.NewRecorder()
	web.websiteContentHandler(resp, req)

	if resp.Code != 200 {
		t.Fatalf("unexpected status code: %d", resp.Code)
	}
	if body := resp.Body.String(); body != "<html>ok</html>" {
		t.Fatalf("unexpected body: %q", body)
	}
	if ctype := resp.Header().Get("Content-Type"); ctype != "text/html" {
		t.Fatalf("unexpected content type: %q", ctype)
	}
}

func TestHandleWebContentUpdateReturnsContentErrors(t *testing.T) {
	lns := &listener{
		websites: map[string]*Website{
			"site-1": {
				Content:  make(map[string]*clientpb.WebContent),
				Artifact: make(map[string]*clientpb.WebContent),
			},
		},
	}

	err := lns.handleWebContentUpdate(&clientpb.JobCtrl{
		Job: &clientpb.Job{
			Pipeline: &clientpb.Pipeline{
				Name: "site-1",
				Body: &clientpb.Pipeline_Web{
					Web: &clientpb.Website{},
				},
			},
		},
		Content: &clientpb.WebContent{
			Path: "/broken",
		},
	})
	if err == nil {
		t.Fatal("expected content validation error")
	}
}

func TestNewHttpPipelineRejectsInvalidParams(t *testing.T) {
	_, err := NewHttpPipeline(nil, &clientpb.Pipeline{
		Name:       "http-bad",
		ListenerId: "listener-1",
		Body: &clientpb.Pipeline_Http{
			Http: &clientpb.HTTPPipeline{
				Host:   "0.0.0.0",
				Port:   8080,
				Params: "{bad-json",
			},
		},
	})
	if err == nil {
		t.Fatal("expected invalid params error")
	}
}

func TestNewRemPrefersLinkAndPreservesRuntimeFields(t *testing.T) {
	pb := &clientpb.Pipeline{
		Name:       "rem-main",
		ListenerId: "listener-1",
		Enable:     true,
		Ip:         "10.0.0.8",
		Parser:     "rem-parser",
		Body: &clientpb.Pipeline_Rem{
			Rem: &clientpb.REM{
				Console: "tcp://127.0.0.1:9001",
				Link:    "tcp://127.0.0.1:9002",
			},
		},
	}

	rem, err := NewRem(nil, pb)
	if err != nil {
		t.Fatalf("NewRem failed: %v", err)
	}

	if rem.Name != "rem-main" || rem.ListenerID != "listener-1" {
		t.Fatalf("unexpected rem identity: %#v", rem)
	}
	if rem.remConfig.Link != "tcp://127.0.0.1:9002" || rem.remConfig.Console != "tcp://127.0.0.1:9001" {
		t.Fatalf("unexpected rem config: %#v", rem.remConfig)
	}
	out := rem.ToProtobuf()
	if out.GetRem().Console != "tcp://127.0.0.1:9001" || out.GetRem().Link == "" {
		t.Fatalf("unexpected rem protobuf: %#v", out.GetRem())
	}
}
