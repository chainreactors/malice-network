package rpc

import (
	"context"
	"fmt"
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
	"github.com/chainreactors/malice-network/server/internal/parser"
	"os"
	"path/filepath"
)

var rpcFileSaveContext = db.SaveContext

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

// Upload - Upload a file from the remote file system
func (rpc *Server) Upload(ctx context.Context, req *implantpb.UploadRequest) (*clientpb.Task, error) {
	if req == nil {
		return nil, types.ErrMissingRequestField
	}
	count := parser.Count(req.Data, getPacketLength(ctx))
	if count == 1 {
		greq, err := newGenericRequest(ctx, req)
		if err != nil {
			return nil, err
		}
		ch, err := rpc.GenericHandler(ctx, greq)
		if err != nil {
			return nil, err
		}
		greq.HandlerResponse(ch, types.MsgAck, func(spite *implantpb.Spite) {
			v := &output.UploadContext{
				FileDescriptor: &output.FileDescriptor{
					Name:       req.Name,
					TargetPath: req.Target,
					Abstract:   fmt.Sprintf("upload -%d -%t", req.Priv, req.Hidden),
					Size:       int64(len(req.Data)),
				},
			}
			ictx, err := db.SaveContext(&clientpb.Context{
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
		})
		if err != nil {
			return nil, err
		}
		return greq.Task.ToProtobuf(), nil
	} else {
		greq, err := newGenericRequest(ctx, &implantpb.UploadRequest{
			Name:   req.Name,
			Target: req.Target,
			Priv:   req.Priv,
			Hidden: req.Hidden,
		}, count)
		if err != nil {
			return nil, err
		}
		in, out, err := rpc.StreamGenericHandler(ctx, greq)
		if err != nil {
			return nil, err
		}
		var blockId = 0
		runTaskHandler(greq.Task, func() error {
			stat, ok := recvSpite(greq.Task.Ctx, out)
			if !ok {
				return ErrTaskContextCancelled
			}
			err := types.HandleMaleficError(stat)
			if err != nil {
				return buildTaskError(err)
			}
			for block := range parser.Chunked(req.Data, greq.Session.GetPacketLength()) {
				msg := &implantpb.Block{
					BlockId: uint32(blockId),
					Content: block,
				}
				blockId++
				if blockId == count {
					msg.End = true
				}
				spite, _ := types.BuildSpite(&implantpb.Spite{
					Timeout: uint64(consts.MinTimeout.Seconds()),
					TaskId:  greq.Task.Id,
				}, msg)
				spite.Name = types.MsgUpload.String()
				if err := in.Send(spite); err != nil {
					return err
				}
				resp, ok := recvSpite(greq.Task.Ctx, out)
				if !ok {
					return ErrTaskContextCancelled
				}
				err = types.AssertSpite(resp, types.MsgAck)
				if err != nil {
					return buildTaskError(err)
				}
				greq.Session.AddMessage(resp, blockId)

				err = greq.Session.TaskLog(greq.Task, resp)
				if err != nil {
					return fmt.Errorf("write task log: %w", err)
				}
				if resp.GetAck().Success {
					greq.Task.Done(resp, "")
					if err != nil {
						logs.Log.Errorf("cannot update task %d , %s in db", greq.Task.Id, err.Error())
						return nil
					}
					if msg.End {
						v := &output.UploadContext{
							FileDescriptor: &output.FileDescriptor{
								Name:       req.Name,
								TargetPath: req.Target,
								Abstract:   fmt.Sprintf("upload -%d -%t", req.Priv, req.Hidden),
								Size:       int64(len(req.Data)),
							},
						}
						ictx, err := db.SaveContext(&clientpb.Context{
							Task:    greq.Task.ToProtobuf(),
							Session: greq.Session.ToProtobuf(),
							Type:    consts.ContextUpload,
							Value:   v.Marshal(),
						})
						if err != nil {
							logs.Log.Errorf("cannot create task %d , %s in db", greq.Task.Id, err.Error())
						}
						greq.Task.Finish(resp, "")
						core.PushContextEvent(consts.ContextUpload, ictx)
					}
				}
			}
			return nil
		}, greq.Task.Close, in.Close)
		return greq.Task.ToProtobuf(), nil
	}
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
		greq.Task.Total = total

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
			greq.Task.Done(resp, "resuming download")
			var complete bool
			current_cur, complete, err = scanDownloadChunks(tempDir, total)
			if err != nil {
				return err
			}
			if complete {
				return finalizeDownload(greq, req, resp, downloadAbs, total, finalPath, tempDir)
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

		for {
			resp, ok := recvSpite(greq.Task.Ctx, out)
			if !ok {
				return ErrTaskContextCancelled
			}
			err := types.AssertStatusAndSpite(resp, types.MsgDownload)
			if err != nil {
				return buildTaskError(err)
			}

			downloadResp := resp.GetDownloadResponse()
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
