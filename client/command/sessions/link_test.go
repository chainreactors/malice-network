package sessions

import (
	"context"
	"testing"

	iomclient "github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"google.golang.org/grpc"
)

type sessionLinkCommandRPC struct {
	clientrpc.MaliceRPCClient

	listRequest   *clientpb.SessionLinkRequest
	setRequest    *clientpb.SessionLinkRequest
	removeRequest *clientpb.SessionLinkRequest
}

func (rpc *sessionLinkCommandRPC) ListSessionLinks(_ context.Context, req *clientpb.SessionLinkRequest, _ ...grpc.CallOption) (*clientpb.SessionLinks, error) {
	rpc.listRequest = req
	return &clientpb.SessionLinks{}, nil
}

func (rpc *sessionLinkCommandRPC) SetSessionLink(_ context.Context, req *clientpb.SessionLinkRequest, _ ...grpc.CallOption) (*clientpb.SessionLink, error) {
	rpc.setRequest = req
	return &clientpb.SessionLink{
		ParentSessionId: req.GetParentSessionId(),
		ChildSessionId:  req.GetChildSessionId(),
		Source:          "manual",
	}, nil
}

func (rpc *sessionLinkCommandRPC) RemoveSessionLink(_ context.Context, req *clientpb.SessionLinkRequest, _ ...grpc.CallOption) (*clientpb.Empty, error) {
	rpc.removeRequest = req
	return &clientpb.Empty{}, nil
}

func TestSessionLinkCommandSetReparentAndUnlink(t *testing.T) {
	con := newSessionTestConsole(t)
	parent := addSessionFixture(t, con, "link-command-parent")
	child := addSessionFixture(t, con, "link-command-child")
	rpc := &sessionLinkCommandRPC{}
	con.Server.ServerState.Rpc = &iomclient.Rpc{MaliceRPCClient: rpc}
	con.Server.ServerState.Client = &clientpb.Client{Name: "tester", ID: 1}

	command := Commands(con)[0]
	command.SetArgs([]string{"link", "reparent", "--parent", parent.SessionId, "--child", child.SessionId})
	if err := command.Execute(); err != nil {
		t.Fatalf("session link reparent failed: %v", err)
	}
	if rpc.setRequest == nil || rpc.setRequest.GetParentSessionId() != parent.SessionId || rpc.setRequest.GetChildSessionId() != child.SessionId {
		t.Fatalf("set request = %#v", rpc.setRequest)
	}

	command = Commands(con)[0]
	command.SetArgs([]string{"link", "unlink", "--child", child.SessionId})
	if err := command.Execute(); err != nil {
		t.Fatalf("session link unlink failed: %v", err)
	}
	if rpc.removeRequest == nil || rpc.removeRequest.GetChildSessionId() != child.SessionId {
		t.Fatalf("remove request = %#v", rpc.removeRequest)
	}
}

func TestSessionLinkCommandListFilters(t *testing.T) {
	con := newSessionTestConsole(t)
	parent := addSessionFixture(t, con, "link-list-parent")
	child := addSessionFixture(t, con, "link-list-child")
	rpc := &sessionLinkCommandRPC{}
	con.Server.ServerState.Rpc = &iomclient.Rpc{MaliceRPCClient: rpc}
	con.Server.ServerState.Client = &clientpb.Client{Name: "tester", ID: 1}

	command := Commands(con)[0]
	command.SetArgs([]string{"link", "list", "--parent", parent.SessionId, "--child", child.SessionId})
	if err := command.Execute(); err != nil {
		t.Fatalf("session link list failed: %v", err)
	}
	if rpc.listRequest == nil || rpc.listRequest.GetParentSessionId() != parent.SessionId || rpc.listRequest.GetChildSessionId() != child.SessionId {
		t.Fatalf("list request = %#v", rpc.listRequest)
	}
}
