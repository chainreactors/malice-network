package website

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	iomclient "github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/IoM-go/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

type websiteRecordedCall struct {
	method  string
	request any
}

type websiteTestRPC struct {
	clientrpc.MaliceRPCClient
	listenerrpc.ListenerRPCClient

	calls []websiteRecordedCall
}

func newWebsiteTestConsole(rpc *websiteTestRPC) *core.Console {
	state := &iomclient.ServerState{
		Rpc: &iomclient.Rpc{
			MaliceRPCClient:   rpc,
			ListenerRPCClient: rpc,
		},
		Client:    &clientpb.Client{Name: "tester", ID: 1},
		Pipelines: map[string]*clientpb.Pipeline{},
	}
	return &core.Console{
		Server: &core.Server{ServerState: state},
		Log:    iomclient.Log,
	}
}

func (r *websiteTestRPC) record(method string, request any) {
	r.calls = append(r.calls, websiteRecordedCall{method: method, request: request})
}

func (r *websiteTestRPC) AddWebsiteContent(ctx context.Context, in *clientpb.Website, opts ...grpc.CallOption) (*clientpb.WebContent, error) {
	r.record("AddWebsiteContent", in)
	for _, content := range in.GetContents() {
		return content, nil
	}
	return &clientpb.WebContent{}, nil
}

func (r *websiteTestRPC) UpdateWebsiteContent(ctx context.Context, in *clientpb.WebContent, opts ...grpc.CallOption) (*clientpb.WebContent, error) {
	r.record("UpdateWebsiteContent", in)
	return in, nil
}

func (r *websiteTestRPC) UpdateWebsiteContentMetadata(ctx context.Context, in *clientpb.WebContent, opts ...grpc.CallOption) (*clientpb.WebContent, error) {
	r.record("UpdateWebsiteContentMetadata", in)
	return in, nil
}

func (r *websiteTestRPC) DownloadArtifact(ctx context.Context, in *clientpb.Artifact, opts ...grpc.CallOption) (*clientpb.Artifact, error) {
	r.record("DownloadArtifact", in)
	return &clientpb.Artifact{
		Name:   in.Name,
		Format: in.Format,
		Bin:    []byte("artifact-binary"),
	}, nil
}

func (r *websiteTestRPC) ListWebContent(ctx context.Context, in *clientpb.Website, opts ...grpc.CallOption) (*clientpb.WebContents, error) {
	r.record("ListWebContent", in)
	return &clientpb.WebContents{}, nil
}

func (r *websiteTestRPC) StopWebsite(ctx context.Context, in *clientpb.CtrlPipeline, opts ...grpc.CallOption) (*clientpb.Empty, error) {
	r.record("StopWebsite", in)
	return &clientpb.Empty{}, nil
}

func (r *websiteTestRPC) RegisterWebsite(ctx context.Context, in *clientpb.Pipeline, opts ...grpc.CallOption) (*clientpb.Empty, error) {
	r.record("RegisterWebsite", in)
	return &clientpb.Empty{}, nil
}

func (r *websiteTestRPC) StartWebsite(ctx context.Context, in *clientpb.CtrlPipeline, opts ...grpc.CallOption) (*clientpb.Empty, error) {
	r.record("StartWebsite", in)
	return &clientpb.Empty{}, nil
}

func (r *websiteTestRPC) UpdateWebsiteTLS(ctx context.Context, in *clientpb.PipelineTLSUpdate, opts ...grpc.CallOption) (*clientpb.Pipeline, error) {
	r.record("UpdateWebsiteTLS", in)
	return &clientpb.Pipeline{Name: in.Name, ListenerId: in.ListenerId}, nil
}

func TestAddWebContentDirectUsesScopedWebsiteCacheKey(t *testing.T) {
	rpc := &websiteTestRPC{}
	con := newWebsiteTestConsole(rpc)
	con.Pipelines["listener-a:site-shared"] = scopedWebsitePipeline("site-shared", "listener-a")

	if err := AddWebContentDirect(con, "listener-a:site-shared", []byte("body"), "/index.html", "text/html"); err != nil {
		t.Fatalf("AddWebContentDirect failed: %v", err)
	}
	req := onlyWebsiteCall[*clientpb.Website](t, rpc, "AddWebsiteContent")
	if req.Name != "site-shared" || req.ListenerId != "listener-a" {
		t.Fatalf("website request = %s/%s, want site-shared/listener-a", req.Name, req.ListenerId)
	}
	content := req.GetContents()["/index.html"]
	if content == nil || content.WebsiteId != "site-shared" || content.ListenerId != "listener-a" {
		t.Fatalf("content request = %#v, want scoped website content", content)
	}
}

func TestAddWebContentWithMetadataForwardsNameAndComment(t *testing.T) {
	rpc := &websiteTestRPC{}
	con := newWebsiteTestConsole(rpc)
	path := t.TempDir() + "/index.html"
	if err := os.WriteFile(path, []byte("hello"), 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	if _, err := AddWebContentWithMetadata(con, path, "/index.html", "site-a", "text/html", "none", "home", "landing page"); err != nil {
		t.Fatalf("AddWebContentWithMetadata failed: %v", err)
	}
	req := onlyWebsiteCall[*clientpb.Website](t, rpc, "AddWebsiteContent")
	content := req.GetContents()["/index.html"]
	if content == nil {
		t.Fatalf("missing /index.html content in request: %#v", req.GetContents())
	}
	if content.Name != "home" || content.Comment != "landing page" || content.Auth != "none" {
		t.Fatalf("metadata = name %q comment %q auth %q, want home/landing page/none", content.Name, content.Comment, content.Auth)
	}
}

func TestUpdateWebContentMetadataForwardsNameAndComment(t *testing.T) {
	rpc := &websiteTestRPC{}
	con := newWebsiteTestConsole(rpc)

	if _, err := UpdateWebContentMetadataFields(con, "content-id", "payload", "staged", []string{"name", "comment"}); err != nil {
		t.Fatalf("UpdateWebContentMetadata failed: %v", err)
	}
	req := onlyWebsiteCall[*clientpb.WebContent](t, rpc, "UpdateWebsiteContentMetadata")
	if req.Id != "content-id" || req.Name != "payload" || req.Comment != "staged" || len(req.UpdateFields) != 2 {
		t.Fatalf("request = %#v, want id/name/comment", req)
	}
}

func TestUpdateWebContentCmdAllowsMetadataOnly(t *testing.T) {
	rpc := &websiteTestRPC{}
	con := newWebsiteTestConsole(rpc)

	cmd := &cobra.Command{}
	cmd.Flags().String("website", "", "")
	cmd.Flags().String("type", "raw", "")
	cmd.Flags().String("name", "", "")
	cmd.Flags().String("comment", "", "")
	if err := cmd.Flags().Parse([]string{"content-id", "--comment", ""}); err != nil {
		t.Fatalf("Parse flags failed: %v", err)
	}

	if err := UpdateWebContentCmd(cmd, con); err != nil {
		t.Fatalf("UpdateWebContentCmd failed: %v", err)
	}
	req := onlyWebsiteCall[*clientpb.WebContent](t, rpc, "UpdateWebsiteContentMetadata")
	if req.Id != "content-id" || req.Comment != "" || len(req.UpdateFields) != 1 || req.UpdateFields[0] != "comment" {
		t.Fatalf("request = %#v, want comment-only metadata update", req)
	}
}

func TestAddWebContentCmdSupportsArtifactFlag(t *testing.T) {
	rpc := &websiteTestRPC{}
	con := newWebsiteTestConsole(rpc)
	con.Pipelines["listener-a:site-a"] = scopedWebsitePipeline("site-a", "listener-a")

	cmd := &cobra.Command{}
	cmd.Flags().String("website", "", "")
	cmd.Flags().String("path", "", "")
	cmd.Flags().String("type", "raw", "")
	cmd.Flags().String("auth", "", "")
	cmd.Flags().String("name", "", "")
	cmd.Flags().String("comment", "", "")
	cmd.Flags().String("artifact", "", "")
	cmd.Flags().String("format", "", "")
	cmd.Flags().String("RDI", "", "")
	if err := cmd.Flags().Parse([]string{
		"--artifact", "beacon",
		"--website", "listener-a:site-a",
		"--format", "shellcode",
		"--path", "/payload.bin",
		"--name", "payload",
	}); err != nil {
		t.Fatalf("Parse flags failed: %v", err)
	}

	if err := AddWebContentCmd(cmd, con); err != nil {
		t.Fatalf("AddWebContentCmd failed: %v", err)
	}
	if len(rpc.calls) != 2 {
		t.Fatalf("call count = %d, want 2", len(rpc.calls))
	}
	download := rpc.calls[0].request.(*clientpb.Artifact)
	if download.Name != "beacon" || download.Format != "raw" {
		t.Fatalf("download request = %#v, want beacon/raw", download)
	}
	add := rpc.calls[1].request.(*clientpb.Website)
	content := add.GetContents()["/payload.bin"]
	if content == nil || content.ContentType != "application/octet-stream" || content.Name != "payload" {
		t.Fatalf("content = %#v, want artifact website content", content)
	}
}

func TestAddArtifactContentDownloadsAndAddsWebsiteContent(t *testing.T) {
	rpc := &websiteTestRPC{}
	con := newWebsiteTestConsole(rpc)
	con.Pipelines["listener-a:site-a"] = scopedWebsitePipeline("site-a", "listener-a")

	if _, err := AddArtifactContent(con, "beacon", "listener-a:site-a", "shellcode", "", "/payload.bin", "application/octet-stream", "payload", "from artifact", "none"); err != nil {
		t.Fatalf("AddArtifactContent failed: %v", err)
	}
	if len(rpc.calls) != 2 {
		t.Fatalf("call count = %d, want 2", len(rpc.calls))
	}
	download := rpc.calls[0].request.(*clientpb.Artifact)
	if rpc.calls[0].method != "DownloadArtifact" || download.Name != "beacon" || download.Format != "raw" {
		t.Fatalf("download call = %s %#v, want beacon/raw", rpc.calls[0].method, download)
	}
	add := rpc.calls[1].request.(*clientpb.Website)
	content := add.GetContents()["/payload.bin"]
	if rpc.calls[1].method != "AddWebsiteContent" || add.Name != "site-a" || add.ListenerId != "listener-a" || content == nil {
		t.Fatalf("add call = %s %#v, content %#v, want scoped website payload", rpc.calls[1].method, add, content)
	}
	if string(content.Content) != "artifact-binary" || content.Name != "payload" || content.Comment != "from artifact" || content.Auth != "none" {
		t.Fatalf("content = %#v, want artifact bytes with metadata", content)
	}
}

func TestListWebContentCmdUsesScopedWebsiteCacheKey(t *testing.T) {
	rpc := &websiteTestRPC{}
	con := newWebsiteTestConsole(rpc)
	con.Pipelines["listener-b:site-shared"] = scopedWebsitePipeline("site-shared", "listener-b")

	cmd := newParsedWebsiteTestCmd(t, []string{"listener-b:site-shared"})
	if err := ListWebContentCmd(cmd, con); err != nil {
		t.Fatalf("ListWebContentCmd failed: %v", err)
	}
	req := onlyWebsiteCall[*clientpb.Website](t, rpc, "ListWebContent")
	if req.Name != "site-shared" || req.ListenerId != "listener-b" {
		t.Fatalf("website request = %s/%s, want site-shared/listener-b", req.Name, req.ListenerId)
	}
}

func TestUpdateWebContentUsesScopedWebsiteCacheKey(t *testing.T) {
	rpc := &websiteTestRPC{}
	con := newWebsiteTestConsole(rpc)
	con.Pipelines["listener-c:site-shared"] = scopedWebsitePipeline("site-shared", "listener-c")
	path := t.TempDir() + "/index.html"
	if err := os.WriteFile(path, []byte("updated"), 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	if _, err := UpdateWebContent(con, "content-id", path, "listener-c:site-shared", "text/html"); err != nil {
		t.Fatalf("UpdateWebContent failed: %v", err)
	}
	req := onlyWebsiteCall[*clientpb.WebContent](t, rpc, "UpdateWebsiteContent")
	if req.WebsiteId != "site-shared" || req.ListenerId != "listener-c" {
		t.Fatalf("content request = %s/%s, want site-shared/listener-c", req.WebsiteId, req.ListenerId)
	}
}

func TestStartWebsitePipelineCmdUsesScopedWebsiteCacheKey(t *testing.T) {
	rpc := &websiteTestRPC{}
	con := newWebsiteTestConsole(rpc)
	con.Pipelines["listener-d:site-shared"] = scopedWebsitePipeline("site-shared", "listener-d")

	cmd := newParsedWebsiteTestCmd(t, []string{"listener-d:site-shared"})
	cmd.Flags().String("listener", "", "")
	cmd.Flags().String("cert-name", "", "")
	if err := StartWebsitePipelineCmd(cmd, con); err != nil {
		t.Fatalf("StartWebsitePipelineCmd failed: %v", err)
	}
	if len(rpc.calls) != 2 {
		t.Fatalf("call count = %d, want 2", len(rpc.calls))
	}
	stopReq := rpc.calls[0].request.(*clientpb.CtrlPipeline)
	startReq := rpc.calls[1].request.(*clientpb.CtrlPipeline)
	if rpc.calls[0].method != "StopWebsite" || stopReq.Name != "site-shared" || stopReq.ListenerId != "listener-d" {
		t.Fatalf("stop call = %s %#v, want scoped stop", rpc.calls[0].method, stopReq)
	}
	if rpc.calls[1].method != "StartWebsite" || startReq.Name != "site-shared" || startReq.ListenerId != "listener-d" {
		t.Fatalf("start call = %s %#v, want scoped start", rpc.calls[1].method, startReq)
	}
}

func TestRestartWebsitePipelineCmdUsesScopedWebsiteCacheKey(t *testing.T) {
	rpc := &websiteTestRPC{}
	con := newWebsiteTestConsole(rpc)
	con.Pipelines["listener-r:site-shared"] = scopedWebsitePipeline("site-shared", "listener-r")

	cmd := newParsedWebsiteTestCmd(t, []string{"listener-r:site-shared"})
	cmd.Flags().String("listener", "", "")
	if err := RestartWebsitePipelineCmd(cmd, con); err != nil {
		t.Fatalf("RestartWebsitePipelineCmd failed: %v", err)
	}
	if len(rpc.calls) != 2 {
		t.Fatalf("call count = %d, want 2", len(rpc.calls))
	}
	stopReq := rpc.calls[0].request.(*clientpb.CtrlPipeline)
	startReq := rpc.calls[1].request.(*clientpb.CtrlPipeline)
	if rpc.calls[0].method != "StopWebsite" || stopReq.Name != "site-shared" || stopReq.ListenerId != "listener-r" {
		t.Fatalf("stop call = %s %#v, want scoped stop", rpc.calls[0].method, stopReq)
	}
	if rpc.calls[1].method != "StartWebsite" || startReq.Name != "site-shared" || startReq.ListenerId != "listener-r" {
		t.Fatalf("start call = %s %#v, want scoped start", rpc.calls[1].method, startReq)
	}
}

func TestWebsiteTLSCmdBindsExistingCert(t *testing.T) {
	rpc := &websiteTestRPC{}
	con := newWebsiteTestConsole(rpc)

	cmd := newWebsiteTLSParsedCmd(t, []string{"site-a", "--listener", "listener-a", "--cert-name", "cert-a"})
	if err := WebsiteTLSCmd(cmd, con); err != nil {
		t.Fatalf("WebsiteTLSCmd failed: %v", err)
	}
	req := onlyWebsiteCall[*clientpb.PipelineTLSUpdate](t, rpc, "UpdateWebsiteTLS")
	if req.Name != "site-a" || req.ListenerId != "listener-a" || req.CertName != "cert-a" || req.Mode != clientpb.TLSUpdateMode_TLS_UPDATE_MODE_EXISTING_CERT {
		t.Fatalf("request = %#v, want existing cert update", req)
	}
}

func TestWebsiteTLSCmdSavesInlineCert(t *testing.T) {
	rpc := &websiteTestRPC{}
	con := newWebsiteTestConsole(rpc)
	certPath, keyPath := writeWebsitePEMFixture(t)

	cmd := newWebsiteTLSParsedCmd(t, []string{
		"site-a",
		"--listener", "listener-a",
		"--cert", certPath,
		"--key", keyPath,
		"--save-cert",
		"--save-cert-name", "site-a-cert",
		"--cert-comment", "rotated",
	})
	if err := WebsiteTLSCmd(cmd, con); err != nil {
		t.Fatalf("WebsiteTLSCmd failed: %v", err)
	}
	req := onlyWebsiteCall[*clientpb.PipelineTLSUpdate](t, rpc, "UpdateWebsiteTLS")
	if req.Mode != clientpb.TLSUpdateMode_TLS_UPDATE_MODE_INLINE_CERT || !req.SaveCert || req.SaveCertName != "site-a-cert" || req.CertComment != "rotated" {
		t.Fatalf("request = %#v, want saved inline cert update", req)
	}
	if req.Tls == nil || req.Tls.Cert == nil || req.Tls.Cert.Cert == "" || req.Tls.Cert.Key == "" {
		t.Fatalf("request TLS = %#v, want inline cert and key", req.Tls)
	}
}

func TestWebsiteTLSCmdGeneratesTemporaryCert(t *testing.T) {
	rpc := &websiteTestRPC{}
	con := newWebsiteTestConsole(rpc)

	cmd := newWebsiteTLSParsedCmd(t, []string{"site-a", "--listener", "listener-a", "--generate"})
	if err := WebsiteTLSCmd(cmd, con); err != nil {
		t.Fatalf("WebsiteTLSCmd failed: %v", err)
	}
	req := onlyWebsiteCall[*clientpb.PipelineTLSUpdate](t, rpc, "UpdateWebsiteTLS")
	if req.Mode != clientpb.TLSUpdateMode_TLS_UPDATE_MODE_INLINE_CERT || req.SaveCert || req.Tls == nil || !req.Tls.Enable {
		t.Fatalf("request = %#v, want generated temporary inline TLS update", req)
	}
	if req.Tls.GetCert().GetCert() != "" || req.Tls.GetCert().GetKey() != "" {
		t.Fatalf("request TLS = %#v, generated cert material should be produced server-side", req.Tls)
	}
}

func TestNewWebsiteCmdAllowsGeneratedTLSWithoutCert(t *testing.T) {
	rpc := &websiteTestRPC{}
	con := newWebsiteTestConsole(rpc)

	cmd := newWebsiteCreateParsedCmd(t, []string{"site-a", "--listener", "listener-a", "--port", "18080", "--tls"})
	if err := NewWebsiteCmd(cmd, con); err != nil {
		t.Fatalf("NewWebsiteCmd failed: %v", err)
	}
	if len(rpc.calls) != 2 {
		t.Fatalf("call count = %d, want RegisterWebsite and StartWebsite", len(rpc.calls))
	}
	registerReq := rpc.calls[0].request.(*clientpb.Pipeline)
	startReq := rpc.calls[1].request.(*clientpb.CtrlPipeline)
	if rpc.calls[0].method != "RegisterWebsite" || registerReq.GetTls() == nil || !registerReq.GetTls().Enable {
		t.Fatalf("register call = %s %#v, want TLS-enabled website", rpc.calls[0].method, registerReq)
	}
	if registerReq.GetTls().GetCert().GetCert() != "" || registerReq.GetTls().GetCert().GetKey() != "" {
		t.Fatalf("register TLS = %#v, generated cert material should be produced server-side", registerReq.GetTls())
	}
	if rpc.calls[1].method != "StartWebsite" || startReq.GetPipeline().GetTls() == nil || !startReq.GetPipeline().GetTls().Enable {
		t.Fatalf("start call = %s %#v, want TLS-enabled pipeline", rpc.calls[1].method, startReq)
	}
}

func TestNewWebsiteCmdSavesGeneratedTLS(t *testing.T) {
	rpc := &websiteTestRPC{}
	con := newWebsiteTestConsole(rpc)

	cmd := newWebsiteCreateParsedCmd(t, []string{
		"site-a",
		"--listener", "listener-a",
		"--port", "18080",
		"--tls",
		"--save-cert",
		"--save-cert-name", "site-a-cert",
		"--cert-comment", "generated",
	})
	if err := NewWebsiteCmd(cmd, con); err != nil {
		t.Fatalf("NewWebsiteCmd failed: %v", err)
	}
	if len(rpc.calls) != 3 {
		t.Fatalf("call count = %d, want RegisterWebsite, StartWebsite, UpdateWebsiteTLS", len(rpc.calls))
	}
	updateReq := rpc.calls[2].request.(*clientpb.PipelineTLSUpdate)
	if rpc.calls[2].method != "UpdateWebsiteTLS" || updateReq.Mode != clientpb.TLSUpdateMode_TLS_UPDATE_MODE_INLINE_CERT {
		t.Fatalf("update call = %s %#v, want generated inline TLS update", rpc.calls[2].method, updateReq)
	}
	if !updateReq.SaveCert || updateReq.SaveCertName != "site-a-cert" || updateReq.CertComment != "generated" {
		t.Fatalf("update save fields = %#v, want saved generated cert", updateReq)
	}
	if updateReq.Tls == nil || !updateReq.Tls.Enable || updateReq.Tls.GetCert().GetCert() != "" || updateReq.Tls.GetCert().GetKey() != "" {
		t.Fatalf("update TLS = %#v, want server-side generated TLS marker", updateReq.Tls)
	}
}

func TestWebsiteTLSCmdRejectsSaveCertWithoutName(t *testing.T) {
	rpc := &websiteTestRPC{}
	con := newWebsiteTestConsole(rpc)
	certPath, keyPath := writeWebsitePEMFixture(t)

	cmd := newWebsiteTLSParsedCmd(t, []string{
		"site-a",
		"--cert", certPath,
		"--key", keyPath,
		"--save-cert",
	})
	if err := WebsiteTLSCmd(cmd, con); err == nil {
		t.Fatal("WebsiteTLSCmd should reject --save-cert without --save-cert-name")
	}
}

func TestResolveWebsiteTargetParsesScopedValueWithoutCache(t *testing.T) {
	name, listenerID, cached := resolveWebsiteTarget(nil, "listener-z:site-z")
	if name != "site-z" || listenerID != "listener-z" || cached {
		t.Fatalf("resolveWebsiteTarget = %s/%s cached=%v, want site-z/listener-z false", name, listenerID, cached)
	}
}

func scopedWebsitePipeline(name, listenerID string) *clientpb.Pipeline {
	return &clientpb.Pipeline{
		Name:       name,
		ListenerId: listenerID,
		Type:       consts.WebsitePipeline,
		Enable:     true,
		Body: &clientpb.Pipeline_Web{
			Web: &clientpb.Website{Name: name},
		},
	}
}

func onlyWebsiteCall[T any](t testing.TB, rpc *websiteTestRPC, method string) T {
	t.Helper()
	var zero T
	if len(rpc.calls) != 1 {
		t.Fatalf("call count = %d, want 1", len(rpc.calls))
	}
	if rpc.calls[0].method != method {
		t.Fatalf("method = %s, want %s", rpc.calls[0].method, method)
	}
	req, ok := rpc.calls[0].request.(T)
	if !ok {
		t.Fatalf("request type = %T, want %T", rpc.calls[0].request, zero)
	}
	return req
}

func newParsedWebsiteTestCmd(t testing.TB, args []string) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{}
	if err := cmd.Flags().Parse(args); err != nil {
		t.Fatalf("Parse flags failed: %v", err)
	}
	return cmd
}

func newWebsiteTLSParsedCmd(t testing.TB, args []string) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{}
	cmd.Flags().String("listener", "", "")
	cmd.Flags().Bool("disable", false, "")
	cmd.Flags().String("cert-name", "", "")
	cmd.Flags().Bool("generate", false, "")
	cmd.Flags().String("cert", "", "")
	cmd.Flags().String("key", "", "")
	cmd.Flags().Bool("save-cert", false, "")
	cmd.Flags().String("save-cert-name", "", "")
	cmd.Flags().String("cert-comment", "", "")
	if err := cmd.Flags().Parse(args); err != nil {
		t.Fatalf("Parse flags failed: %v", err)
	}
	return cmd
}

func newWebsiteCreateParsedCmd(t testing.TB, args []string) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{}
	cmd.Flags().String("root", "/", "")
	cmd.Flags().String("auth", "", "")
	cmd.Flags().String("listener", "", "")
	cmd.Flags().String("host", "0.0.0.0", "")
	cmd.Flags().String("proxy", "", "")
	cmd.Flags().Uint32("port", 0, "")
	cmd.Flags().String("cert", "", "")
	cmd.Flags().String("key", "", "")
	cmd.Flags().Bool("tls", false, "")
	cmd.Flags().String("cert-name", "", "")
	cmd.Flags().Bool("save-cert", false, "")
	cmd.Flags().String("save-cert-name", "", "")
	cmd.Flags().String("cert-comment", "", "")
	if err := cmd.Flags().Parse(args); err != nil {
		t.Fatalf("Parse flags failed: %v", err)
	}
	return cmd
}

func writeWebsitePEMFixture(t testing.TB) (string, string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   "website.example",
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
