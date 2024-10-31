package rpc

import (
	"context"
	"errors"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/build"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/docker/docker/client"
	"os"
	"path/filepath"
	"sync"
)

var dockerClient *client.Client
var once sync.Once
var maleficConfig = "malefic_config"
var community = "community"
var prebuild = "prebuild"

func getDockerClient() (*client.Client, error) {
	var err error
	once.Do(func() {
		dockerClient, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		if err != nil {
			logs.Log.Errorf("Error creating Docker client: %v", err)
		}
	})
	return dockerClient, err
}

func (rpc *Server) Generate(ctx context.Context, req *clientpb.Generate) (*clientpb.Empty, error) {
	cli, err := getDockerClient()
	if err != nil {
		return &clientpb.Empty{}, err
	}
	err = build.DbToConfig(req)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Err create config: %v", err))
	}
	logs.Log.Infof("start to build ...", req.Target)

	switch req.Type {
	case consts.PE:
		err = build.BuildPE(cli, req)
		if err != nil {
			return nil, err
		}
	}

	fileName, err := db.SaveBuilder(req)
	if fileName != "" {
		err = build.MoveBuildOutput(req.Target, req.Type, fileName)
	} else {
		logs.Log.Errorf("save builder error: %v, you can find build output in ./malice/build/target/%s/", err,
			req.Target)
	}
	return &clientpb.Empty{}, err
}

func (rpc *Server) GetBuilders(ctx context.Context, req *clientpb.Empty) (*clientpb.Builders, error) {
	builders, err := db.GetBuilders()
	if err != nil {
		return nil, err
	}
	return builders, nil
}

func (rpc *Server) DownloadOutput(ctx context.Context, req *clientpb.Sync) (*clientpb.SyncResp, error) {
	filePath, err := build.GetOutPutPath(req.FileId)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return &clientpb.SyncResp{
		Name:    filepath.Base(filePath),
		Content: data,
	}, nil
}
