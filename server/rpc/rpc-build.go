package rpc

import (
	"context"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/build"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
)

func (rpc *Server) Build(ctx context.Context, req *clientpb.BuildConfig) (*clientpb.Artifact, error) {
	builder, err := build.NewBuilder(req)
	if err != nil {
		return nil, err
	}
	artifact, err := builder.Generate()
	if err != nil {
		return nil, err
	}
	go func() {
		executeErr := builder.Execute()
		if executeErr == nil {
			builder.Collect()
		} else {
			logs.Log.Errorf("failed to build %s: %s", artifact.Name, executeErr)
			build.SendFailedMsg(artifact)
		}
	}()
	return artifact, nil
}

func (rpc *Server) BuildLog(ctx context.Context, req *clientpb.Artifact) (*clientpb.Artifact, error) {
	resultLog, err := db.GetBuilderLogs(req.Name, int(req.LogNum))
	if err != nil {
		return nil, err
	}
	req.Log = []byte(resultLog)
	return req, nil
}

func (rpc *Server) DockerStatus(ctx context.Context, req *clientpb.Empty) (*clientpb.Empty, error) {
	cli, err := build.GetDockerClient()
	if err != nil {
		return nil, err
	}
	_, err = cli.Ping(ctx)
	if err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, nil
}

func (rpc *Server) WorkflowStatus(ctx context.Context, req *clientpb.GithubWorkflowConfig) (*clientpb.Empty, error) {
	if req.Owner == "" || req.Repo == "" || req.Token == "" {
		config := configs.GetGithubConfig()
		if config == nil {
			return nil, fmt.Errorf("please set github config use flag or server config")
		}
		req = config.ToProtobuf()
	}
	err := build.GetWorkflowStatus(req)
	if err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, nil
}
