package rpc

import (
	"context"
	"github.com/chainreactors/files"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/encoders/hash"
	"github.com/chainreactors/malice-network/helper/packet"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/commonpb"
	"github.com/chainreactors/malice-network/proto/implant/pluginpb"
	"github.com/chainreactors/malice-network/server/configs"
	"github.com/chainreactors/malice-network/server/core"
	"io/fs"
	"os"
	"path"
)

// Upload - Upload a file from the remote file system
func (rpc *Server) Upload(ctx context.Context, req *pluginpb.UploadRequest) (*clientpb.Task, error) {
	filename := path.Join(configs.GetTempDir(), hash.Md5Hash(req.Data))
	if files.IsExist(filename) {

	} else {
		err := os.WriteFile(filename, req.Data, fs.FileMode(req.Priv))
		if err != nil {
			return nil, err
		}
	}

	greq := newGenericRequest(req)
	greq.Task = core.NewTask("upload", packet.Count(req.Data, configs.GetConfig(consts.MaxPacketLength).(int)))
	ch, err := rpc.asyncGenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	var blockId = 0
	go func() {
		for block := range packet.Chunked(req.Data, configs.GetConfig(consts.MaxPacketLength).(int)) {
			msg := &commonpb.Block{
				BlockId: uint32(blockId),
				Content: block,
			}
			spite := &commonpb.Spite{
				Timeout: uint64(consts.MinTimeout.Seconds()),
				TaskId:  greq.Task.Id,
			}
			spite, _ = types.BuildSpite(spite, msg)
			ch <- spite
		}
	}()

	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) Download(ctx context.Context, req *pluginpb.DownloadRequest) (*clientpb.Task, error) {
	resp, err := rpc.GenericHandler(ctx, newGenericRequest(req))
	if err != nil {
		return nil, err
	}
	return resp.(*clientpb.Task), nil
}

func (rpc *Server) Sync(ctx context.Context, req *clientpb.Sync) (*clientpb.SyncResp, error) {
	resp, err := rpc.GenericHandler(ctx, newGenericRequest(req))
	if err != nil {
		return nil, err
	}
	return resp.(*clientpb.SyncResp), nil
}
