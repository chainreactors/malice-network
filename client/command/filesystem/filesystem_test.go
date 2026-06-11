package filesystem_test

import (
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/client/command/testsupport"
	"google.golang.org/grpc/metadata"
)

func TestFilesystemCommandConformance(t *testing.T) {
	testsupport.RunCases(t, []testsupport.CommandCase{
		{
			Name: "pwd sends request without input",
			Argv: []string{consts.ModulePwd},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.Request](t, h, "Pwd")
				if req.Name != consts.ModulePwd || req.Input != "" {
					t.Fatalf("pwd request = %#v, want name pwd and empty input", req)
				}
				assertFilesystemTaskEvent(t, h, md, consts.ModulePwd)
			},
		},
		{
			Name: "cat forwards file path",
			Argv: []string{consts.ModuleCat, `C:\Temp\notes.txt`},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.Request](t, h, "Cat")
				if req.Name != consts.ModuleCat || req.Input != `C:\Temp\notes.txt` {
					t.Fatalf("cat request = %#v", req)
				}
				assertFilesystemTaskEvent(t, h, md, consts.ModuleCat)
			},
		},
		{
			Name: "cd forwards target path",
			Argv: []string{consts.ModuleCd, `C:\Windows\Temp`},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.Request](t, h, "Cd")
				if req.Name != consts.ModuleCd || req.Input != `C:\Windows\Temp` {
					t.Fatalf("cd request = %#v", req)
				}
				assertFilesystemTaskEvent(t, h, md, consts.ModuleCd)
			},
		},
		{
			Name: "cp preserves source and target order",
			Argv: []string{consts.ModuleCp, `C:\Temp\source.txt`, `C:\Temp\target.txt`},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.Request](t, h, "Cp")
				if req.Name != consts.ModuleCp || len(req.Args) != 2 || req.Args[0] != `C:\Temp\source.txt` || req.Args[1] != `C:\Temp\target.txt` {
					t.Fatalf("cp request = %#v", req)
				}
				assertFilesystemTaskEvent(t, h, md, consts.ModuleCp)
			},
		},
		{
			Name: "ls defaults to current directory marker",
			Argv: []string{consts.ModuleLs},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.Request](t, h, "Ls")
				if req.Name != consts.ModuleLs || req.Input != "./" {
					t.Fatalf("ls request = %#v, want input ./", req)
				}
				assertFilesystemTaskEvent(t, h, md, consts.ModuleLs)
			},
		},
		{
			Name: "mkdir forwards directory path",
			Argv: []string{consts.ModuleMkdir, `C:\Temp\malice-e2e`},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.Request](t, h, "Mkdir")
				if req.Name != consts.ModuleMkdir || req.Input != `C:\Temp\malice-e2e` {
					t.Fatalf("mkdir request = %#v", req)
				}
				assertFilesystemTaskEvent(t, h, md, consts.ModuleMkdir)
			},
		},
		{
			Name: "touch forwards file path",
			Argv: []string{consts.ModuleTouch, `C:\Temp\marker.txt`},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.Request](t, h, "Touch")
				if req.Name != consts.ModuleTouch || req.Input != `C:\Temp\marker.txt` {
					t.Fatalf("touch request = %#v", req)
				}
				assertFilesystemTaskEvent(t, h, md, consts.ModuleTouch)
			},
		},
		{
			Name: "mv preserves source and target order",
			Argv: []string{consts.ModuleMv, `C:\Temp\old.txt`, `C:\Temp\new.txt`},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.Request](t, h, "Mv")
				if req.Name != consts.ModuleMv || len(req.Args) != 2 || req.Args[0] != `C:\Temp\old.txt` || req.Args[1] != `C:\Temp\new.txt` {
					t.Fatalf("mv request = %#v", req)
				}
				assertFilesystemTaskEvent(t, h, md, consts.ModuleMv)
			},
		},
		{
			Name: "rm forwards file path",
			Argv: []string{consts.ModuleRm, `C:\Temp\obsolete.txt`},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.Request](t, h, "Rm")
				if req.Name != consts.ModuleRm || req.Input != `C:\Temp\obsolete.txt` {
					t.Fatalf("rm request = %#v", req)
				}
				assertFilesystemTaskEvent(t, h, md, consts.ModuleRm)
			},
		},
		{
			Name: "enum drivers sends module request",
			Argv: []string{consts.ModuleEnumDrivers},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.Request](t, h, "EnumDrivers")
				if req.Name != consts.ModuleEnumDrivers {
					t.Fatalf("enum drivers request name = %q, want %q", req.Name, consts.ModuleEnumDrivers)
				}
				assertFilesystemTaskEvent(t, h, md, consts.ModuleEnumDrivers)
			},
		},
	})
}

func assertFilesystemTaskEvent(t testing.TB, h *testsupport.Harness, md metadata.MD, wantType string) {
	t.Helper()

	testsupport.RequireSessionID(t, md, h.Session.SessionId)
	testsupport.RequireCallee(t, md, consts.CalleeCMD)

	event, eventMD := testsupport.MustSingleSessionEvent(t, h)
	if event.Task == nil || event.Task.Type != wantType {
		t.Fatalf("filesystem session event task = %#v, want type %q", event.Task, wantType)
	}
	testsupport.RequireSessionID(t, eventMD, h.Session.SessionId)
	testsupport.RequireCallee(t, eventMD, consts.CalleeCMD)
}
