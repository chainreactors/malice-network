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

	beaconBuilder, builder, err := build.NewBuilder(req)
	if err != nil {
		return nil, err
	}
	if beaconBuilder != nil {
		beaconArtifact, err := beaconBuilder.GenerateConfig()
		if err != nil {
			return nil, err
		}
		go func() {
			executeErr := beaconBuilder.ExecuteBuild()
			if executeErr == nil {
				beaconBuilder.CollectArtifact()
			} else {
				logs.Log.Errorf("failed to build %s: %s", beaconArtifact.Name, executeErr)
				build.SendFailedMsg(beaconArtifact)
			}
		}()
		if builder.GetBeaconID() != 0 {
			err = builder.SetBeaconID(beaconArtifact.Id)
			if err != nil {
				return nil, err
			}
		}
	}
	artifact, err := builder.GenerateConfig()
	if err != nil {
		return nil, err
	}
	go func() {
		executeErr := builder.ExecuteBuild()
		if executeErr == nil {
			builder.CollectArtifact()
		} else {
			logs.Log.Errorf("failed to build %s: %s", artifact.Name, executeErr)
		}
	}()

	return artifact, nil
}

func (rpc *Server) BuildLog(ctx context.Context, req *clientpb.Artifact) (*clientpb.Artifact, error) {
	resultLog, err := db.GetBuilderLogs(req.Id, int(req.LogNum))
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
