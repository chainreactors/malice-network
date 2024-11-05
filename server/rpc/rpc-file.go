package rpc

import (
	"context"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/helper/utils/file"
	"github.com/chainreactors/malice-network/helper/utils/handler"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"github.com/chainreactors/malice-network/server/internal/parser"
	"github.com/gookit/config/v2"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm/utils"
	"os"
	"path"
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
		taskPb := greq.Task.ToProtobuf()
		err = db.AddFile("upload", taskPb, &models.FileDescription{
			Name:    req.Name,
			Path:    req.Target,
			Command: fmt.Sprintf("upload -%d -%t", req.Priv, req.Hidden),
			Size:    int64(len(req.Data)),
		})
		if err != nil {
			logs.Log.Errorf("cannot create task %d, %s in db", greq.Task.Id, err.Error())
			return nil, err
		}
		go greq.HandlerResponse(ch, types.MsgBlock)
		taskID := greq.Task.SessionId + "-" + utils.ToString(greq.Task.Id)
		err = db.UpdateFileByID(taskID, greq.Task.Cur+1)
		if err != nil {
			logs.Log.Errorf("cannot update task %d , %s in db", greq.Task.Id, err.Error())
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
		taskPb := greq.Task.ToProtobuf()
		err = db.AddFile("upload", taskPb, &models.FileDescription{
			Name:     req.Name,
			NickName: "",
			Path:     req.Target,
			Command:  fmt.Sprintf("upload -%d -%t", req.Priv, req.Hidden),
			Size:     int64(len(req.Data)),
		})
		if err != nil {
			logs.Log.Errorf("cannot create task %d , %s in db", greq.Task.Id, err.Error())
		}
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
				err = handler.AssertResponse(resp, types.MsgAck)
				if err != nil {
					return
				}
				greq.Session.AddMessage(resp, blockId)
				respByte, err := proto.Marshal(resp)
				if err != nil {
					logs.Log.Errorf("Failed to marshal resp to byte: %v", err)
					return
				}
				err = greq.Session.TaskLog(greq.Task, respByte)
				if err != nil {
					logs.Log.Errorf("Failed to write task log: %v", err)
					return
				}
				if resp.GetAck().Success {
					greq.Task.Done(resp, "")
					taskID := greq.Task.SessionId + "-" + utils.ToString(greq.Task.Id)
					err = db.UpdateFileByID(taskID, blockId)
					if err != nil {
						logs.Log.Errorf("cannot update task %d , %s in db", greq.Task.Id, err.Error())
						return
					}
					if msg.End {
						greq.Task.Finish(resp, "")
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
		err := handler.AssertStatusAndResponse(resp, types.MsgDownload)
		if err != nil {
			greq.Task.Panic(buildErrorEvent(greq.Task, err))
			return
		}
		respCheckSum := resp.GetDownloadResponse().Checksum
		fileName := path.Join(configs.TempPath, respCheckSum)
		greq.Session.AddMessage(resp, 0)
		greq.Task.Total = int(resp.GetDownloadResponse().Size)/config.Int(consts.ConfigMaxPacketLength) + 1
		td := &models.FileDescription{
			Name:     req.Name,
			NickName: respCheckSum,
			Path:     req.Path,
			Command:  fmt.Sprintf("download -%s -%s ", req.Name, req.Path),
			Size:     int64(resp.GetDownloadResponse().Size),
		}
		taskPb := greq.Task.ToProtobuf()
		err = db.AddFile("download", taskPb, td)
		if err != nil {
			logs.Log.Errorf("cannot create task %d , %s in db", greq.Task.Id, err.Error())
		}
		err = db.UpdateDownloadTotal(greq.Task.ToProtobuf(), greq.Task.Total)
		if err != nil {
			logs.Log.Errorf("cannot update task %d , %s in db", greq.Task.Id, err.Error())
		}
		downloadFile, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			logs.Log.Errorf("cannot create file %s, %s", fileName, err.Error())
			return
		}
		defer downloadFile.Close()
		ack, _ := types.BuildSpite(&implantpb.Spite{
			Timeout: uint64(consts.MinTimeout.Seconds()),
			TaskId:  greq.Task.Id,
		}, &implantpb.ACK{
			Success: true,
			End:     false,
		})
		respByte, err := proto.Marshal(ack)
		if err != nil {
			logs.Log.Errorf("Failed to marshal resp to byte: %v", err)
			return
		}
		err = greq.Session.TaskLog(greq.Task, respByte)
		if err != nil {
			logs.Log.Errorf("Failed to write task log: %v", err)
			return
		}
		ack.Name = types.MsgDownload.String()
		in <- ack
		for resp := range out {
			err := handler.AssertStatusAndResponse(resp, types.MsgBlock)
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
			ack, _ := greq.NewSpite(&implantpb.ACK{Success: true})
			ack.TaskId = greq.Task.Id
			in <- ack
			ack.Name = types.MsgDownload.String()
			greq.Session.AddMessage(resp, int(block.BlockId+1))
			taskID := greq.Task.SessionId + "-" + utils.ToString(greq.Task.Id)
			err = db.UpdateFileByID(taskID, int(block.BlockId+1))
			if err != nil {
				logs.Log.Errorf("cannot update task %d , %s in db", greq.Task.Id, err.Error())
				return
			}
			greq.Task.Done(ack, "")
			if block.End {
				checksum, _ := file.CalculateSHA256Checksum(fileName)
				if checksum != respCheckSum {
					greq.Task.Panic(buildErrorEvent(greq.Task, fmt.Errorf("checksum error")))
					return
				}
				greq.Task.Finish(resp, "sync id "+checksum)
			}
		}
	}()
	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) Sync(ctx context.Context, req *clientpb.Sync) (*clientpb.SyncResp, error) {
	td, err := db.GetTaskDescriptionByID(req.FileId)
	if err != nil {
		logs.Log.Errorf("cannot find task in db by fileid: %s", err)
		return nil, err
	}
	//if !file.Exist(td.Path + td.Name) {
	//	return nil, os.ErrExist
	//}
	data, err := os.ReadFile(path.Join(configs.TempPath, td.NickName))
	if err != nil {
		return nil, err
	}
	resp := &clientpb.SyncResp{
		Name:    td.Name,
		Content: data,
	}
	return resp, nil
}
