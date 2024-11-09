package rpc

import (
	"context"
	"errors"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/codenames"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/build"
	"github.com/chainreactors/malice-network/server/internal/db"
	"os"
	"path/filepath"
	"strings"
)

func (rpc *Server) Generate(ctx context.Context, req *clientpb.Generate) (*clientpb.Bin, error) {
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
	case consts.CommandBeacon:
		err = build.BuildBeacon(cli, req)
		if err != nil {
			return nil, err
		}
	case consts.CommandBind:
		err = build.BuildBind(cli, req)
		if err != nil {
			return nil, err
		}
	case consts.CommandPrelude:
		err = build.BuildPrelude(cli, req)
		if err != nil {
			return nil, err
		}
	case consts.CommandModules:
		err = build.BuildModules(cli, req)
		if err != nil {
			return nil, err
		}
	case consts.CommandLoader:
		err = build.BuildLoader(cli, req)
		if err != nil {
			return nil, err
		}
	}
	name, err := codenames.GetCodename()
	if err != nil {
		return nil, err
	}
	req.Name = name
	maleficPath, buildSrcPath, err := build.MoveBuildOutput(req.Target, req.Platform, name)
	if err != nil {
		return nil, err
	}
	_, err = db.SaveBuilderFromGenerate(req, filepath.Base(maleficPath))
	if err != nil {
		logs.Log.Errorf("move build output error: %v", err)
		return nil, err
	}
	if req.ShellcodeType != "" {
		req.Stager = "shellcode"
		shellCodeName, err := codenames.GetCodename()
		if err != nil {
			return nil, err
		}
		req.Name = shellCodeName
		srdiPath, _ := db.SaveBuilderFromGenerate(req, filepath.Base(buildSrcPath))
		data, err := build.MaleficSRDI(&clientpb.MutantFile{
			Name:     shellCodeName,
			Type:     req.ShellcodeType,
			Arch:     req.Target,
			Platform: req.Platform,
		}, buildSrcPath, srdiPath)
		if err != nil {
			return nil, err
		}
		return &clientpb.Bin{
			Bin:  data,
			Name: strings.TrimSuffix(shellCodeName, filepath.Ext(shellCodeName)),
		}, err
	}

	data, err := os.ReadFile(buildSrcPath)
	if err != nil {
		return nil, err
	}
	return &clientpb.Bin{
		Bin:  data,
		Name: name,
	}, err
}

func (rpc *Server) GetBuilders(ctx context.Context, req *clientpb.Empty) (*clientpb.Builders, error) {
	builders, err := db.GetBuilders()
	if err != nil {
		return nil, err
	}
	return builders, nil
}

func (rpc *Server) DownloadOutput(ctx context.Context, req *clientpb.Sync) (*clientpb.SyncResp, error) {
	filePath, fileName, err := db.GetBuilderResource(req.FileId)
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
