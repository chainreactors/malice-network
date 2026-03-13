package service_test

import (
	"strings"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/client/command/testsupport"
	"google.golang.org/grpc/metadata"
)

func TestServiceCommandConformance(t *testing.T) {
	testsupport.RunCases(t, []testsupport.CommandCase{
		{
			Name: "list sends service list request",
			Argv: []string{consts.CommandService, "list"},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.Request](t, h, "ServiceList")
				if req.Name != consts.ModuleServiceList {
					t.Fatalf("service list name = %q, want %q", req.Name, consts.ModuleServiceList)
				}
				assertServiceTaskEvent(t, h, md, consts.ModuleServiceList)
			},
		},
		{
			Name: "create maps flags to service config",
			Argv: []string{
				consts.CommandService, "create",
				"--name", "spoolsvc",
				"--display", "Spool Service",
				"--path", `C:\Windows\spoolsvc.exe`,
				"--start_type", "Disabled",
				"--error", "Critical",
				"--account", `.\\svc-user`,
			},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.ServiceRequest](t, h, "ServiceCreate")
				if req.Type != consts.ModuleServiceCreate {
					t.Fatalf("service create type = %q, want %q", req.Type, consts.ModuleServiceCreate)
				}
				if req.Service == nil {
					t.Fatal("service create payload is nil")
				}
				if req.Service.Name != "spoolsvc" || req.Service.DisplayName != "Spool Service" || req.Service.ExecutablePath != `C:\Windows\spoolsvc.exe` {
					t.Fatalf("service create payload = %#v", req.Service)
				}
				if req.Service.StartType != 4 {
					t.Fatalf("start type = %d, want 4", req.Service.StartType)
				}
				if req.Service.ErrorControl != 3 {
					t.Fatalf("error control = %d, want 3", req.Service.ErrorControl)
				}
				if req.Service.AccountName != `.\\svc-user` {
					t.Fatalf("account = %q, want .\\\\svc-user", req.Service.AccountName)
				}
				assertServiceTaskEvent(t, h, md, consts.ModuleServiceCreate)
			},
		},
		{
			Name:    "create enforces required flags",
			Argv:    []string{consts.CommandService, "create", "--display", "missing"},
			WantErr: "required flag(s)",
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				if err == nil || !strings.Contains(err.Error(), "name") || !strings.Contains(err.Error(), "path") {
					t.Fatalf("service create error = %v, want required name and path flags", err)
				}
				testsupport.RequireNoPrimaryCalls(t, h)
				testsupport.RequireNoSessionEvents(t, h)
			},
		},
		{
			Name: "start uses service start rpc type",
			Argv: []string{consts.CommandService, "start", "Spooler"},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.ServiceRequest](t, h, "ServiceStart")
				if req.Type != consts.ModuleServiceStart {
					t.Fatalf("service start type = %q, want %q", req.Type, consts.ModuleServiceStart)
				}
				if req.Service == nil || req.Service.Name != "Spooler" {
					t.Fatalf("service start payload = %#v", req.Service)
				}
				assertServiceTaskEvent(t, h, md, consts.ModuleServiceStart)
			},
		},
		{
			Name: "stop forwards service name",
			Argv: []string{consts.CommandService, "stop", "Spooler"},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.ServiceRequest](t, h, "ServiceStop")
				if req.Type != consts.ModuleServiceStop || req.Service == nil || req.Service.Name != "Spooler" {
					t.Fatalf("service stop request = %#v", req)
				}
				assertServiceTaskEvent(t, h, md, consts.ModuleServiceStop)
			},
		},
		{
			Name: "query forwards service name",
			Argv: []string{consts.CommandService, "query", "Spooler"},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.ServiceRequest](t, h, "ServiceQuery")
				if req.Type != consts.ModuleServiceQuery || req.Service == nil || req.Service.Name != "Spooler" {
					t.Fatalf("service query request = %#v", req)
				}
				assertServiceTaskEvent(t, h, md, consts.ModuleServiceQuery)
			},
		},
		{
			Name: "delete forwards service name",
			Argv: []string{consts.CommandService, "delete", "Spooler"},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.ServiceRequest](t, h, "ServiceDelete")
				if req.Type != consts.ModuleServiceDelete || req.Service == nil || req.Service.Name != "Spooler" {
					t.Fatalf("service delete request = %#v", req)
				}
				assertServiceTaskEvent(t, h, md, consts.ModuleServiceDelete)
			},
		},
	})
}

func assertServiceTaskEvent(t testing.TB, h *testsupport.Harness, md metadata.MD, wantType string) {
	t.Helper()

	testsupport.RequireSessionID(t, md, h.Session.SessionId)
	testsupport.RequireCallee(t, md, consts.CalleeCMD)

	event, eventMD := testsupport.MustSingleSessionEvent(t, h)
	if event.Task == nil || event.Task.Type != wantType {
		t.Fatalf("service session event task = %#v, want type %q", event.Task, wantType)
	}
	testsupport.RequireSessionID(t, eventMD, h.Session.SessionId)
	testsupport.RequireCallee(t, eventMD, consts.CalleeCMD)
}
