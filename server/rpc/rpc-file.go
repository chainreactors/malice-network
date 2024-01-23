package rpc

import (
	"context"
	"github.com/chainreactors/files"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/packet"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/commonpb"
	"github.com/chainreactors/malice-network/proto/implant/pluginpb"
	"github.com/chainreactors/malice-network/server/core"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/gookit/config/v2"
	"os"
	"path"
)

// Upload - Upload a file from the remote file system
func (rpc *Server) Upload(ctx context.Context, req *pluginpb.UploadRequest) (*clientpb.Task, error) {
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
		go func() {
			resp := <-ch
			err := AssertAsyncResponse(resp, types.MsgBlock)
			if err != nil {
				return
			}
			greq.SetCallback(func() {
				greq.Task.Spite = resp
				core.EventBroker.Publish(core.Event{
					EventType: consts.EventTaskCallback,
					Task:      greq.Task,
				})
			})
			greq.Task.Done()
		}()
		return greq.Task.ToProtobuf(), nil
	} else {
		greq, err := newGenericRequest(ctx, &pluginpb.UploadRequest{
			Name:   req.Name,
			Target: req.Target,
			Priv:   req.Priv,
			Hidden: req.Hidden,
		}, count)
		in, out, err := rpc.streamGenericHandler(ctx, greq)
		if err != nil {
			return nil, err
		}
		var blockId = 0
		go func() {
			stat := <-out
			err := AssertAsyncResponse(stat, types.MsgNil)
			if err != nil {
				return
			}
			for block := range packet.Chunked(req.Data, count) {
				msg := &commonpb.Block{
					BlockId: uint32(blockId),
					Content: block,
				}
				spite := &commonpb.Spite{
					Timeout: uint64(consts.MinTimeout.Seconds()),
					TaskId:  greq.Task.Id,
				}
				_, isEND := <-packet.Chunked(req.Data, count)
				if !isEND {
					spite.End = true
				}
				spite, _ = types.BuildSpite(spite, msg)
				in <- spite
				resp := <-out
				if !resp.GetAsyncAck().Success {
					greq.Task.Done()
					return
				}
			}
			close(in)
		}()
		return greq.Task.ToProtobuf(), nil
	}
}

// Download - Download a file from implant
func (rpc *Server) Download(ctx context.Context, req *pluginpb.DownloadRequest) (*clientpb.Task, error) {
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
		if stat.Status != 0 {
			greq.Task.Done()
			return
		}
		fileName := path.Join(configs.TempPath, stat.GetDownloadResponse().Checksum)
		greq.Task.Total = int(stat.GetDownloadResponse().Size) / config.Int(consts.MaxPacketLength)
		if files.IsExist(fileName) {
			// TODO - DB SELECT TASK
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
					ack, _ := greq.NewSpite(&commonpb.AsyncACK{Success: true})
					in <- ack
				}
			}()
		}
	}()
	return greq.Task.ToProtobuf(), nil
}

//func (rpc *Server) Download(ctx context.Context, req *pluginpb.DownloadRequest) (*clientpb.Task, error) {
//	filename := path.Join(configs.TempPath, hash.Md5Hash(req.))
//	if files.IsExist(filename) {
//
//	} else {
//		err := os.WriteFile(filename, req.Data, fs.FileMode(req.Priv))
//		if err != nil {
//			return nil, err
//		}
//	}
//
//	greq := newGenericRequest(&pluginpb.DownloadRequest{
//		Name: req.Name,
//		Path: req.Path,
//	})
//	in, out, err := rpc.streamGenericHandler(ctx, greq)
//	if err != nil {
//		return nil, err
//	}
//	go func() {
//		for resp := range out {
//			resp.GetBlock()
//		}
//	}()
//	return resp.(*clientpb.Task), nil
//}

func (rpc *Server) Sync(ctx context.Context, req *clientpb.Sync) (*clientpb.SyncResp, error) {
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		logs.Log.Errorf(err.Error())
		return nil, err
	}
	sid, err := getSessionID(ctx)
	if err != nil {
		logs.Log.Errorf(err.Error())
		return nil, err
	}
	session, ok := core.Sessions.Get(sid)
	if !ok {
		return nil, ErrInvalidSessionID
	}
	session.Tasks.Add(greq.Task)

	if !files.IsExist(req.Target) {
		return nil, os.ErrExist
	}
	data, err := os.ReadFile(req.Target)
	if err != nil {
		return nil, err
	}
	resp := &clientpb.SyncResp{
		Task:    greq.Task.ToProtobuf(),
		Target:  req.Target,
		Content: data,
	}
	return resp, nil
}
