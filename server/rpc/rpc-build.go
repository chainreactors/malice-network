package rpc

import (
	"context"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/build"
	"github.com/chainreactors/malice-network/server/internal/core"
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

	core.SafeGo(func() {
		if execErr := builder.Execute(); execErr != nil {
			logs.Log.Errorf("failed to build %s: %s", artifact.Name, execErr)
			build.SendBuildMsg(artifact, consts.BuildStatusFailure, make([]byte, 0), execErr)
			return
		}

		_, status, err := builder.Collect()
		if status == consts.BuildStatusCompleted {
			if amtErr := build.AmountArtifact(artifact.Name); amtErr != nil {
				logs.Log.Errorf("failed to add artifact path to website: %s", amtErr)
			}
		}
		//build.SendBuildMsg(artifact, status, req.ParamsBytes, err)
		build.SendBuildMsg(artifact, status, make([]byte, 0), err)
	})

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
	return db.FindArtifact(artifact, true)
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
	return build.CheckSource(ctx, req)
}
