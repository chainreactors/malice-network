package privilege_test

import (
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/client/command/testsupport"
	"google.golang.org/grpc/metadata"
)

func TestPrivilegeCommandConformance(t *testing.T) {
	testsupport.RunCases(t, []testsupport.CommandCase{
		{
			Name: "runas maps flags to runas request",
			Argv: []string{
				consts.ModuleRunas,
				"--username", "svc-build",
				"--domain", "CORP",
				"--password", "Password123!",
				"--path", `C:\Windows\System32\cmd.exe`,
				"--args", "/c whoami",
				"--use-profile",
				"--use-env",
				"--netonly",
			},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.RunAsRequest](t, h, "Runas")
				if req.Username != "svc-build" || req.Domain != "CORP" || req.Password != "Password123!" {
					t.Fatalf("runas identity = %#v", req)
				}
				if req.Program != `C:\Windows\System32\cmd.exe` || req.Args != "/c whoami" {
					t.Fatalf("runas program = %#v", req)
				}
				if !req.UseProfile || !req.UseEnv || !req.Netonly {
					t.Fatalf("runas flags = %#v, want use-profile/use-env/netonly true", req)
				}
				assertPrivilegeTaskEvent(t, h, md, consts.ModuleRunas)
			},
		},
		{
			Name: "privs sends module request",
			Argv: []string{consts.ModulePrivs},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.Request](t, h, "Privs")
				if req.Name != consts.ModulePrivs {
					t.Fatalf("privs request name = %q, want %q", req.Name, consts.ModulePrivs)
				}
				assertPrivilegeTaskEvent(t, h, md, consts.ModulePrivs)
			},
		},
		{
			Name: "getsystem sends module request",
			Argv: []string{consts.ModuleGetSystem},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.Request](t, h, "GetSystem")
				if req.Name != consts.ModuleGetSystem {
					t.Fatalf("getsystem request name = %q, want %q", req.Name, consts.ModuleGetSystem)
				}
				assertPrivilegeTaskEvent(t, h, md, consts.ModuleGetSystem)
			},
		},
		{
			Name: "rev2self sends module request",
			Argv: []string{consts.ModuleRev2Self},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.Request](t, h, "Rev2Self")
				if req.Name != consts.ModuleRev2Self {
					t.Fatalf("rev2self request name = %q, want %q", req.Name, consts.ModuleRev2Self)
				}
				assertPrivilegeTaskEvent(t, h, md, consts.ModuleRev2Self)
			},
		},
	})
}

func assertPrivilegeTaskEvent(t testing.TB, h *testsupport.Harness, md metadata.MD, wantType string) {
	t.Helper()

	testsupport.RequireSessionID(t, md, h.Session.SessionId)
	testsupport.RequireCallee(t, md, consts.CalleeCMD)

	event, eventMD := testsupport.MustSingleSessionEvent(t, h)
	if event.Task == nil || event.Task.Type != wantType {
		t.Fatalf("privilege session event task = %#v, want type %q", event.Task, wantType)
	}
	testsupport.RequireSessionID(t, eventMD, h.Session.SessionId)
	testsupport.RequireCallee(t, eventMD, consts.CalleeCMD)
}
