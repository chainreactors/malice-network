package testsupport

import (
	"encoding/binary"
	"strings"
	"sync"
	"testing"

	iomclient "github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/client/assets"
	commandpkg "github.com/chainreactors/malice-network/client/command"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

type Harness struct {
	Console  *core.Console
	Recorder *RecorderRPC
	Session  *iomclient.Session
}

type CommandCase struct {
	Name    string
	Argv    []string
	Setup   func(testing.TB, *Harness)
	WantErr string
	Assert  func(testing.TB, *Harness, error)
}

func NewHarness(t testing.TB) *Harness {
	t.Helper()

	h := newHarness(t)
	h.Session = h.AddSession(t, "test-session-12345678")
	h.Console.ActiveTarget.Set(h.Session)
	h.Console.App.SwitchMenu(consts.ImplantMenu)
	return h
}

func NewClientHarness(t testing.TB) *Harness {
	t.Helper()

	h := newHarness(t)
	h.Console.App.SwitchMenu(consts.ClientMenu)
	return h
}

func newHarness(t testing.TB) *Harness {
	t.Helper()

	oldDir := assets.MaliceDirName
	assets.MaliceDirName = t.TempDir()
	assets.InitLogDir()
	t.Cleanup(func() {
		assets.MaliceDirName = oldDir
		assets.InitLogDir()
	})

	recorder := NewRecorderRPC()
	state := &iomclient.ServerState{
		Rpc:             &iomclient.Rpc{MaliceRPCClient: recorder, ListenerRPCClient: recorder},
		Client:          &clientpb.Client{Name: "tester", ID: 1},
		ActiveTarget:    &iomclient.ActiveTarget{},
		Listeners:       map[string]*clientpb.Listener{},
		Pipelines:       map[string]*clientpb.Pipeline{},
		Sessions:        map[string]*iomclient.Session{},
		Observers:       map[string]*iomclient.Session{},
		FinishCallbacks: &sync.Map{},
		DoneCallbacks:   &sync.Map{},
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
	con.App.Shell().Line().Set([]rune("command test")...)

	h := &Harness{
		Console:  con,
		Recorder: recorder,
	}
	return h
}

func (h *Harness) AddSession(t testing.TB, sessionID string) *iomclient.Session {
	t.Helper()

	rawID := uint32(7)
	session := &clientpb.Session{
		SessionId:  sessionID,
		RawId:      rawID,
		Type:       consts.ImplantMalefic,
		PipelineId: "pipe-test",
		Note:       "fixture",
		GroupName:  "group",
		Timer: &implantpb.Timer{
			Expression: "*/30 * * * * * *",
			Jitter:     0.25,
		},
		Os: &implantpb.Os{
			Name:     "windows",
			Arch:     "amd64",
			Hostname: "host-a",
		},
		Process: &implantpb.Process{
			Name: "agent.exe",
			Pid:  4321,
			Ppid: 1234,
		},
		Data: "null",
	}
	sess := iomclient.NewSession(session, h.Console.Server.ServerState)
	h.Console.Sessions[sessionID] = sess
	h.Recorder.SetSession(session)
	t.Cleanup(func() {
		_ = sess.Close()
	})
	return sess
}

func (h *Harness) SetSessionResponse(session *clientpb.Session) {
	h.Recorder.SetSession(session)
}

func (h *Harness) AddTCPPipeline(name, host string, port uint32) {
	h.Console.Pipelines[name] = &clientpb.Pipeline{
		Name: name,
		Ip:   host,
		Body: &clientpb.Pipeline_Tcp{
			Tcp: &clientpb.TCPPipeline{
				Name: name,
				Port: port,
			},
		},
	}
}

func (h *Harness) Execute(argv ...string) error {
	root := commandpkg.ImplantCmd(h.Console)
	root.SilenceErrors = true
	root.SilenceUsage = true
	h.Console.App.Shell().Line().Set([]rune(strings.Join(argv, " "))...)

	args := argv
	if !containsUseFlag(args) {
		args = append([]string{"--use", h.Session.SessionId}, args...)
	}
	root.SetArgs(args)
	return root.Execute()
}

func (h *Harness) ExecuteClient(argv ...string) error {
	root := commandpkg.BindClientsCommands(h.Console)()
	root.SilenceErrors = true
	root.SilenceUsage = true
	h.Console.App.Shell().Line().Set([]rune(strings.Join(argv, " "))...)
	root.SetArgs(argv)
	return root.Execute()
}

func RunCases(t *testing.T, cases []CommandCase) {
	t.Helper()

	for _, tc := range cases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			h := NewHarness(t)
			if tc.Setup != nil {
				tc.Setup(t, h)
			}

			err := h.Execute(tc.Argv...)
			switch {
			case tc.WantErr == "" && err != nil:
				t.Fatalf("execute %q failed: %v", strings.Join(tc.Argv, " "), err)
			case tc.WantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.WantErr)):
				t.Fatalf("execute %q error = %v, want substring %q", strings.Join(tc.Argv, " "), err, tc.WantErr)
			}

			if tc.Assert != nil {
				tc.Assert(t, h, err)
			}
		})
	}
}

func RunClientCases(t *testing.T, cases []CommandCase) {
	t.Helper()

	for _, tc := range cases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			h := NewClientHarness(t)
			if tc.Setup != nil {
				tc.Setup(t, h)
			}

			err := h.ExecuteClient(tc.Argv...)
			switch {
			case tc.WantErr == "" && err != nil:
				t.Fatalf("execute %q failed: %v", strings.Join(tc.Argv, " "), err)
			case tc.WantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.WantErr)):
				t.Fatalf("execute %q error = %v, want substring %q", strings.Join(tc.Argv, " "), err, tc.WantErr)
			}

			if tc.Assert != nil {
				tc.Assert(t, h, err)
			}
		})
	}
}

func MustSingleCall[T any](t testing.TB, h *Harness, method string) (T, metadata.MD) {
	t.Helper()

	var zero T
	calls := h.Recorder.Calls()
	if len(calls) != 1 {
		t.Fatalf("primary call count = %d, want 1", len(calls))
	}
	if calls[0].Method != method {
		t.Fatalf("primary method = %s, want %s", calls[0].Method, method)
	}
	request, ok := calls[0].Request.(T)
	if !ok {
		t.Fatalf("request type = %T, want %T", calls[0].Request, zero)
	}
	return request, calls[0].Metadata
}

func MustSingleSessionEvent(t testing.TB, h *Harness) (*clientpb.Event, metadata.MD) {
	t.Helper()

	events := h.Recorder.SessionEvents()
	if len(events) != 1 {
		t.Fatalf("session event count = %d, want 1", len(events))
	}
	event, ok := events[0].Request.(*clientpb.Event)
	if !ok {
		t.Fatalf("session event type = %T, want *clientpb.Event", events[0].Request)
	}
	return event, events[0].Metadata
}

func RequireNoPrimaryCalls(t testing.TB, h *Harness) {
	t.Helper()
	if got := len(h.Recorder.Calls()); got != 0 {
		t.Fatalf("primary call count = %d, want 0", got)
	}
}

func RequireNoSessionEvents(t testing.TB, h *Harness) {
	t.Helper()
	if got := len(h.Recorder.SessionEvents()); got != 0 {
		t.Fatalf("session event count = %d, want 0", got)
	}
}

func RequireSessionID(t testing.TB, md metadata.MD, want string) {
	t.Helper()
	values := md.Get("session_id")
	if len(values) != 1 || values[0] != want {
		t.Fatalf("session_id metadata = %v, want %q", values, want)
	}
}

func RequireCallee(t testing.TB, md metadata.MD, want string) {
	t.Helper()
	values := md.Get("callee")
	if len(values) != 1 || values[0] != want {
		t.Fatalf("callee metadata = %v, want %q", values, want)
	}
}

func SessionClone(session *iomclient.Session) *clientpb.Session {
	if session == nil {
		return nil
	}
	return proto.Clone(session.Session).(*clientpb.Session)
}

func SessionRaw(rawID uint32) []byte {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, rawID)
	return buf
}

func containsUseFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--use" {
			return true
		}
	}
	return false
}
