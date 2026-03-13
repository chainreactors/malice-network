package sessions

import (
	"strings"
	"testing"

	iomclient "github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
)

func TestUseSessionCmdSwitchesActiveSessionAndMenu(t *testing.T) {
	con := newSessionTestConsole(t)
	sess := addSessionFixture(t, con, "use-session-12345678")

	cmd := &cobra.Command{Use: "use"}
	if err := cmd.Flags().Parse([]string{sess.SessionId}); err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if err := UseSessionCmd(cmd, con); err != nil {
		t.Fatalf("UseSessionCmd failed: %v", err)
	}
	if got := con.GetInteractive(); got == nil || got.SessionId != sess.SessionId {
		t.Fatalf("interactive session = %#v, want %s", got, sess.SessionId)
	}
	if menu := con.App.ActiveMenu(); menu == nil || menu.Name() != consts.ImplantMenu {
		t.Fatalf("active menu = %#v, want %s", menu, consts.ImplantMenu)
	}
}

func TestFindSessionByPrefixHandlesShortIDsSafely(t *testing.T) {
	con := newSessionTestConsole(t)
	addSessionFixture(t, con, "alpha1")
	addSessionFixture(t, con, "alpha2")

	_, err := findSessionByPrefix(con, "alpha")
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("findSessionByPrefix error = %v, want ambiguous prefix error", err)
	}
}

func TestObserveCmdUsesInteractiveSessionWhenNoArgs(t *testing.T) {
	con := newSessionTestConsole(t)
	sess := addSessionFixture(t, con, "observe-session")
	con.ActiveTarget.Set(sess)

	cmd := &cobra.Command{Use: "observe"}
	cmd.Flags().Bool("list", false, "")
	cmd.Flags().Bool("remove", false, "")

	if err := ObserveCmd(cmd, con); err != nil {
		t.Fatalf("ObserveCmd failed: %v", err)
	}
	if _, ok := con.Observers[sess.SessionId]; !ok {
		t.Fatalf("expected %s to be added as observer", sess.SessionId)
	}
}

func TestObserveCmdRemovesInteractiveSessionWhenRequested(t *testing.T) {
	con := newSessionTestConsole(t)
	sess := addSessionFixture(t, con, "observe-remove")
	con.ActiveTarget.Set(sess)
	con.AddObserver(sess)

	cmd := &cobra.Command{Use: "observe"}
	cmd.Flags().Bool("list", false, "")
	cmd.Flags().Bool("remove", false, "")
	if err := cmd.Flags().Set("remove", "true"); err != nil {
		t.Fatalf("Set(remove) failed: %v", err)
	}

	if err := ObserveCmd(cmd, con); err != nil {
		t.Fatalf("ObserveCmd failed: %v", err)
	}
	if _, ok := con.Observers[sess.SessionId]; ok {
		t.Fatalf("expected %s to be removed from observers", sess.SessionId)
	}
}

func TestBackGroundClearsInteractiveSessionAndReturnsToClientMenu(t *testing.T) {
	con := newSessionTestConsole(t)
	sess := addSessionFixture(t, con, "background-session")
	con.ActiveTarget.Set(sess)
	con.App.SwitchMenu(consts.ImplantMenu)

	if err := BackGround(&cobra.Command{Use: "background"}, con); err != nil {
		t.Fatalf("BackGround failed: %v", err)
	}
	if con.ActiveTarget.Get() != nil {
		t.Fatal("expected interactive session to be cleared")
	}
	if menu := con.App.ActiveMenu(); menu == nil || menu.Name() != consts.ClientMenu {
		t.Fatalf("active menu = %#v, want %s", menu, consts.ClientMenu)
	}
}

func TestShortSessionIDLeavesShortValuesUnchanged(t *testing.T) {
	if got := shortSessionID("abc123"); got != "abc123" {
		t.Fatalf("shortSessionID = %q, want %q", got, "abc123")
	}
}

func newSessionTestConsole(t testing.TB) *core.Console {
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
	con.App.Menu(consts.ClientMenu).Command = &cobra.Command{Use: consts.ClientMenu}
	con.App.Menu(consts.ImplantMenu).Command = &cobra.Command{Use: consts.ImplantMenu}
	con.App.SwitchMenu(consts.ClientMenu)
	return con
}

func addSessionFixture(t testing.TB, con *core.Console, sessionID string) *iomclient.Session {
	t.Helper()

	sess := iomclient.NewSession(&clientpb.Session{
		SessionId:  sessionID,
		Type:       consts.ImplantMalefic,
		PipelineId: "pipe-session",
		Note:       "note",
		GroupName:  "group",
		Modules:    []string{},
		Addons:     []*implantpb.Addon{},
		Timer: &implantpb.Timer{
			Expression: "*/30 * * * * * *",
			Jitter:     0.15,
		},
		Os: &implantpb.Os{
			Name:     "windows",
			Arch:     "amd64",
			Hostname: "host-a",
		},
		Process: &implantpb.Process{
			Ppid: 4,
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
