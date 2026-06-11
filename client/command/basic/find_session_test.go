package basic

import (
	"testing"

	iomclient "github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
)

func TestFindSessionByPrefix(t *testing.T) {
	con := newPrefixTestConsole(t)
	addPrefixSession(t, con, "alpha1111")
	addPrefixSession(t, con, "alpha2222")
	addPrefixSession(t, con, "beta3333")

	if _, err := findSessionByPrefix(con, "alpha"); err == nil {
		t.Fatal("expected ambiguous prefix error")
	}

	sess, err := findSessionByPrefix(con, "beta")
	if err != nil {
		t.Fatalf("findSessionByPrefix failed: %v", err)
	}
	if sess.SessionId != "beta3333" {
		t.Fatalf("session id = %q, want beta3333", sess.SessionId)
	}
}

func newPrefixTestConsole(t testing.TB) *core.Console {
	t.Helper()

	oldDir := assets.MaliceDirName
	assets.MaliceDirName = t.TempDir()
	assets.InitLogDir()
	t.Cleanup(func() {
		assets.MaliceDirName = oldDir
		assets.InitLogDir()
	})

	state := &iomclient.ServerState{
		Rpc:           &iomclient.Rpc{},
		ActiveTarget:  &iomclient.ActiveTarget{},
		Listeners:     map[string]*clientpb.Listener{},
		Pipelines:     map[string]*clientpb.Pipeline{},
		Sessions:      map[string]*iomclient.Session{},
		Observers:     map[string]*iomclient.Session{},
		EventHook:     map[iomclient.EventCondition][]iomclient.OnEventFunc{},
		EventCallback: map[string]func(*clientpb.Event){},
	}
	con := &core.Console{
		Server:  &core.Server{ServerState: state},
		Log:     iomclient.Log,
		CMDs:    map[string]*cobra.Command{},
		Helpers: map[string]*cobra.Command{},
	}
	con.NewConsole()
	con.App.SwitchMenu(consts.ImplantMenu)
	return con
}

func addPrefixSession(t testing.TB, con *core.Console, sessionID string) {
	t.Helper()

	sess := iomclient.NewSession(&clientpb.Session{
		SessionId:  sessionID,
		Type:       consts.ImplantMalefic,
		PipelineId: "pipe-prefix",
		Timer: &implantpb.Timer{
			Expression: "*/30 * * * * * *",
			Jitter:     0.25,
		},
		Os: &implantpb.Os{
			Name: "windows",
			Arch: "amd64",
		},
		Data: "null",
	}, con.Server.ServerState)
	con.Sessions[sessionID] = sess
	t.Cleanup(func() {
		_ = sess.Close()
	})
}
