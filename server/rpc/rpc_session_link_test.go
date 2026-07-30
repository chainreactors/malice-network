package rpc

import (
	"context"
	"testing"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestSessionLinkRPCSetReparentListAndRemove(t *testing.T) {
	env := newRPCTestEnv(t)
	env.seedSession(t, "link-parent-a", "link-pipe-a", false)
	env.seedSession(t, "link-parent-b", "link-pipe-b", false)
	env.seedSession(t, "link-child", "link-pipe-child", false)
	server := &Server{}

	created, err := server.SetSessionLink(context.Background(), &clientpb.SessionLinkRequest{
		ParentSessionId: "link-parent-a",
		ChildSessionId:  "link-child",
	})
	if err != nil {
		t.Fatalf("SetSessionLink create failed: %v", err)
	}
	if created.GetParentSessionId() != "link-parent-a" || created.GetSource() != "manual" {
		t.Fatalf("created link = %#v", created)
	}

	reparented, err := server.SetSessionLink(context.Background(), &clientpb.SessionLinkRequest{
		ParentSessionId: "link-parent-b",
		ChildSessionId:  "link-child",
	})
	if err != nil {
		t.Fatalf("SetSessionLink reparent failed: %v", err)
	}
	if reparented.GetParentSessionId() != "link-parent-b" {
		t.Fatalf("reparented link = %#v", reparented)
	}

	listed, err := server.ListSessionLinks(context.Background(), &clientpb.SessionLinkRequest{ChildSessionId: "link-child"})
	if err != nil {
		t.Fatalf("ListSessionLinks failed: %v", err)
	}
	if len(listed.GetLinks()) != 1 || listed.GetLinks()[0].GetParentSessionId() != "link-parent-b" {
		t.Fatalf("listed links = %#v", listed.GetLinks())
	}

	if _, err := server.RemoveSessionLink(context.Background(), &clientpb.SessionLinkRequest{ChildSessionId: "link-child"}); err != nil {
		t.Fatalf("RemoveSessionLink failed: %v", err)
	}
	listed, err = server.ListSessionLinks(context.Background(), &clientpb.SessionLinkRequest{})
	if err != nil {
		t.Fatalf("ListSessionLinks after remove failed: %v", err)
	}
	if len(listed.GetLinks()) != 0 {
		t.Fatalf("links after remove = %#v, want none", listed.GetLinks())
	}
}

func TestSessionLinkRPCValidationCodes(t *testing.T) {
	env := newRPCTestEnv(t)
	env.seedSession(t, "link-cycle-a", "link-cycle-pipe-a", false)
	env.seedSession(t, "link-cycle-b", "link-cycle-pipe-b", false)
	server := &Server{}

	if _, err := server.SetSessionLink(context.Background(), &clientpb.SessionLinkRequest{}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("missing IDs code = %s, want %s", status.Code(err), codes.InvalidArgument)
	}
	if _, err := server.SetSessionLink(context.Background(), &clientpb.SessionLinkRequest{
		ParentSessionId: "link-cycle-a",
		ChildSessionId:  "link-missing",
	}); status.Code(err) != codes.NotFound {
		t.Fatalf("missing session code = %s, want %s", status.Code(err), codes.NotFound)
	}
	if _, err := server.SetSessionLink(context.Background(), &clientpb.SessionLinkRequest{
		ParentSessionId: "link-cycle-a",
		ChildSessionId:  "link-cycle-b",
	}); err != nil {
		t.Fatalf("SetSessionLink A -> B failed: %v", err)
	}
	if _, err := server.SetSessionLink(context.Background(), &clientpb.SessionLinkRequest{
		ParentSessionId: "link-cycle-b",
		ChildSessionId:  "link-cycle-a",
	}); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("cycle code = %s, want %s", status.Code(err), codes.FailedPrecondition)
	}
	if _, err := server.RemoveSessionLink(context.Background(), &clientpb.SessionLinkRequest{ChildSessionId: "link-cycle-a"}); status.Code(err) != codes.NotFound {
		t.Fatalf("missing link code = %s, want %s", status.Code(err), codes.NotFound)
	}
}
