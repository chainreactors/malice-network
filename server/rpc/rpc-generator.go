package rpc

import (
	"context"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/build"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/docker/docker/client"
	"os/exec"
	"path/filepath"
	"sync"
)

var dockerClient *client.Client
var once sync.Once
var maleficConfig = "malefic_config"
var community = "community"
var prebuild = "prebuild"

func setEnv() error {
	return nil
}

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
		return nil, err
	}

	cmd := exec.Command(filepath.Join(configs.BuildPath, maleficConfig), req.Stager, community, prebuild)
	_, err = cmd.CombinedOutput()
	if err != nil {
		logs.Log.Errorf("exec failed %s", err)
	}
	//logs.Log.Infof("config output %s", output)
	switch req.Type {
	case consts.CommandPE:
		err = build.BuildPE(cli, req)
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
