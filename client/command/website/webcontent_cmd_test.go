package website

import (
	"context"
	"os"
	"testing"

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

func (r *websiteTestRPC) ListWebContent(ctx context.Context, in *clientpb.Website, opts ...grpc.CallOption) (*clientpb.WebContents, error) {
	r.record("ListWebContent", in)
	return &clientpb.WebContents{}, nil
}

func (r *websiteTestRPC) StopWebsite(ctx context.Context, in *clientpb.CtrlPipeline, opts ...grpc.CallOption) (*clientpb.Empty, error) {
	r.record("StopWebsite", in)
	return &clientpb.Empty{}, nil
}

func (r *websiteTestRPC) StartWebsite(ctx context.Context, in *clientpb.CtrlPipeline, opts ...grpc.CallOption) (*clientpb.Empty, error) {
	r.record("StartWebsite", in)
	return &clientpb.Empty{}, nil
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
