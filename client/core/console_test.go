package core

import (
	"strings"
	"testing"

	iomclient "github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
)

func TestGetPromptWithoutServerDoesNotPanic(t *testing.T) {
	con := &Console{}

	prompt := con.GetPrompt()
	if !strings.Contains(prompt, "❯") {
		t.Fatalf("prompt = %q, want prompt character", prompt)
	}
}

func TestGetPromptHandlesShortSessionID(t *testing.T) {
	con := newPromptTestConsole()
	sess := addPromptSessionFixture(t, con, "abc123")
	con.ActiveTarget.Set(sess)

	prompt := con.GetPrompt()
	if !strings.Contains(prompt, "abc123") {
		t.Fatalf("prompt = %q, want to contain short session id", prompt)
	}
}

func TestGetStatusLineFallsBackToCachedSessionCountWhenRPCMissing(t *testing.T) {
	con := newPromptTestConsole()
	addPromptSessionFixture(t, con, "sess-a")
	addPromptSessionFixture(t, con, "sess-b")

	status := con.getStatusLine()
	if !strings.Contains(status, "2") {
		t.Fatalf("status line = %q, want cached session count", status)
	}
}

func TestRefreshActiveSessionIgnoresMissingInteractiveSession(t *testing.T) {
	con := newPromptTestConsole()
	con.RefreshActiveSession()
}

func newPromptTestConsole() *Console {
	state := &iomclient.ServerState{
		ActiveTarget: &iomclient.ActiveTarget{},
		Sessions:     map[string]*iomclient.Session{},
	}
	return &Console{
		Server: &Server{ServerState: state},
		Log:    iomclient.Log,
	}
}

func addPromptSessionFixture(t testing.TB, con *Console, sessionID string) *iomclient.Session {
	t.Helper()

	sess := iomclient.NewSession(&clientpb.Session{
		SessionId:  sessionID,
		Type:       "malefic",
		PipelineId: "pipe-a",
		GroupName:  "ops",
		Os: &implantpb.Os{
			Name:     "windows",
			Arch:     "amd64",
			Hostname: "host-a",
		},
		Process: &implantpb.Process{
			Name: "agent.exe",
		},
		Data: "null",
	}, con.Server.ServerState)
	con.Sessions[sessionID] = sess
	t.Cleanup(func() {
		_ = sess.Close()
	})
	return sess
}
