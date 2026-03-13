package command

import (
	"context"
	"testing"

	iomclient "github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

type fakeImplantRPC struct {
	clientrpc.MaliceRPCClient
	waitTaskFinishFunc func(context.Context, *clientpb.Task, ...grpc.CallOption) (*clientpb.TaskContext, error)
}

func (f *fakeImplantRPC) WaitTaskFinish(ctx context.Context, in *clientpb.Task, opts ...grpc.CallOption) (*clientpb.TaskContext, error) {
	if f.waitTaskFinishFunc != nil {
		return f.waitTaskFinishFunc(ctx, in, opts...)
	}
	return nil, nil
}

func TestMakeRunnersPreSelectsLocalSessionAndSwitchesMenu(t *testing.T) {
	con := newImplantTestConsole(t, &fakeImplantRPC{})
	sess := addImplantTestSession(t, con, "implant-pre")

	root := newImplantTestRoot(con)
	if err := root.Flags().Set("use", sess.SessionId); err != nil {
		t.Fatalf("set use flag: %v", err)
	}

	pre, _ := makeRunners(root, con)
	if err := pre(root, nil); err != nil {
		t.Fatalf("pre runner failed: %v", err)
	}

	if got := con.ActiveTarget.Get(); got == nil || got.SessionId != sess.SessionId {
		t.Fatalf("active target = %#v, want session %s", got, sess.SessionId)
	}
	if menu := con.App.ActiveMenu(); menu == nil || menu.Name() != consts.ImplantMenu {
		t.Fatalf("active menu = %#v, want %s", menu, consts.ImplantMenu)
	}
}

func TestMakeRunnersPreRequiresSessionOutsideAllowedGroups(t *testing.T) {
	con := newImplantTestConsole(t, &fakeImplantRPC{})
	root := newImplantTestRoot(con)

	pre, _ := makeRunners(root, con)
	err := pre(root, nil)
	if err == nil || err.Error() != "no implant to run command on" {
		t.Fatalf("pre runner error = %v, want no implant error", err)
	}
}

func TestMakeRunnersPreAllowsGenericGroupWithoutSession(t *testing.T) {
	con := newImplantTestConsole(t, &fakeImplantRPC{})
	root := newImplantTestRoot(con)
	cmd := &cobra.Command{Use: "help", GroupID: consts.GenericGroup}

	pre, _ := makeRunners(root, con)
	if err := pre(cmd, nil); err != nil {
		t.Fatalf("pre runner failed for generic command: %v", err)
	}
}

func TestMakeRunnersPreBypassesCompletionMode(t *testing.T) {
	con := newImplantTestConsole(t, &fakeImplantRPC{})
	root := newImplantTestRoot(con)

	t.Setenv("IOM_COMPLETING", "1")

	pre, _ := makeRunners(root, con)
	if err := pre(root, nil); err != nil {
		t.Fatalf("pre runner failed in completion mode: %v", err)
	}
}

func TestMakeRunnersPostWaitsForLastTask(t *testing.T) {
	var waited bool
	rpc := &fakeImplantRPC{
		waitTaskFinishFunc: func(_ context.Context, task *clientpb.Task, _ ...grpc.CallOption) (*clientpb.TaskContext, error) {
			waited = true
			return &clientpb.TaskContext{
				Task: task,
				Spite: &implantpb.Spite{
					Body: &implantpb.Spite_Empty{Empty: &implantpb.Empty{}},
				},
			}, nil
		},
	}
	con := newImplantTestConsole(t, rpc)
	sess := addImplantTestSession(t, con, "implant-post")
	sess.LastTask = &clientpb.Task{
		TaskId:    9,
		SessionId: sess.SessionId,
		Type:      consts.ModuleSleep,
		Cur:       1,
		Total:     1,
	}
	con.ActiveTarget.Set(sess)

	root := newImplantTestRoot(con)
	if err := root.Flags().Set("wait", "true"); err != nil {
		t.Fatalf("set wait flag: %v", err)
	}

	_, post := makeRunners(root, con)
	if err := post(root, nil); err != nil {
		t.Fatalf("post runner failed: %v", err)
	}
	if !waited {
		t.Fatal("expected WaitTaskFinish to be called")
	}
}

func newImplantTestRoot(con *core.Console) *cobra.Command {
	root := &cobra.Command{Use: "implant"}
	root.Flags().String("use", "", "")
	root.Flags().Bool("wait", false, "")
	root.Flags().Bool("yes", false, "")
	con.App.Menu(consts.ImplantMenu).Command = root
	return root
}

func newImplantTestConsole(t testing.TB, rpc clientrpc.MaliceRPCClient) *core.Console {
	t.Helper()

	oldDir := assets.MaliceDirName
	assets.MaliceDirName = t.TempDir()
	assets.InitLogDir()
	t.Cleanup(func() {
		assets.MaliceDirName = oldDir
		assets.InitLogDir()
	})

	state := &iomclient.ServerState{
		Rpc:             &iomclient.Rpc{MaliceRPCClient: rpc},
		Client:          &clientpb.Client{Name: "tester", ID: 1},
		ActiveTarget:    &iomclient.ActiveTarget{},
		Listeners:       map[string]*clientpb.Listener{},
		Pipelines:       map[string]*clientpb.Pipeline{},
		Sessions:        map[string]*iomclient.Session{},
		Observers:       map[string]*iomclient.Session{},
		FinishCallbacks: nil,
		DoneCallbacks:   nil,
		EventHook:       map[iomclient.EventCondition][]iomclient.OnEventFunc{},
		EventCallback:   map[string]func(*clientpb.Event){},
	}
	con := &core.Console{
		Server:  &core.Server{ServerState: state},
		Log:     iomclient.Log,
		CMDs:    map[string]*cobra.Command{},
		Helpers: map[string]*cobra.Command{},
	}
	con.NewConsole()
	con.App.Shell().Line().Set([]rune("implant test")...)
	return con
}

func addImplantTestSession(t testing.TB, con *core.Console, sessionID string) *iomclient.Session {
	t.Helper()

	sess := iomclient.NewSession(&clientpb.Session{
		SessionId:  sessionID,
		Type:       consts.ImplantMalefic,
		PipelineId: "pipe-test",
		Timer: &implantpb.Timer{
			Expression: "*/30 * * * * * *",
			Jitter:     0.25,
		},
		Os:   &implantpb.Os{Name: "windows", Arch: "amd64"},
		Data: "null",
	}, con.Server.ServerState)
	con.Sessions[sessionID] = sess
	t.Cleanup(func() {
		_ = sess.Close()
	})
	return sess
}
