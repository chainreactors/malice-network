package rpc

import (
	"context"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/errs"
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
		if execErr := builder.Execute(); execErr != nil {
			logs.Log.Errorf("failed to build %s: %s", artifact.Name, execErr)
			build.SendBuildMsg(artifact, consts.BuildStatusFailure, make([]byte, 0))
			return
		}

		_, status := builder.Collect()
		if status == consts.BuildStatusCompleted {
			if amtErr := build.AmountArtifact(artifact.Name); amtErr != nil {
				logs.Log.Errorf("failed to add artifact path to website: %s", amtErr)
			}
		}
		build.SendBuildMsg(artifact, status, req.ParamsBytes)
	}()

	return artifact, nil
}

func (rpc *Server) SyncBuild(ctx context.Context, req *clientpb.BuildConfig) (*clientpb.Artifact, error) {
	builder, err := build.NewBuilder(req)
	if err != nil {
		return nil, err
	}
	artifact, err := builder.Generate()
	if err != nil {
		return nil, err
	}
	err = builder.Execute()
	if err == nil {
		builder.Collect()
	} else {
		return nil, err
	}
	return db.FindArtifact(artifact)
}

func (rpc *Server) BuildLog(ctx context.Context, req *clientpb.Artifact) (*clientpb.Artifact, error) {
	resultLog, err := db.GetBuilderLogs(req.Name, int(req.LogNum))
	if err != nil {
		return nil, err
	}
	req.Log = []byte(resultLog)
	return req, nil
}

func (rpc *Server) CheckSource(ctx context.Context, req *clientpb.BuildConfig) (*clientpb.BuildConfig, error) {
	if cli, err := build.GetDockerClient(); err == nil {
		if _, err := cli.Ping(ctx); err == nil {
			req.Source = consts.ArtifactFromDocker
			return req, nil
		}
	}
	if req.Github == nil {
		if config := configs.GetGithubConfig(); config != nil {
			req.Github = config.ToProtobuf()
		} else {
			req.Github = nil
		}
	}
	if err := build.GetWorkflowStatus(req.Github); err == nil {
		req.Source = consts.ArtifactFromAction
		return req, nil
	}
	if saasConfig := configs.GetSaasConfig(); saasConfig != nil && saasConfig.Enable && saasConfig.Url != "" && saasConfig.Token != "" {
		req.Source = consts.ArtifactFromSaas
		return req, nil
	}

	return nil, errs.ErrSouceUnable
}
