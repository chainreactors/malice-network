package listener

import (
	"context"
	"errors"
	"net"
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
	"github.com/chainreactors/malice-network/server/internal/core"
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

func TestHTTPPipelineToProtobufPreservesParamsAndSecure(t *testing.T) {
	pipeline := &HTTPPipeline{
		Name:       "http-1",
		Port:       8080,
		Host:       "0.0.0.0",
		Enable:     true,
		CertName:   "cert-a",
		Headers:    map[string][]string{"X-Test": {"a", "b"}},
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

func TestBindPipelineStartReturnsForwardCreationError(t *testing.T) {
	want := errors.New("forward stream unavailable")
	pipeline, err := NewBindPipeline(&failingBindRPCClient{err: want}, &clientpb.Pipeline{
		Name:       "bind-start-fail",
		ListenerId: "listener-1",
		Enable:     false,
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

type countCloseListener struct {
	closed int
}

func (l *countCloseListener) Accept() (net.Conn, error) { return nil, net.ErrClosed }
func (l *countCloseListener) Close() error {
	l.closed++
	if l.closed > 1 {
		return net.ErrClosed
	}
	return nil
}
func (l *countCloseListener) Addr() net.Addr { return testAddr("127.0.0.1:0") }

func TestTCPPipelineCloseIsIdempotent(t *testing.T) {
	ln := &countCloseListener{}
	pipeline := &TCPPipeline{
		Name:   "tcp-close-idempotent",
		Enable: true,
		ln:     ln,
	}

	if err := pipeline.Close(); err != nil {
		t.Fatalf("first Close failed: %v", err)
	}
	if err := pipeline.Close(); err != nil {
		t.Fatalf("second Close failed: %v", err)
	}
	if pipeline.ln != nil {
		t.Fatal("pipeline listener should be cleared after Close")
	}
	if ln.closed != 1 {
		t.Fatalf("close count = %d, want 1", ln.closed)
	}
}

func TestHTTPPipelineCloseIsIdempotent(t *testing.T) {
	ln := &countCloseListener{}
	pipeline := &HTTPPipeline{
		Name:   "http-close-idempotent",
		Enable: true,
		srv:    ln,
	}

	if err := pipeline.Close(); err != nil {
		t.Fatalf("first Close failed: %v", err)
	}
	if err := pipeline.Close(); err != nil {
		t.Fatalf("second Close failed: %v", err)
	}
	if pipeline.srv != nil {
		t.Fatal("pipeline server listener should be cleared after Close")
	}
	if ln.closed != 1 {
		t.Fatalf("close count = %d, want 1", ln.closed)
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
