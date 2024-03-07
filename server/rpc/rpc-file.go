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
	dbSession := db.Session()
	if count == 1 {
		greq, err := newGenericRequest(ctx, req)
		if err != nil {
			return nil, err
		}
		ch, err := rpc.asyncGenericHandler(ctx, greq)
		if err != nil {
			return nil, err
		}
		td := &models.TaskDescription{
			Name:    req.Name,
			Path:    req.Target,
			Command: fmt.Sprintf("upload -%d -%t", req.Priv, req.Hidden),
			Size:    int64(len(req.Data)),
		}
		taskModel := models.ConvertToTaskDB(greq.Task, "upload", td)
		err = dbSession.Create(taskModel).Error
		if err != nil {
			logs.Log.Errorf("cannot create task %d, %s in db", greq.Task.Id, err.Error())
			return nil, err
		}
		go func() {
			resp := <-ch

			err := AssertStatusAndResponse(resp, types.MsgBlock)
			if err != nil {
				core.EventBroker.Publish(buildErrorEvent(greq.Task, err))
				return
			}
			greq.SetCallback(func() {
				greq.Task.Spite = resp
				core.EventBroker.Publish(core.Event{
					EventType: consts.EventTaskCallback,
					Task:      greq.Task,
				})
			})
		}()
		greq.Task.Done()
		err = taskModel.UpdateCur(dbSession, greq.Task.Cur)
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
		td := &models.TaskDescription{
			Name:    req.Name,
			Path:    req.Target,
			Command: fmt.Sprintf("upload -%d -%t", req.Priv, req.Hidden),
			Size:    int64(len(req.Data)),
		}
		taskModel := models.ConvertToTaskDB(greq.Task, "upload", td)
		err = dbSession.Create(taskModel).Error
		if err != nil {
			logs.Log.Errorf("cannot create task %d , %s in db", greq.Task.Id, err.Error())
			return nil, err
		}
		var blockId = 0
		go func() {
			stat := <-out
			err := AssertResponse(stat, types.MsgNil)
			if err != nil {
				core.EventBroker.Publish(buildErrorEvent(greq.Task, err))
				return
			}
			for block := range packet.Chunked(req.Data, count) {
				msg := &implantpb.Block{
					BlockId: uint32(blockId),
					Content: block,
				}
				spite := &implantpb.Spite{
					Timeout: uint64(consts.MinTimeout.Seconds()),
					TaskId:  greq.Task.Id,
				}
				spite, _ = types.BuildSpite(spite, msg)
				in <- spite
				resp := <-out
				if resp.GetAsyncAck().Success {
					greq.Task.Done()
					core.EventBroker.Publish(core.Event{
						EventType: consts.EventTaskDone,
						Task:      greq.Task,
						Err:       stat.Status.Error,
					})
					err = taskModel.UpdateCur(dbSession, blockId)
					if err != nil {
						logs.Log.Errorf("cannot update task %d , %s in db", greq.Task.Id, err.Error())
					}
					return
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
	dbSession := db.Session()
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
			core.EventBroker.Publish(buildErrorEvent(greq.Task, err))
			return
		}
		fileName := path.Join(configs.TempPath, stat.GetDownloadResponse().Checksum)
		greq.Task.Total = int(stat.GetDownloadResponse().Size) / config.Int(consts.MaxPacketLength)
		td := &models.TaskDescription{
			Name:    req.Name,
			Path:    req.Path,
			Command: fmt.Sprintf("download -%s -%s", req.Name, req.Path),
			Size:    int64(stat.GetDownloadResponse().Size),
		}
		taskModel := models.ConvertToTaskDB(greq.Task, "download", td)
		err = dbSession.Create(taskModel).Error
		if err != nil {
			logs.Log.Errorf("cannot create task %d , %s in db", greq.Task.Id, err.Error())
		}
		if files.IsExist(fileName) {
			if err != nil {
				logs.Log.Errorf("db store download error: %s", err)
			}
			err := taskModel.UpdateCur(dbSession, greq.Task.Total)
			if err != nil {
				logs.Log.Errorf("cannot update task %d , %s in db", greq.Task.Id, err.Error())
			}
			return
		} else {
			downloadFile, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				return
			}
			defer downloadFile.Close()
			go func() {
				for resp := range out {
					block := resp.GetBlock()
					_, fileErr := downloadFile.Write(block.Content)
					if fileErr != nil {
						return
					}
					ack, _ := greq.NewSpite(&implantpb.AsyncACK{Success: true})
					in <- ack
					err := taskModel.UpdateCur(dbSession, int(block.BlockId))
					if err != nil {
						logs.Log.Errorf("cannot update task %d , %s in db", greq.Task.Id, err.Error())
						return
					}
				}
			}()
		}
	}()
	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) Sync(ctx context.Context, req *clientpb.Sync) (*clientpb.SyncResp, error) {
	dbSession := db.Session()
	td, err := models.GetTaskDescriptionByID(dbSession, req.FileId)
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
