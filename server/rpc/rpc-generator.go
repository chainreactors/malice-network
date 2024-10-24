package rpc

import (
	"context"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/rpc/generator"
	"github.com/docker/docker/client"
	"sync"
)

var dockerClient *client.Client
var once sync.Once

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
	err = generator.DbToConfig(req)
	if err != nil {
		return nil, err
	}
	switch req.Type {
	case consts.CommandPE:
		err = generator.BuildPE(cli)
		if err != nil {
			return nil, err
		}
	}
	return &clientpb.Empty{}, nil
	//if req.Target != "" {
	//	profile.Target = req.Target
	//}
	//if req.Type != "" {
	//	profile.Type = req.Type
	//}
	//if len(req.Params) > 0 {
	//	profile.Params = req.Params
	//}

}
