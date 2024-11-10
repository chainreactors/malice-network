package rpc

import (
	"context"
	"errors"
	"fmt"
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
	cli, err := build.GetDockerClient()
	if err != nil {
		return nil, err
	}
	err = build.DbToConfig(req)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Err create config: %v", err))
	}
	logs.Log.Infof("start to build %s ...", req.Target)

	switch req.Type {
	case consts.CommandBuildBeacon:
		err = build.BuildBeacon(cli, req)
		if err != nil {
			return nil, err
		}
	case consts.CommandBuildBind:
		err = build.BuildBind(cli, req)
		if err != nil {
			return nil, err
		}
	case consts.CommandBuildPrelude:
		err = build.BuildPrelude(cli, req)
		if err != nil {
			return nil, err
		}
	case consts.CommandBuildModules:
		err = build.BuildModules(cli, req)
		if err != nil {
			return nil, err
		}
	case consts.CommandBuildLoader:
		err = build.BuildLoader(cli, req)
		if err != nil {
			return nil, err
		}
	}
	maleficPath, artifactPath, err := build.MoveBuildOutput(req.Target, req.Platform)
	if err != nil {
		return nil, err
	}

	builder, err := db.SaveArtifactFromGenerate(req, filepath.Base(maleficPath), artifactPath)
	if err != nil {
		logs.Log.Errorf("move build output error: %v", err)
		return nil, err
	}
	if req.ShellcodeType == "" {
		data, err := os.ReadFile(artifactPath)
		if err != nil {
			return nil, err
		}
		return &clientpb.Builder{
			Bin:  data,
			Name: req.Name,
			Id:   builder.ID,
		}, nil
	} else {
		builder, bin, err := build.NewMaleficSRDIArtifact(req.Name+"_srdi", artifactPath, req.Platform, "", req.Stager)
		if err != nil {
			return nil, err
		}
		return &clientpb.Builder{
			Bin:  bin,
			Name: builder.Name,
			Id:   builder.ID,
		}, nil
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
	return &clientpb.Builder{
		Id:   builder.ID,
		Name: builder.Name,
		Bin:  data,
	}, nil
}

func (rpc *Server) MaleficSRDI(ctx context.Context, req *clientpb.Builder) (*clientpb.Builder, error) {
	var filePath, realName string
	var builder *models.Builder
	var err error
	if req.Name != "" {
		builder, err = db.GetArtifactByName(req.Name)
		filePath = builder.Path
		realName = builder.Name
	} else if req.Id != 0 {
		builder, err = db.GetArtifactById(req.Id)
		filePath = builder.Path
		realName = builder.Name
	} else {
		dst := encoders.UUID()
		filePath = filepath.Join(configs.BuildOutputPath, dst)
		realName = req.Name
		err = build.SaveArtifact(dst, req.Bin)
	}
	if err != nil {
		return nil, err
	}

	builder, bin, err := build.NewMaleficSRDIArtifact(realName+"_srdi", filePath, req.Platform, req.Arch, req.Stage)
	if err != nil {
		return nil, err
	}
	return &clientpb.Builder{Bin: bin, Name: req.Name, Id: builder.ID}, nil
}

func (rpc *Server) UploadArtifact(ctx context.Context, req *clientpb.Builder) (*clientpb.Builder, error) {
	if req.Name == "" {
		req.Name = codenames.GetCodename()
	}
	builder, err := db.SaveArtifact(req.Name, req.Type, req.Stage)
	if err != nil {
		return nil, err
	}
	err = os.WriteFile(builder.Path, req.Bin, 0644)
	if err != nil {
		return nil, err
	}
	return &clientpb.Builder{
		Id:   builder.ID,
		Name: builder.Name,
	}, nil
}
