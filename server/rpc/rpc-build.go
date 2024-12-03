package rpc

import (
	"context"
	"errors"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/codenames"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/encoders"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/build"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"os"
	"path/filepath"
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

	return builder.ToProtobuf(nil), nil
}

func (rpc *Server) ListArtifact(ctx context.Context, req *clientpb.Empty) (*clientpb.Builders, error) {
	builders, err := db.GetArtifacts()
	if err != nil {
		return nil, err
	}
	return builders, nil
}

func (rpc *Server) DownloadArtifact(ctx context.Context, req *clientpb.Builder) (*clientpb.Builder, error) {
	builder, err := db.GetArtifactByName(req.Name)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(builder.Path)
	if err != nil {
		return nil, err
	}
	result := builder.ToProtobuf(data)
	result.Name = result.Name + filepath.Ext(builder.Path)
	return result, nil
}

func (rpc *Server) MaleficSRDI(ctx context.Context, req *clientpb.Builder) (*clientpb.Builder, error) {
	var filePath, realName string
	var builder *models.Builder
	var err error
	if req.Id != 0 {
		builder, err = db.GetArtifactById(req.Id)
		if err != nil {
			return nil, err
		}
		filePath = builder.Path
		realName = builder.Name
	} else {
		dst := encoders.UUID()
		filePath = filepath.Join(configs.TempPath, dst)
		realName = req.Name
		err = build.SaveArtifact(dst, req.Bin)
	}

	builder, bin, err := build.NewMaleficSRDIArtifact(realName+"_"+consts.ShellcodeTYPE, filePath, req.Platform, req.Arch, req.Stage, req.FunctionName, req.UserDataPath)
	if err != nil {
		return nil, err
	}
	return builder.ToProtobuf(bin), nil
}

func (rpc *Server) UploadArtifact(ctx context.Context, req *clientpb.Builder) (*clientpb.Builder, error) {
	if req.Name == "" {
		req.Name = codenames.GetCodename()
	}
	builder, err := db.SaveArtifact(req.Name, req.Type, req.Platform, req.Arch, req.Stage)
	if err != nil {
		return nil, err
	}
	err = os.WriteFile(builder.Path, req.Bin, 0644)
	if err != nil {
		return nil, err
	}
	return builder.ToProtobuf(nil), nil
}

// for listener
func (rpc *Server) GetArtifact(ctx context.Context, req *clientpb.Builder) (*clientpb.Builder, error) {
	builder, err := db.GetArtifactById(req.Id)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(builder.Path)
	if err != nil {
		return nil, err
	}

	return builder.ToProtobuf(data), nil
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
	builder, err := db.GetBuilderByModules(req.Modules)
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
		data, err := os.ReadFile(artifactPath)
		if err != nil {
			return nil, err
		}
		return builder.ToProtobuf(data), nil
	}
	data, err := os.ReadFile(builder.Path)
	if err != nil {
		return nil, err
	}
	return builder.ToProtobuf(data), nil
}
