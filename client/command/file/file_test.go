package file_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/client/command/testsupport"
	"google.golang.org/grpc/metadata"
)

func TestFileCommandConformance(t *testing.T) {
	testsupport.RunCases(t, []testsupport.CommandCase{
		{
			Name: "download forwards path and basename",
			Argv: []string{consts.ModuleDownload, `C:\Temp\archive.zip`},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.DownloadRequest](t, h, "Download")
				if req.Path != `C:\Temp\archive.zip` || req.Name != "archive.zip" || req.Dir {
					t.Fatalf("download request = %#v", req)
				}
				assertFileTaskEvent(t, h, md, consts.ModuleDownload)
			},
		},
		{
			Name: "download dir forwards recursive flag in request body",
			Argv: []string{consts.ModuleDownload, "--dir", `C:\Temp\reports`},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.DownloadRequest](t, h, "Download")
				if req.Path != `C:\Temp\reports` || req.Name != "reports" || !req.Dir {
					t.Fatalf("download dir request = %#v", req)
				}
				assertFileTaskEvent(t, h, md, consts.ModuleDownload)
			},
		},
	})
}

func TestFileUploadCommandCases(t *testing.T) {
	h := testsupport.NewHarness(t)

	localPath := filepath.Join(t.TempDir(), "payload.bin")
	if err := os.WriteFile(localPath, []byte("payload-body"), 0o600); err != nil {
		t.Fatalf("write local upload fixture: %v", err)
	}

	if err := h.Execute(consts.ModuleUpload, localPath, `C:\Temp\payload.bin`, "--priv", "0600", "--hidden"); err != nil {
		t.Fatalf("upload execute failed: %v", err)
	}

	req, md := testsupport.MustSingleCall[*implantpb.UploadRequest](t, h, "Upload")
	if req.Name != "payload.bin" || req.Target != `C:\Temp\payload.bin` {
		t.Fatalf("upload request = %#v", req)
	}
	if req.Priv != 0o600 || !req.Hidden {
		t.Fatalf("upload flags = %#v, want priv=0600 hidden=true", req)
	}
	if string(req.Data) != "payload-body" {
		t.Fatalf("upload data = %q, want payload-body", string(req.Data))
	}
	assertFileTaskEvent(t, h, md, consts.ModuleUpload)
}

func TestFileUploadCommandErrors(t *testing.T) {
	t.Run("upload rejects missing local file", func(t *testing.T) {
		h := testsupport.NewHarness(t)
		err := h.Execute(consts.ModuleUpload, filepath.Join(t.TempDir(), "missing.bin"), `C:\Temp\remote.bin`)
		if err == nil || (!strings.Contains(strings.ToLower(err.Error()), "cannot find") && !strings.Contains(strings.ToLower(err.Error()), "no such file")) {
			t.Fatalf("upload missing file error = %v, want file-not-found error", err)
		}
		testsupport.RequireNoPrimaryCalls(t, h)
		testsupport.RequireNoSessionEvents(t, h)
	})

	t.Run("upload rejects invalid octal privilege", func(t *testing.T) {
		h := testsupport.NewHarness(t)
		localPath := filepath.Join(t.TempDir(), "payload.bin")
		if err := os.WriteFile(localPath, []byte("payload-body"), 0o600); err != nil {
			t.Fatalf("write local upload fixture: %v", err)
		}
		err := h.Execute(consts.ModuleUpload, localPath, `C:\Temp\remote.bin`, "--priv", "bad")
		if err == nil || !strings.Contains(err.Error(), "invalid syntax") {
			t.Fatalf("upload invalid priv error = %v, want invalid syntax", err)
		}
		testsupport.RequireNoPrimaryCalls(t, h)
		testsupport.RequireNoSessionEvents(t, h)
	})
}

func assertFileTaskEvent(t testing.TB, h *testsupport.Harness, md metadata.MD, wantType string) {
	t.Helper()

	testsupport.RequireSessionID(t, md, h.Session.SessionId)
	testsupport.RequireCallee(t, md, consts.CalleeCMD)

	event, eventMD := testsupport.MustSingleSessionEvent(t, h)
	if event.Task == nil || event.Task.Type != wantType {
		t.Fatalf("file session event task = %#v, want type %q", event.Task, wantType)
	}
	testsupport.RequireSessionID(t, eventMD, h.Session.SessionId)
	testsupport.RequireCallee(t, eventMD, consts.CalleeCMD)
}
