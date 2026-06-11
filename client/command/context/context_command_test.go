package context_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/command/testsupport"
	"github.com/chainreactors/malice-network/helper/utils/output"
)

func TestContextCommandConformance(t *testing.T) {
	testsupport.RunClientCases(t, []testsupport.CommandCase{
		{
			Name:    "context delete requires explicit confirmation in static mode",
			Argv:    []string{"context", "delete", "ctx-1"},
			WantErr: "interactive confirmation",
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				testsupport.RequireNoPrimaryCalls(t, h)
				testsupport.RequireNoSessionEvents(t, h)
			},
		},
		{
			Name: "context delete forwards id when confirmed",
			Argv: []string{"context", "delete", "ctx-1", "--yes"},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, _ := testsupport.MustSingleCall[*clientpb.Context](t, h, "DeleteContext")
				if req.Id != "ctx-1" {
					t.Fatalf("delete context id = %q, want ctx-1", req.Id)
				}
				testsupport.RequireNoSessionEvents(t, h)
			},
		},
		{
			Name:    "sync propagates rpc errors",
			Argv:    []string{consts.CommandSync, "ctx-1"},
			WantErr: "sync context failed",
			Setup: func(t testing.TB, h *testsupport.Harness) {
				h.Recorder.OnContext("Sync", func(ctx context.Context, request any) (*clientpb.Context, error) {
					return nil, context.DeadlineExceeded
				})
			},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, _ := testsupport.MustSingleCall[*clientpb.Sync](t, h, "Sync")
				if req.ContextId != "ctx-1" {
					t.Fatalf("sync context id = %q, want ctx-1", req.ContextId)
				}
				testsupport.RequireNoSessionEvents(t, h)
			},
		},
		{
			Name: "sync writes file-backed context content",
			Argv: []string{consts.CommandSync, "ctx-1"},
			Setup: func(t testing.TB, h *testsupport.Harness) {
				h.Recorder.OnContext("Sync", func(ctx context.Context, request any) (*clientpb.Context, error) {
					return &clientpb.Context{
						Id:   "ctx-1",
						Type: consts.ContextDownload,
						Value: output.MarshalContext(&output.DownloadContext{
							FileDescriptor: &output.FileDescriptor{
								Name:       "capture.bin",
								FilePath:   "/remote/capture.bin",
								TargetPath: "/remote/capture.bin",
								Size:       4,
							},
						}),
						Content: []byte("body"),
					}, nil
				})
			},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, _ := testsupport.MustSingleCall[*clientpb.Sync](t, h, "Sync")
				if req.ContextId != "ctx-1" {
					t.Fatalf("sync context id = %q, want ctx-1", req.ContextId)
				}

				savePath := filepath.Join(assets.GetTempDir(), "ctx-1_capture.bin")
				data, readErr := os.ReadFile(savePath)
				if readErr != nil {
					t.Fatalf("expected synced file at %s: %v", savePath, readErr)
				}
				if string(data) != "body" {
					t.Fatalf("synced file content = %q, want body", data)
				}
				testsupport.RequireNoSessionEvents(t, h)
			},
		},
	})
}
