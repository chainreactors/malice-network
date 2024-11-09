package rpc

import (
	"context"
	"errors"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/encoders"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/build"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"os"
	"path/filepath"
)

func (rpc *Server) Build(ctx context.Context, req *clientpb.Generate) (*clientpb.Bin, error) {
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
	req.Name = encoders.UUID()
	maleficPath, buildSrcPath, err := build.MoveBuildOutput(req.Target, req.Platform, req.Name)
	if err != nil {
		return nil, err
	}

	_, err = db.SaveArtifactFromGenerate(req, filepath.Base(maleficPath))
	if err != nil {
		logs.Log.Errorf("move build output error: %v", err)
		return nil, err
	}
	if req.ShellcodeType != "" {
		req.Stager = "shellcode"
		srdiPath, _ := db.SaveArtifactFromGenerate(req, filepath.Base(buildSrcPath))
		data, err := build.MaleficSRDI(buildSrcPath, srdiPath, req.Target, req.Platform)
		if err != nil {
			return nil, err
		}
		return &clientpb.Bin{
			Bin:  data,
			Name: req.Name,
		}, err
	} else {
		data, err := os.ReadFile(buildSrcPath)
		if err != nil {
			return nil, err
		}
		return &clientpb.Bin{
			Bin:  data,
			Name: req.Name,
		}, err
	}
}

func (rpc *Server) ListArtifact(ctx context.Context, req *clientpb.Empty) (*clientpb.Builders, error) {
	builders, err := db.GetArtifacts()
	if err != nil {
		return nil, err
	}
	return builders, nil
}

func (rpc *Server) DownloadArtifact(ctx context.Context, req *clientpb.Sync) (*clientpb.SyncResp, error) {
	filePath, fileName, err := db.GetArtifactByName(req.FileId)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return &clientpb.SyncResp{
		Name:    fileName,
		Content: data,
	}, nil
}

func (rpc *Server) MaleficSRDI(ctx context.Context, req *clientpb.MutantFile) (*clientpb.Bin, error) {
	if req.Id != "" {
		filePath, realName, err := db.GetArtifactByName(req.Id)
		if err != nil {
			return nil, err
		}
		bin, err := build.MaleficSRDI(realName, filePath, req.Platform, req.Arch)
		if err != nil {
			return nil, err
		}
		return &clientpb.Bin{Bin: bin, Name: realName}, nil
	} else {
		src := encoders.UUID()
		err := build.SaveArtifact(src, req.Bin)
		bin, err := build.MaleficSRDI(req.Name, src, req.Platform, req.Arch)
		if err != nil {
			return nil, err
		}
		return &clientpb.Bin{Bin: bin, Name: req.Name}, nil
	}
}

func (rpc *Server) UploadArtifact(ctx context.Context, req *clientpb.Bin) (*clientpb.Empty, error) {
	_, dstPath, err := db.AddArtifact(req.Name, req.Type)
	if err != nil {
		return nil, err
	}
	srcPath := filepath.Join(configs.BuildOutputPath, dstPath)
	err = os.WriteFile(srcPath, req.Bin, 0644)
	if err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, nil
}
