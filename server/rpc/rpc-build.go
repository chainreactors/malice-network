package rpc

import (
	"context"
	"errors"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/codenames"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/encoders"
	"github.com/chainreactors/malice-network/helper/errs"
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
	target, ok := consts.GetBuildTarget(req.Target)
	if !ok {
		return nil, errs.ErrInvalidateTarget
	}
	cli, err := build.GetDockerClient()
	if err != nil {
		return nil, err
	}
	err = build.GenerateProfile(req)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Err create config: %v", err))
	}
	logs.Log.Infof("start to build %s ...", req.Target)

	switch req.Type {
	case consts.CommandBuildBeacon:
		err = build.BuildBeacon(cli, req)
	case consts.CommandBuildBind:
		err = build.BuildBind(cli, req)
	case consts.CommandBuildPrelude:
		err = build.BuildPrelude(cli, req)
	case consts.CommandBuildModules:
		err = build.BuildModules(cli, req)
	case consts.CommandBuildPulse:
		err = build.BuildPulse(cli, req)
	}
	if err != nil {
		return nil, err
	}
	maleficPath, artifactPath, err := build.MoveBuildOutput(req.Target, req.Type)
	if err != nil {
		return nil, err
	}

	builder, err := db.SaveArtifactFromGenerate(req, filepath.Base(maleficPath), artifactPath)
	if err != nil {
		logs.Log.Errorf("move build output error: %v", err)
		return nil, err
	}
	if !req.Srdi {
		data, err := os.ReadFile(artifactPath)
		if err != nil {
			return nil, err
		}
		return builder.ToProtobuf(data), nil
	} else {
		builder, bin, err := build.NewMaleficSRDIArtifact(req.Name+"_srdi", artifactPath, target.OS, target.Arch, req.Stager, "", "")
		if err != nil {
			return nil, err
		}
		return builder.ToProtobuf(bin), nil
	}
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
	return builder.ToProtobuf(data), nil
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

	builder, bin, err := build.NewMaleficSRDIArtifact(realName+"_"+consts.SRDIType, filePath, req.Platform, req.Arch, req.Stage, req.FunctionName, req.UserDataPath)
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
