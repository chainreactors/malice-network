package rpc

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	types "github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
)

const maxChunkRetries = 3

var rpcFileSaveContext = db.SaveContext

// uploadBlockCount returns how many implant Block messages are needed for a
// reader-based upload of totalSize bytes. Zero-byte uploads need one final
// empty Block (implant treats empty UploadRequest.data as streamed mode).
func uploadBlockCount(totalSize int64, packetLength int) int {
	if totalSize <= 0 {
		return 1
	}
	if packetLength <= 0 {
		return 1
	}
	return int((totalSize + int64(packetLength) - 1) / int64(packetLength))
}

// shouldInlineUpload is true when the entire payload fits in one implant packet
// and is non-empty. Empty payloads must use the streamed Block path.
func shouldInlineUpload(totalSize int64, packetLength int) bool {
	if totalSize <= 0 {
		return false
	}
	if packetLength <= 0 {
		return true
	}
	return totalSize <= int64(packetLength)
}

// cloneUploadMetadata copies implant-facing upload fields without payload data.
// Override must be preserved for multi-block delivery.
func cloneUploadMetadata(req *implantpb.UploadRequest) *implantpb.UploadRequest {
	if req == nil {
		return &implantpb.UploadRequest{}
	}
	return &implantpb.UploadRequest{
		Name:     req.Name,
		Target:   req.Target,
		Priv:     req.Priv,
		Hidden:   req.Hidden,
		Override: req.Override,
	}
}

func downloadChunkCount(size int, chunkSize int) int {
	if chunkSize <= 0 {
		return 0
	}
	if size <= 0 {
		return 0
	}
	return (size + chunkSize - 1) / chunkSize
}

func scanDownloadChunks(tempDir string, total int) (int32, bool, error) {
	if total <= 0 {
		return 1, true, nil
	}
	for i := 1; i <= total; i++ {
		chunkFile := filepath.Join(tempDir, fmt.Sprintf("%d.chunk", i))
		_, err := os.Stat(chunkFile)
		if err == nil {
			continue
		}
		if os.IsNotExist(err) {
			return int32(i), false, nil
		}
		return 0, false, fmt.Errorf("stat chunk %d: %w", i, err)
	}
	return int32(total), true, nil
}

func isChunkSizeCompatible(tempDir string, bufferSize int, total int) bool {
	if total <= 1 {
		return true
	}
	firstChunk := filepath.Join(tempDir, "1.chunk")
	info, err := os.Stat(firstChunk)
	if err != nil {
		return true
	}
	return info.Size() == int64(bufferSize)
}

// Upload - Upload a file from the remote file system.
// Large payloads still arrive as a full unary request for legacy clients; the
// downstream implant path is reader-based so UploadChunk staging can share it.
func (rpc *Server) Upload(ctx context.Context, req *implantpb.UploadRequest) (*clientpb.Task, error) {
	if req == nil {
		return nil, types.ErrMissingRequestField
	}
	return rpc.dispatchUpload(ctx, req, bytes.NewReader(req.Data), int64(len(req.Data)))
}

func saveUploadContext(greq *GenericRequest, meta *implantpb.UploadRequest, totalSize int64) {
	if greq == nil || greq.Task == nil || greq.Session == nil || meta == nil {
		return
	}
	v := &output.UploadContext{
		FileDescriptor: &output.FileDescriptor{
			Name:       meta.Name,
			TargetPath: meta.Target,
			Abstract:   fmt.Sprintf("upload -%d -%t", meta.Priv, meta.Hidden),
			Size:       totalSize,
		},
	}
	ictx, err := rpcFileSaveContext(&clientpb.Context{
		Task:    greq.Task.ToProtobuf(),
		Session: greq.Session.ToProtobuf(),
		Type:    consts.ContextUpload,
		Value:   v.Marshal(),
	})
	if err != nil {
		logs.Log.Errorf("cannot create task %d, %s in db", greq.Task.Id, err.Error())
		return
	}
	core.PushContextEvent(consts.ContextUpload, ictx)
}

// closeIfCloser closes reader when it implements io.Closer (e.g. *os.File from staging).
func closeIfCloser(reader io.Reader) {
	if c, ok := reader.(io.Closer); ok && c != nil {
		_ = c.Close()
	}
}

// dispatchUpload sends an upload to the implant using either a single inline
// UploadRequest (payload fits in one packet) or a metadata-only request followed
// by sequential Block/ACK messages read from reader.
//
// If reader implements io.Closer, it is closed when delivery finishes (inline
// path after ReadAll; streamed path in the async handler cleanup).
func (rpc *Server) dispatchUpload(ctx context.Context, meta *implantpb.UploadRequest, reader io.Reader, totalSize int64) (*clientpb.Task, error) {
	if meta == nil {
		return nil, types.ErrMissingRequestField
	}
	if reader == nil {
		reader = bytes.NewReader(nil)
	}
	if totalSize < 0 {
		totalSize = 0
	}

	packetLength := getPacketLength(ctx)
	if shouldInlineUpload(totalSize, packetLength) {
		defer closeIfCloser(reader)
		payload, err := io.ReadAll(io.LimitReader(reader, totalSize))
		if err != nil {
			return nil, fmt.Errorf("read upload payload: %w", err)
		}
		if int64(len(payload)) != totalSize {
			return nil, fmt.Errorf("upload payload size mismatch: got %d, want %d", len(payload), totalSize)
		}
		req := cloneUploadMetadata(meta)
		req.Data = payload

		greq, err := newGenericRequest(ctx, req)
		if err != nil {
			return nil, err
		}
		ch, err := rpc.GenericHandler(ctx, greq)
		if err != nil {
			return nil, err
		}
		greq.HandlerResponse(ch, types.MsgAck, func(spite *implantpb.Spite) {
			saveUploadContext(greq, meta, totalSize)
		})
		return greq.Task.ToProtobuf(), nil
	}

	count := uploadBlockCount(totalSize, packetLength)
	metaOnly := cloneUploadMetadata(meta)
	greq, err := newGenericRequest(ctx, metaOnly, count)
	if err != nil {
		closeIfCloser(reader)
		return nil, err
	}
	in, out, err := rpc.StreamGenericHandler(ctx, greq)
	if err != nil {
		closeIfCloser(reader)
		return nil, err
	}

	// Touch deadline so multi-block work is not reported timed-out while still
	// actively progressing (Deadline is observational, not a cancel timer).
	extendTaskDeadline := func() {
		if greq.Task != nil {
			greq.Task.ExtendDeadline(time.Now().Add(consts.MinTimeout))
		}
	}

	runTaskHandler(greq.Task, func() error {
		stat, ok := recvSpite(greq.Task.Ctx, out)
		if !ok {
			return ErrTaskContextCancelled
		}
		if err := types.HandleMaleficError(stat); err != nil {
			return buildTaskError(err)
		}
		extendTaskDeadline()

		packetLen := greq.Session.GetPacketLength()
		if packetLen <= 0 {
			// Fall back to a single-read path if session has no packet limit.
			payload, readErr := io.ReadAll(reader)
			if readErr != nil {
				return fmt.Errorf("read upload payload: %w", readErr)
			}
			return sendUploadBlocks(greq, in, out, [][]byte{payload}, totalSize, meta, extendTaskDeadline)
		}

		buf := make([]byte, packetLen)
		blockID := 0
		var remaining int64 = totalSize
		// Zero-byte uploads: send one empty final Block after the initial ACK.
		if totalSize == 0 {
			return sendUploadBlocks(greq, in, out, [][]byte{nil}, totalSize, meta, extendTaskDeadline)
		}

		for remaining > 0 {
			toRead := packetLen
			if remaining < int64(packetLen) {
				toRead = int(remaining)
			}
			n, readErr := io.ReadFull(reader, buf[:toRead])
			if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
				return fmt.Errorf("read upload block %d: %w", blockID, readErr)
			}
			if n == 0 {
				return fmt.Errorf("upload reader ended early at block %d, remaining %d", blockID, remaining)
			}
			content := make([]byte, n)
			copy(content, buf[:n])
			remaining -= int64(n)
			isEnd := remaining == 0 || blockID+1 == count
			if err := sendOneUploadBlock(greq, in, out, blockID, content, isEnd, totalSize, meta, extendTaskDeadline); err != nil {
				return err
			}
			blockID++
			if isEnd {
				break
			}
		}
		return nil
	}, func() { closeIfCloser(reader) }, greq.Task.Close, in.Close)

	return greq.Task.ToProtobuf(), nil
}

func sendUploadBlocks(
	greq *GenericRequest,
	in *core.SpiteStreamWriter,
	out chan *implantpb.Spite,
	blocks [][]byte,
	totalSize int64,
	meta *implantpb.UploadRequest,
	onProgress func(),
) error {
	for i, content := range blocks {
		isEnd := i == len(blocks)-1
		if err := sendOneUploadBlock(greq, in, out, i, content, isEnd, totalSize, meta, onProgress); err != nil {
			return err
		}
	}
	return nil
}

func sendOneUploadBlock(
	greq *GenericRequest,
	in *core.SpiteStreamWriter,
	out chan *implantpb.Spite,
	blockID int,
	content []byte,
	isEnd bool,
	totalSize int64,
	meta *implantpb.UploadRequest,
	onProgress func(),
) error {
	msg := &implantpb.Block{
		BlockId: uint32(blockID),
		Content: content,
		End:     isEnd,
	}
	spite, err := types.BuildSpite(&implantpb.Spite{
		Timeout: uint64(consts.MinTimeout.Seconds()),
		TaskId:  greq.Task.Id,
	}, msg)
	if err != nil {
		return err
	}
	// Implant stream expects upload-named spites for block continuation.
	spite.Name = types.MsgUpload.String()
	if err := in.Send(spite); err != nil {
		return err
	}
	resp, ok := recvSpite(greq.Task.Ctx, out)
	if !ok {
		return ErrTaskContextCancelled
	}
	if err := types.AssertSpite(resp, types.MsgAck); err != nil {
		return buildTaskError(err)
	}
	greq.Session.AddMessage(resp, blockID+1)
	if err := greq.Session.TaskLog(greq.Task, resp); err != nil {
		return fmt.Errorf("write task log: %w", err)
	}
	if !resp.GetAck().Success {
		return fmt.Errorf("upload block %d not acked", blockID)
	}
	greq.Task.Done(resp, "")
	if onProgress != nil {
		onProgress()
	}
	if isEnd {
		saveUploadContext(greq, meta, totalSize)
		greq.Task.Finish(resp, "")
	}
	return nil
}

func mergeChunks(tempDir, finalPath string, totalChunks int) error {
	outputDir := filepath.Dir(finalPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	tempFile, err := os.CreateTemp(outputDir, ".download-*")
	if err != nil {
		return fmt.Errorf("failed to create temp output file: %w", err)
	}
	tempPath := tempFile.Name()
	defer func() {
		tempFile.Close()
		_ = os.Remove(tempPath)
	}()

	for i := 1; i <= totalChunks; i++ {
		chunkFile := filepath.Join(tempDir, fmt.Sprintf("%d.chunk", i))
		chunkData, err := os.ReadFile(chunkFile)
		if err != nil {
			return fmt.Errorf("failed to read chunk %d: %w", i, err)
		}

		if _, err := tempFile.Write(chunkData); err != nil {
			return fmt.Errorf("failed to write chunk %d to output: %w", i, err)
		}
	}

	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp output file: %w", err)
	}
	if err := os.Rename(tempPath, finalPath); err != nil {
		return fmt.Errorf("failed to finalize merged output: %w", err)
	}
	return nil
}

func finalizeDownload(greq *GenericRequest, req *implantpb.DownloadRequest, resp *implantpb.Spite, downloadAbs *implantpb.DownloadResponse, total int, finalPath, tempDir string) error {
	if err := mergeChunks(tempDir, finalPath, total); err != nil {
		return err
	}

	actualChecksum, err := fileutils.CalculateSHA256Checksum(finalPath)
	if err != nil {
		return fmt.Errorf("calculate final file checksum: %w", err)
	}
	if actualChecksum != downloadAbs.Checksum {
		_ = os.Remove(finalPath)
		return fmt.Errorf("final file checksum mismatch: expected %s, got %s", downloadAbs.Checksum, actualChecksum)
	}

	if restored, restoreErr := output.RestoreInvalidMinidumpSignature(finalPath); restoreErr != nil {
		logs.Log.Errorf("restore minidump signature %s: %s", finalPath, restoreErr)
	} else if restored {
		logs.Log.Infof("restored minidump signature: %s", finalPath)
		actualChecksum, err = fileutils.CalculateSHA256Checksum(finalPath)
		if err != nil {
			return fmt.Errorf("calculate restored dump checksum: %w", err)
		}
	}
	core.SaveParsedMinidumpCredentials(finalPath, greq.Task)

	downloadName := req.Name
	if req.Dir {
		downloadName += ".tar"
	}
	v := &output.DownloadContext{
		FileDescriptor: &output.FileDescriptor{
			Name:       downloadName,
			Checksum:   actualChecksum,
			TargetPath: req.Path,
			FilePath:   finalPath,
			Abstract:   fmt.Sprintf("download -%s -%s ", downloadName, req.Path),
			Size:       int64(downloadAbs.Size),
		},
	}

	ictx, err := rpcFileSaveContext(&clientpb.Context{
		Task:    greq.Task.ToProtobuf(),
		Session: greq.Session.ToProtobuf(),
		Type:    consts.ContextDownload,
		Value:   v.Marshal(),
	})
	if err != nil {
		logs.Log.Errorf("cannot create task %d , %s in db", greq.Task.Id, err.Error())
		greq.Task.Finish(resp, "download completed")
		return nil
	}

	core.PushContextEvent(consts.ContextDownload, ictx)
	greq.Task.Finish(resp, "sync id "+ictx.ID.String())
	return nil
}

// Download - Download a file from implant
func (rpc *Server) Download(ctx context.Context, req *implantpb.DownloadRequest) (*clientpb.Task, error) {
	if req == nil {
		return nil, types.ErrMissingRequestField
	}
	req.BufferSize = uint32(getPacketLength(ctx))
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	in, out, err := rpc.StreamGenericHandler(ctx, greq)
	if err != nil {
		logs.Log.Debugf("stream generate error: %s", err)
		return nil, err
	}
	runTaskHandler(greq.Task, func() error {
		resp, ok := recvSpite(greq.Task.Ctx, out)
		if !ok {
			return ErrTaskContextCancelled
		}
		err := types.AssertStatusAndSpite(resp, types.MsgDownload)
		if err != nil {
			return buildTaskError(err)
		}

		err = greq.Session.TaskLog(greq.Task, resp)
		if err != nil {
			return fmt.Errorf("write task log: %w", err)
		}
		total := downloadChunkCount(int(resp.GetDownloadResponse().Size), greq.Session.GetPacketLength())
		downloadAbs := resp.GetDownloadResponse()
		greq.Task.UpdateTotal(total)

		finalPath, err := fileutils.SafeJoin(configs.ContextPath, filepath.Join(greq.Session.ID, consts.DownloadPath, downloadAbs.Checksum))
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(finalPath), 0o700); err != nil {
			return err
		}
		if _, err := os.Stat(finalPath); err == nil {
			if actualChecksum, err := fileutils.CalculateSHA256Checksum(finalPath); err == nil && actualChecksum == downloadAbs.Checksum {
				greq.Task.Finish(resp, "file already exists and verified")
				return nil
			} else {
				os.Remove(finalPath)
			}
		}
		// mkdir for download chunk
		tempDir := filepath.Join(configs.TempPath, "downloads", resp.GetDownloadResponse().Checksum)
		var current_cur int32 = 1
		if _, err := os.Stat(tempDir); err == nil {
			if !isChunkSizeCompatible(tempDir, greq.Session.GetPacketLength(), total) {
				backupDir := tempDir + fmt.Sprintf(".bak.%d", time.Now().Unix())
				logs.Log.Warnf("[download] buffer size changed, backing up stale chunks to %s", backupDir)
				os.Rename(tempDir, backupDir)
				os.MkdirAll(tempDir, 0755)
			} else {
				greq.Task.Cur = int(current_cur) - 1
				greq.Task.Done(resp, "resuming download")
				var complete bool
				current_cur, complete, err = scanDownloadChunks(tempDir, total)
				if err != nil {
					return err
				}
				if complete {
					return finalizeDownload(greq, req, resp, downloadAbs, total, finalPath, tempDir)
				}
			}
		} else {
			err = os.MkdirAll(tempDir, 0755)
			if err != nil {
				return fmt.Errorf("create temp directory %s: %w", tempDir, err)
			}
		}

		//
		curRequest, _ := types.BuildSpite(&implantpb.Spite{
			Timeout: uint64(consts.MinTimeout.Seconds()),
			TaskId:  greq.Task.Id,
		}, &implantpb.DownloadRequest{
			Path:       req.Path,
			Name:       req.Name,
			Cur:        current_cur,
			Dir:        false,
			BufferSize: req.BufferSize,
		})
		if err := in.Send(curRequest); err != nil {
			return err
		}

		retries := 0
		for {
			var resp *implantpb.Spite
			var ok bool
			select {
			case resp, ok = <-out:
				retries = 0
			case <-greq.Task.Ctx.Done():
				return ErrTaskContextCancelled
			case <-time.After(2 * consts.MinTimeout):
				retries++
				if retries >= maxChunkRetries {
					return fmt.Errorf("chunk %d timed out after %d retries", current_cur, maxChunkRetries)
				}
				logs.Log.Debugf("[download] chunk %d timeout, retrying (%d/%d)", current_cur, retries, maxChunkRetries)
				if err := in.Send(curRequest); err != nil {
					return err
				}
				continue
			}

			if !ok {
				return ErrTaskContextCancelled
			}
			err := types.AssertStatusAndSpite(resp, types.MsgDownload)
			if err != nil {
				return buildTaskError(err)
			}

			downloadResp := resp.GetDownloadResponse()

			// Discard stale duplicate responses from retries
			if downloadResp.Cur != current_cur {
				chunkFile := filepath.Join(tempDir, fmt.Sprintf("%d.chunk", downloadResp.Cur))
				os.WriteFile(chunkFile, downloadResp.Content, 0644)
				logs.Log.Debugf("[download] discarding duplicate chunk %d (expected %d)", downloadResp.Cur, current_cur)
				continue
			}

			chunkFile := filepath.Join(tempDir, fmt.Sprintf("%d.chunk", downloadResp.Cur))
			err = os.WriteFile(chunkFile, downloadResp.Content, 0644)
			if err != nil {
				return fmt.Errorf("save chunk %d: %w", downloadResp.Cur, err)
			}
			if checksum, _ := fileutils.CalculateSHA256Checksum(chunkFile); checksum != downloadResp.Checksum {
				os.Remove(chunkFile)
				return fmt.Errorf("chunk %d checksum mismatch: expected %s, got %s", downloadResp.Cur, downloadResp.Checksum, checksum)
			}
			greq.Task.Done(resp, fmt.Sprintf("chunk %d/%d", downloadResp.Cur, total))
			if downloadResp.Cur == int32(total) {
				break
			}

			current_cur += 1
			curRequest, _ = types.BuildSpite(&implantpb.Spite{
				Timeout: uint64(consts.MinTimeout.Seconds()),
				TaskId:  greq.Task.Id,
			}, &implantpb.DownloadRequest{
				Path:       req.Path,
				Name:       req.Name,
				Cur:        current_cur,
				Dir:        false,
				BufferSize: req.BufferSize,
			})
			if err := in.Send(curRequest); err != nil {
				return err
			}
		}

		return finalizeDownload(greq, req, resp, downloadAbs, total, finalPath, tempDir)
	}, greq.Task.Close, in.Close)

	return greq.Task.ToProtobuf(), nil
}
