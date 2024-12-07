package rpc

import (
	"context"
	"errors"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/codenames"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/build"
	"github.com/chainreactors/malice-network/server/internal/db"
	"os"
)

func (rpc *Server) Build(ctx context.Context, req *clientpb.Generate) (*clientpb.Builder, error) {
	if req.Name == "" {
		req.Name = codenames.GetCodename()
	}
	builder, err := db.SaveArtifactFromGenerate(req)
	if err != nil {
		logs.Log.Errorf("save build db error: %v", err)
		return nil, err
	}
	go func() {
		_, err = build.GlobalBuildQueueManager.AddTask(req, *builder)
		if err != nil {
			logs.Log.Errorf("Failed to enqueue build request: %v", err)
			return
		}
	}()
	logs.Log.Infof("Build request processed successfully for target: %s", req.Target)

	return builder.ToProtobuf(), nil
}

func (rpc *Server) BuildLog(ctx context.Context, req *clientpb.Builder) (*clientpb.Builder, error) {
	resultLog, err := db.GetBuilderLogs(req.Name, int(req.Num))
	if err != nil {
		return nil, err
	}
	req.Log = []byte(resultLog)
	return req, nil
}

func (rpc *Server) BuildModules(ctx context.Context, req *clientpb.Generate) (*clientpb.Builder, error) {
	builder, err := db.GetBuilderByModules(req.Target, req.Modules)
	if err != nil && !errors.Is(err, db.ErrRecordNotFound) {
		return nil, err
	} else if errors.Is(err, db.ErrRecordNotFound) {
		if req.Name == "" {
			req.Name = codenames.GetCodename()
		}
		cli, err := build.GetDockerClient()
		if err != nil {
			return nil, err
		}
		logs.Log.Infof("start to build %s ...", req.Target)
		builder, err := db.SaveArtifactFromGenerate(req)
		if err != nil {
			logs.Log.Errorf("move build output error: %v", err)
			return nil, err
		}
		err = build.BuildModules(cli, req, true)
		if err != nil {
			return nil, err
		}
		_, artifactPath, err := build.MoveBuildOutput(req.Target, consts.CommandBuildModules)
		if err != nil {
			logs.Log.Errorf("move build output error: %v", err)
			return nil, err
		}
		builder.Path = artifactPath
		err = db.UpdateBuilderPath(builder)
		if err != nil {
			return nil, err
		}
		_, err = os.ReadFile(artifactPath)
		if err != nil {
			return nil, err
		}
		return builder.ToProtobuf(), nil
	}
	_, err = os.ReadFile(builder.Path)
	if err != nil {
		return nil, err
	}
	return builder.ToProtobuf(), nil
}
