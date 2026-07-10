package context_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/command/testsupport"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"google.golang.org/grpc"
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
				h.Recorder.OnContextStream("SyncStream", func(ctx context.Context, request any) (grpc.ServerStreamingClient[clientpb.ContextChunk], error) {
					return nil, context.DeadlineExceeded
				})
			},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, _ := testsupport.MustSingleCall[*clientpb.Sync](t, h, "SyncStream")
				if req.ContextId != "ctx-1" {
					t.Fatalf("sync context id = %q, want ctx-1", req.ContextId)
				}
				testsupport.RequireNoSessionEvents(t, h)
			},
		},
		{
			Name: "sync streams file-backed context content",
			Argv: []string{consts.CommandSync, "ctx-1"},
			Setup: func(t testing.TB, h *testsupport.Harness) {
				h.Recorder.OnContextStream("SyncStream", func(ctx context.Context, request any) (grpc.ServerStreamingClient[clientpb.ContextChunk], error) {
					header := &clientpb.Context{
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
					}
					return contextChunkStream(
						&clientpb.ContextChunk{Header: header, TotalSize: 4},
						&clientpb.ContextChunk{Content: []byte("bo"), TotalSize: 4},
						&clientpb.ContextChunk{Content: []byte("dy"), Offset: 2, TotalSize: 4, Eof: true},
					), nil
				})
			},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, _ := testsupport.MustSingleCall[*clientpb.Sync](t, h, "SyncStream")
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
		{
			Name: "sync falls back to unary rpc for older servers",
			Argv: []string{consts.CommandSync, "ctx-1"},
			Setup: func(t testing.TB, h *testsupport.Harness) {
				h.Recorder.OnContext("Sync", func(ctx context.Context, request any) (*clientpb.Context, error) {
					return downloadContext("ctx-1", "fallback.bin", []byte("fallback")), nil
				})
			},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				calls := h.Recorder.Calls()
				if len(calls) != 2 || calls[0].Method != "SyncStream" || calls[1].Method != "Sync" {
					t.Fatalf("rpc calls = %#v, want SyncStream then Sync", calls)
				}
				data, readErr := os.ReadFile(filepath.Join(assets.GetTempDir(), "ctx-1_fallback.bin"))
				if readErr != nil || string(data) != "fallback" {
					t.Fatalf("fallback file = %q, %v", data, readErr)
				}
			},
		},
		{
			Name:    "sync rejects discontinuous stream offsets",
			Argv:    []string{consts.CommandSync, "ctx-1"},
			WantErr: "unexpected stream offset",
			Setup: func(t testing.TB, h *testsupport.Harness) {
				h.Recorder.OnContextStream("SyncStream", func(ctx context.Context, request any) (grpc.ServerStreamingClient[clientpb.ContextChunk], error) {
					header := downloadContext("ctx-1", "broken.bin", nil)
					return contextChunkStream(
						&clientpb.ContextChunk{Header: header, TotalSize: 4},
						&clientpb.ContextChunk{Content: []byte("body"), Offset: 1, TotalSize: 4, Eof: true},
					), nil
				})
			},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				path := filepath.Join(assets.GetTempDir(), "ctx-1_broken.bin")
				if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
					t.Fatalf("partial file %s should not exist: %v", path, statErr)
				}
			},
		},
	})
}

func contextChunkStream(chunks ...*clientpb.ContextChunk) grpc.ServerStreamingClient[clientpb.ContextChunk] {
	index := 0
	return &testsupport.ContextChunkStream{RecvFunc: func() (*clientpb.ContextChunk, error) {
		if index == len(chunks) {
			return nil, io.EOF
		}
		chunk := chunks[index]
		index++
		return chunk, nil
	}}
}

func downloadContext(id, name string, content []byte) *clientpb.Context {
	return &clientpb.Context{
		Id:   id,
		Type: consts.ContextDownload,
		Value: output.MarshalContext(&output.DownloadContext{
			FileDescriptor: &output.FileDescriptor{
				Name:       name,
				FilePath:   "/remote/" + name,
				TargetPath: "/remote/" + name,
				Size:       int64(len(content)),
			},
		}),
		Content: content,
	}
}
