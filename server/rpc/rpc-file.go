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
		in, out, err := rpc.StreamGenericHandler(ctx, greq)
		if err != nil {
			return nil, err
		}
		var blockId = 0
		core.SafeGoWithTask(greq.Task, func() {
			stat := <-out
			err := types.HandleMaleficError(stat)
			if err != nil {
				greq.Task.Panic(buildErrorEvent(greq.Task, err))
				return
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
				in <- spite
				resp := <-out
				err = types.AssertSpite(resp, types.MsgAck)
				if err != nil {
					return
				}
				greq.Session.AddMessage(resp, blockId)

				err = greq.Session.TaskLog(greq.Task, resp)
				if err != nil {
					logs.Log.Errorf("Failed to write task log: %v", err)
					return
				}
				if resp.GetAck().Success {
					greq.Task.Done(resp, "")
					if err != nil {
						logs.Log.Errorf("cannot update task %d , %s in db", greq.Task.Id, err.Error())
						return
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
		}, greq.Task.Close, func() { close(in) })
		return greq.Task.ToProtobuf(), nil
	}
}

func mergeChunks(tempDir, finalPath string, totalChunks int) error {
	outputDir := filepath.Dir(finalPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	outputFile, err := os.Create(finalPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()

	for i := 1; i <= totalChunks; i++ {
		chunkFile := filepath.Join(tempDir, fmt.Sprintf("%d.chunk", i))
		chunkData, err := os.ReadFile(chunkFile)
		if err != nil {
			return fmt.Errorf("failed to read chunk %d: %w", i, err)
		}

		if _, err := outputFile.Write(chunkData); err != nil {
			return fmt.Errorf("failed to write chunk %d to output: %w", i, err)
		}
	}

	return nil
}

// Download - Download a file from implant
func (rpc *Server) Download(ctx context.Context, req *implantpb.DownloadRequest) (*clientpb.Task, error) {
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
	core.SafeGoWithTask(greq.Task, func() {
		resp := <-out
		err := types.AssertStatusAndSpite(resp, types.MsgDownload)
		if err != nil {
			greq.Task.Panic(buildErrorEvent(greq.Task, err))
			return
		}

		err = greq.Session.TaskLog(greq.Task, resp)
		if err != nil {
			logs.Log.Errorf("Failed to write task log: %v", err)
			return
		}
		total := int(resp.GetDownloadResponse().Size)/config.Int(consts.ConfigMaxPacketLength) + 1
		downloadAbs := resp.GetDownloadResponse()
		greq.Task.Total = total

		finalPath, err := fileutils.SafeJoin(configs.ContextPath, filepath.Join(greq.Session.ID, consts.DownloadPath, downloadAbs.Checksum))
		if err != nil {
			greq.Task.Panic(buildErrorEvent(greq.Task, err))
			return
		}
		if err := os.MkdirAll(filepath.Dir(finalPath), 0o700); err != nil {
			greq.Task.Panic(buildErrorEvent(greq.Task, err))
			return
		}
		if _, err := os.Stat(finalPath); err == nil {
			if actualChecksum, err := fileutils.CalculateSHA256Checksum(finalPath); err == nil && actualChecksum == downloadAbs.Checksum {
				greq.Task.Finish(resp, "file already exists and verified")
				return
			} else {
				os.Remove(finalPath)
			}
		}
		// mkdir for download chunk
		tempDir := filepath.Join(configs.TempPath, "downloads", resp.GetDownloadResponse().Checksum)
		var current_cur int32 = 1
		if _, err := os.Stat(tempDir); err == nil {
			greq.Task.Finish(resp, "file already exists")
			for i := 1; i <= total; i++ {
				chunkFile := filepath.Join(tempDir, fmt.Sprintf("%d.chunk", i))
				if _, err := os.Stat(chunkFile); err != nil {
					current_cur = int32(i)
					break
				}
				if i == total {
					// merge
					if err := mergeChunks(tempDir, finalPath, total); err != nil {
						greq.Task.Panic(buildErrorEvent(greq.Task, err))
						return
					}
					return
				}
			}

		} else {
			err = os.MkdirAll(tempDir, 0755)
			if err != nil {
				logs.Log.Errorf("cannot create temp directory %s, %s", tempDir, err.Error())
				return
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
		in <- curRequest

		for resp := range out {
			err := types.AssertStatusAndSpite(resp, types.MsgDownload)
			if err != nil {
				logs.Log.Errorf(err.Error())
				return
			}

			downloadResp := resp.GetDownloadResponse()
			chunkFile := filepath.Join(tempDir, fmt.Sprintf("%d.chunk", downloadResp.Cur))
			err = os.WriteFile(chunkFile, downloadResp.Content, 0644)
			if err != nil {
				logs.Log.Errorf("failed to save chunk %d: %v", downloadResp.Cur, err)
				return
			}
			if checksum, _ := fileutils.CalculateSHA256Checksum(chunkFile); checksum != downloadResp.Checksum {
				os.Remove(chunkFile)
				greq.Task.Panic(buildErrorEvent(greq.Task, fmt.Errorf("chunk %d checksum mismatch: expected %s, got %s", downloadResp.Cur, downloadResp.Checksum, checksum)))
				return
			}
			greq.Task.Done(resp, fmt.Sprintf("chunk %d/%d", downloadResp.Cur, total))
			if downloadResp.Cur == int32(total) {
				// merge
				if err := mergeChunks(tempDir, finalPath, total); err != nil {
					greq.Task.Panic(buildErrorEvent(greq.Task, err))
					return
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
			in <- curRequest
		}

		actualChecksum, err := fileutils.CalculateSHA256Checksum(finalPath)
		if err != nil {
			greq.Task.Panic(buildErrorEvent(greq.Task, fmt.Errorf("failed to calculate final file checksum: %w", err)))
			return
		}

		if actualChecksum != downloadAbs.Checksum {
			os.Remove(finalPath)
			greq.Task.Panic(buildErrorEvent(greq.Task, fmt.Errorf("final file checksum mismatch: expected %s, got %s", downloadAbs.Checksum, actualChecksum)))
			return
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
	}, greq.Task.Close, func() { close(in) })

	return greq.Task.ToProtobuf(), nil
}
