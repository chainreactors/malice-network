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
	"github.com/gookit/config/v2"
	"os"
	"path/filepath"
)

// Upload - Upload a file from the remote file system
func (rpc *Server) Upload(ctx context.Context, req *implantpb.UploadRequest) (*clientpb.Task, error) {
	if req == nil {
		return nil, types.ErrMissingRequestField
	}
	count := parser.Count(req.Data, config.Int(consts.ConfigMaxPacketLength))
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
			for block := range parser.Chunked(req.Data, config.Int(consts.ConfigMaxPacketLength)) {
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

// Download - Download a file from implant
func (rpc *Server) Download(ctx context.Context, req *implantpb.DownloadRequest) (*clientpb.Task, error) {
	if req == nil {
		return nil, types.ErrMissingRequestField
	}
	req.BufferSize = uint32(config.Uint(consts.ConfigMaxPacketLength))
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
		total := int(resp.GetDownloadResponse().Size)/config.Int(consts.ConfigMaxPacketLength) + 1
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
			for i := 1; i <= total; i++ {
				chunkFile := filepath.Join(tempDir, fmt.Sprintf("%d.chunk", i))
				if _, err := os.Stat(chunkFile); err != nil {
					current_cur = int32(i)
					break
				}
				if i == total {
					// merge
					if err := mergeChunks(tempDir, finalPath, total); err != nil {
						return err
					}
					return nil
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
				// merge
				if err := mergeChunks(tempDir, finalPath, total); err != nil {
					return err
				}
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

		actualChecksum, err := fileutils.CalculateSHA256Checksum(finalPath)
		if err != nil {
			return fmt.Errorf("calculate final file checksum: %w", err)
		}

		if actualChecksum != downloadAbs.Checksum {
			os.Remove(finalPath)
			return fmt.Errorf("final file checksum mismatch: expected %s, got %s", downloadAbs.Checksum, actualChecksum)
		}
		if req.Dir == true {
			req.Name = req.Name + ".tar"
		}
		v := &output.DownloadContext{
			FileDescriptor: &output.FileDescriptor{
				Name:       req.Name,
				Checksum:   actualChecksum,
				TargetPath: req.Path,
				FilePath:   finalPath,
				Abstract:   fmt.Sprintf("download -%s -%s ", req.Name, req.Path),
				Size:       int64(downloadAbs.Size),
			},
		}

		ictx, err := db.SaveContext(&clientpb.Context{
			Task:    greq.Task.ToProtobuf(),
			Session: greq.Session.ToProtobuf(),
			Type:    consts.ContextDownload,
			Value:   v.Marshal(),
		})
		if err != nil {
			logs.Log.Errorf("cannot create task %d , %s in db", greq.Task.Id, err.Error())
		}

		core.PushContextEvent(consts.ContextDownload, ictx)
		greq.Task.Finish(resp, "sync id "+ictx.ID.String())
		return nil
	}, greq.Task.Close, in.Close)

	return greq.Task.ToProtobuf(), nil
}
