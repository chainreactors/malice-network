package rpc

import (
	"context"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/packet"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/commonpb"
	"github.com/chainreactors/malice-network/proto/implant/pluginpb"
	"github.com/gookit/config/v2"
)

// Upload - Upload a file from the remote file system
func (rpc *Server) Upload(ctx context.Context, req *pluginpb.UploadRequest) (*clientpb.Task, error) {
	count := packet.Count(req.Data, config.Int(consts.MaxPacketLength))
	if count == 1 {
		greq := newGenericRequest(req)
		resp, err := rpc.genericHandler(ctx, greq)
		if err != nil {
			return nil, err
		}
		return resp.(*clientpb.Task), nil
	} else {
		greq := newGenericRequest(&pluginpb.UploadRequest{
			Name:   req.Name,
			Target: req.Target,
			Priv:   req.Priv,
			Hidden: req.Hidden,
		})
		in, out, err := rpc.streamGenericHandler(ctx, greq)
		if err != nil {
			return nil, err
		}
		var blockId = 0
		go func() {
			for block := range packet.Chunked(req.Data, count) {
				msg := &commonpb.Block{
					BlockId: uint32(blockId),
					Content: block,
				}
				spite := &commonpb.Spite{
					Timeout: uint64(consts.MinTimeout.Seconds()),
					TaskId:  greq.Task.Id,
				}
				spite, _ = types.BuildSpite(spite, msg)
				in <- spite
				resp := <-out
				if !resp.GetAsyncAck().Success {
					// todo error parser
					return
				}
			}
			close(in)
		}()
		return greq.Task.ToProtobuf(), nil
	}
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
	resp, err := rpc.genericHandler(ctx, newGenericRequest(req))
	if err != nil {
		return nil, err
	}
	return resp.(*clientpb.SyncResp), nil
}
