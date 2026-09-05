package context

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func SyncCmd(cmd *cobra.Command, con *core.Console) error {
	contextID := cmd.Flags().Arg(0)
	if contextID == "" {
		return fmt.Errorf("context_id is required")
	}

	req := &clientpb.Sync{ContextId: contextID}
	if err := syncContextStream(con, req); err != nil {
		if status.Code(err) == codes.Unimplemented {
			if unaryErr := syncContextUnary(con, req); unaryErr != nil {
				return fmt.Errorf("sync context failed: %w", unaryErr)
			}
			return nil
		}
		return fmt.Errorf("sync context failed: %w", err)
	}
	return nil
}

func syncContextStream(con *core.Console, req *clientpb.Sync) error {
	stream, err := con.Rpc.SyncStream(con.Context(), req)
	if err != nil {
		return err
	}
	return receiveContextStream(con, stream)
}

func syncContextUnary(con *core.Console, req *clientpb.Sync) error {
	ctx, err := con.Rpc.Sync(con.Context(), req)
	if err != nil {
		return err
	}

	filename, fileBacked, err := describeSyncedContext(con, ctx)
	if err != nil || !fileBacked {
		return err
	}

	savePath, err := syncedContextPath(ctx.GetId(), filename)
	if err != nil {
		return err
	}
	if err := fileutils.AtomicWriteFile(savePath, ctx.GetContent(), 0o644); err != nil {
		return fmt.Errorf("write file failed: %w", err)
	}
	con.Log.Infof("File saved to: %s\n", savePath)
	return nil
}

func receiveContextStream(con *core.Console, stream clientrpc.MaliceRPC_SyncStreamClient) error {
	first, err := stream.Recv()
	if err != nil {
		return err
	}
	if first.GetHeader() == nil {
		return errors.New("stream header is missing")
	}
	if len(first.GetContent()) != 0 || first.GetOffset() != 0 {
		return errors.New("stream header contains content")
	}
	if first.GetTotalSize() < 0 {
		return fmt.Errorf("invalid stream size %d", first.GetTotalSize())
	}

	header := first.GetHeader()
	filename, fileBacked, err := describeSyncedContext(con, header)
	if err != nil {
		return err
	}
	if !fileBacked {
		if first.GetTotalSize() != 0 || !first.GetEof() {
			return errors.New("non-file context contains streamed content")
		}
		return nil
	}

	savePath, err := syncedContextPath(header.GetId(), filename)
	if err != nil {
		return err
	}
	writer, err := newContextStreamWriter(savePath)
	if err != nil {
		return fmt.Errorf("create file failed: %w", err)
	}
	defer writer.abort()

	if first.GetTotalSize() == 0 {
		if !first.GetEof() {
			return errors.New("empty context stream is missing eof")
		}
		if err := writer.commit(); err != nil {
			return fmt.Errorf("write file failed: %w", err)
		}
		con.Log.Infof("File saved to: %s\n", savePath)
		return nil
	}
	if first.GetEof() {
		return errors.New("non-empty context stream ended in header")
	}

	written := int64(0)
	for {
		chunk, recvErr := stream.Recv()
		if recvErr != nil {
			if errors.Is(recvErr, io.EOF) {
				return io.ErrUnexpectedEOF
			}
			return recvErr
		}
		if chunk.GetHeader() != nil {
			return errors.New("stream contains duplicate header")
		}
		if chunk.GetTotalSize() != first.GetTotalSize() {
			return fmt.Errorf("stream size changed from %d to %d", first.GetTotalSize(), chunk.GetTotalSize())
		}
		if chunk.GetOffset() != written {
			return fmt.Errorf("unexpected stream offset %d, want %d", chunk.GetOffset(), written)
		}
		if len(chunk.GetContent()) == 0 {
			return errors.New("stream contains empty content chunk")
		}
		if written+int64(len(chunk.GetContent())) > first.GetTotalSize() {
			return fmt.Errorf("stream content exceeds total size %d", first.GetTotalSize())
		}
		if _, err := writer.Write(chunk.GetContent()); err != nil {
			return fmt.Errorf("write file failed: %w", err)
		}
		written += int64(len(chunk.GetContent()))

		if !chunk.GetEof() {
			continue
		}
		if written != first.GetTotalSize() {
			return fmt.Errorf("stream ended at %d bytes, want %d", written, first.GetTotalSize())
		}
		if err := writer.commit(); err != nil {
			return fmt.Errorf("write file failed: %w", err)
		}
		con.Log.Infof("File saved to: %s\n", savePath)
		return nil
	}
}

func describeSyncedContext(con *core.Console, ctx *clientpb.Context) (string, bool, error) {
	parsed, err := output.ParseContext(ctx.GetType(), ctx.GetValue())
	if err != nil {
		return "", false, fmt.Errorf("parse context failed: %w", err)
	}
	con.Log.Infof("Context: \n%s\n", parsed.String())

	switch value := parsed.(type) {
	case *output.ScreenShotContext:
		return value.Name, true, nil
	case *output.DownloadContext:
		return value.Name, true, nil
	case *output.KeyLoggerContext:
		return value.Name, true, nil
	case *output.UploadContext:
		return value.Name, true, nil
	case *output.MediaContext:
		return value.Name, true, nil
	default:
		return "", false, nil
	}
}

func syncedContextPath(contextID, filename string) (string, error) {
	safeID, err := fileutils.SanitizeBasename(contextID)
	if err != nil {
		return "", fmt.Errorf("invalid context id: %w", err)
	}
	safeName, err := fileutils.SanitizeBasename(filename)
	if err != nil {
		return "", fmt.Errorf("invalid context filename: %w", err)
	}
	return fileutils.SafeJoin(assets.GetTempDir(), fmt.Sprintf("%s_%s", safeID, safeName))
}

type contextStreamWriter struct {
	file      *os.File
	tempPath  string
	finalPath string
	committed bool
}

func newContextStreamWriter(finalPath string) (*contextStreamWriter, error) {
	file, err := os.CreateTemp(filepath.Dir(finalPath), ".context-*.part")
	if err != nil {
		return nil, err
	}
	return &contextStreamWriter{file: file, tempPath: file.Name(), finalPath: finalPath}, nil
}

func (w *contextStreamWriter) Write(content []byte) (int, error) {
	return w.file.Write(content)
}

func (w *contextStreamWriter) commit() error {
	if err := w.file.Chmod(0o644); err != nil {
		return err
	}
	if err := w.file.Sync(); err != nil {
		return err
	}
	if err := w.file.Close(); err != nil {
		return err
	}
	if err := os.Rename(w.tempPath, w.finalPath); err != nil {
		return err
	}
	w.committed = true
	return nil
}

func (w *contextStreamWriter) abort() {
	if w.committed {
		return
	}
	_ = w.file.Close()
	_ = os.Remove(w.tempPath)
}
