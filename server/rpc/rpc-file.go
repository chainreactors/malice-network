package rpc

import (
	"context"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/helper/utils/handler"
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
		go greq.HandlerResponse(ch, types.MsgAck, func(spite *implantpb.Spite) {
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
		go func() {
			stat := <-out
			err := handler.HandleMaleficError(stat)
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
				err = handler.AssertSpite(resp, types.MsgAck)
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
			close(in)
		}()
		return greq.Task.ToProtobuf(), nil
	}
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

	go func() {
		resp := <-out
		err := handler.AssertStatusAndSpite(resp, types.MsgDownload)
		if err != nil {
			greq.Task.Panic(buildErrorEvent(greq.Task, err))
			return
		}

		err = greq.Session.TaskLog(greq.Task, resp)
		if err != nil {
			logs.Log.Errorf("Failed to write task log: %v", err)
			return
		}
		greq.Task.Total = int(resp.GetDownloadResponse().Size)/config.Int(consts.ConfigMaxPacketLength) + 1

		// temp file
		downloadAbs := resp.GetDownloadResponse()
		fileName := filepath.Join(configs.TempPath, downloadAbs.Checksum)
		moveName := filepath.Join(configs.ContextPath, greq.Session.ID, consts.DownloadPath, downloadAbs.Checksum)
		downloadFile, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			logs.Log.Errorf("cannot create file %s, %s", fileName, err.Error())
			return
		}
		defer downloadFile.Close()

		// return ack
		ack, _ := types.BuildSpite(&implantpb.Spite{
			Timeout: uint64(consts.MinTimeout.Seconds()),
			TaskId:  greq.Task.Id,
		}, &implantpb.ACK{
			Success: true,
			End:     false,
		})
		ack.Name = types.MsgDownload.String()
		in <- ack

		for resp := range out {
			err := handler.AssertStatusAndSpite(resp, types.MsgBlock)
			if err != nil {
				logs.Log.Errorf(err.Error())
				return
			}
			block := resp.GetBlock()
			_, err = downloadFile.Write(block.Content)
			if err != nil {
				logs.Log.Errorf(err.Error())
				return
			}
			if err != nil {
				logs.Log.Errorf("cannot update task %d , %s in db", greq.Task.Id, err.Error())
				return
			}
			greq.Task.Done(ack, "")
			if !block.End {
				ack, _ := greq.NewSpite(&implantpb.ACK{Success: true})
				ack.TaskId = greq.Task.Id
				ack.Name = types.MsgDownload.String()
				in <- ack
				//greq.Session.Add(resp, int(block.BlockId+1))
			} else {
				checksum, _ := fileutils.CalculateSHA256Checksum(fileName)
				if checksum != downloadAbs.Checksum {
					greq.Task.Panic(buildErrorEvent(greq.Task, fmt.Errorf("checksum error")))
					return
				}
				v := &output.DownloadContext{
					FileDescriptor: &output.FileDescriptor{
						Name:       req.Name,
						Checksum:   downloadAbs.Checksum,
						TargetPath: req.Path,
						FilePath:   moveName,
						Abstract:   fmt.Sprintf("download -%s -%s ", req.Name, req.Path),
						Size:       int64(resp.GetDownloadResponse().Size),
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
				err = fileutils.MoveFile(fileName, moveName)
				if err != nil {
					return
				}
				core.PushContextEvent(consts.ContextUpload, ictx)
				greq.Task.Finish(resp, "sync id "+checksum)
			}
		}
	}()
	return greq.Task.ToProtobuf(), nil
}
