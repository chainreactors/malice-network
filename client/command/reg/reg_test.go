package reg_test

import (
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/client/command/testsupport"
	"google.golang.org/grpc/metadata"
)

func TestRegCommandConformance(t *testing.T) {
	testsupport.RunCases(t, []testsupport.CommandCase{
		{
			Name: "query normalizes windows registry path",
			Argv: []string{consts.CommandReg, "query", "HKLM/SOFTWARE/Test/Path", "ValueName"},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.RegistryRequest](t, h, "RegQuery")
				if req.Type != consts.ModuleRegQuery {
					t.Fatalf("reg query type = %q, want %q", req.Type, consts.ModuleRegQuery)
				}
				if req.Registry == nil || req.Registry.Hive != "HKLM" || req.Registry.Path != `SOFTWARE\Test\Path` || req.Registry.Key != "ValueName" {
					t.Fatalf("reg query payload = %#v", req.Registry)
				}
				assertRegTaskEvent(t, h, md, consts.ModuleRegQuery)
			},
		},
		{
			Name: "add defaults to reg_sz",
			Argv: []string{consts.CommandReg, "add", `HKLM\SOFTWARE\Test`, "--value", "Greeting", "--data", "hello"},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.RegistryWriteRequest](t, h, "RegAdd")
				if req.Hive != "HKLM" || req.Path != `SOFTWARE\Test` || req.Key != "Greeting" {
					t.Fatalf("reg add payload = %#v", req)
				}
				if req.Regtype != 1 || req.StringValue != "hello" {
					t.Fatalf("reg add reg_sz = %#v, want regtype 1 and string hello", req)
				}
				assertRegTaskEvent(t, h, md, consts.ModuleRegAdd)
			},
		},
		{
			Name: "add decodes reg_binary",
			Argv: []string{consts.CommandReg, "add", `HKLM\SOFTWARE\Test`, "--value", "Blob", "--type", "REG_BINARY", "--data", "aa bb cc"},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.RegistryWriteRequest](t, h, "RegAdd")
				if req.Regtype != 3 || string(req.ByteValue) != string([]byte{0xaa, 0xbb, 0xcc}) {
					t.Fatalf("reg binary payload = %#v, want decoded bytes", req)
				}
				assertRegTaskEvent(t, h, md, consts.ModuleRegAdd)
			},
		},
		{
			Name: "add parses reg_dword",
			Argv: []string{consts.CommandReg, "add", `HKLM\SOFTWARE\Test`, "--value", "Enabled", "--type", "REG_DWORD", "--data", "0x10"},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.RegistryWriteRequest](t, h, "RegAdd")
				if req.Regtype != 4 || req.DwordValue != 16 {
					t.Fatalf("reg dword payload = %#v, want regtype 4 and value 16", req)
				}
				assertRegTaskEvent(t, h, md, consts.ModuleRegAdd)
			},
		},
		{
			Name: "add parses reg_qword",
			Argv: []string{consts.CommandReg, "add", `HKLM\SOFTWARE\Test`, "--value", "Large", "--type", "REG_QWORD", "--data", "0x20"},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.RegistryWriteRequest](t, h, "RegAdd")
				if req.Regtype != 11 || req.QwordValue != 32 {
					t.Fatalf("reg qword payload = %#v, want regtype 11 and value 32", req)
				}
				assertRegTaskEvent(t, h, md, consts.ModuleRegAdd)
			},
		},
		{
			Name: "delete forwards hive path and key",
			Argv: []string{consts.CommandReg, "delete", `HKLM\SOFTWARE\Test`, "ValueName"},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.RegistryRequest](t, h, "RegDelete")
				if req.Type != consts.ModuleRegDelete || req.Registry == nil || req.Registry.Hive != "HKLM" || req.Registry.Path != `SOFTWARE\Test` || req.Registry.Key != "ValueName" {
					t.Fatalf("reg delete payload = %#v", req.Registry)
				}
				assertRegTaskEvent(t, h, md, consts.ModuleRegDelete)
			},
		},
		{
			Name: "list_key forwards registry location",
			Argv: []string{consts.CommandReg, "list_key", `HKLM\SOFTWARE\Test`},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.RegistryRequest](t, h, "RegListKey")
				if req.Type != consts.ModuleRegListKey || req.Registry == nil || req.Registry.Hive != "HKLM" || req.Registry.Path != `SOFTWARE\Test` {
					t.Fatalf("reg list_key payload = %#v", req.Registry)
				}
				assertRegTaskEvent(t, h, md, consts.ModuleRegListKey)
			},
		},
		{
			Name: "list_value forwards registry location",
			Argv: []string{consts.CommandReg, "list_value", `HKLM\SOFTWARE\Test`},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.RegistryRequest](t, h, "RegListValue")
				if req.Type != consts.ModuleRegListValue || req.Registry == nil || req.Registry.Hive != "HKLM" || req.Registry.Path != `SOFTWARE\Test` {
					t.Fatalf("reg list_value payload = %#v", req.Registry)
				}
				assertRegTaskEvent(t, h, md, consts.ModuleRegListValue)
			},
		},
	})
}

func assertRegTaskEvent(t testing.TB, h *testsupport.Harness, md metadata.MD, wantType string) {
	t.Helper()

	testsupport.RequireSessionID(t, md, h.Session.SessionId)
	testsupport.RequireCallee(t, md, consts.CalleeCMD)

	event, eventMD := testsupport.MustSingleSessionEvent(t, h)
	if event.Task == nil || event.Task.Type != wantType {
		t.Fatalf("reg session event task = %#v, want type %q", event.Task, wantType)
	}
	testsupport.RequireSessionID(t, eventMD, h.Session.SessionId)
	testsupport.RequireCallee(t, eventMD, consts.CalleeCMD)
}
