package rpc

import (
	"context"
	"errors"
	"fmt"
	"github.com/chainreactors/logs"
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
	fileName, _, err := db.SaveBuilderFromGenerate(req)
	_, srcPath, err := build.MoveBuildOutput(req.Target, req.Platform, fileName)
	if fileName != "" {
		if err != nil {
			logs.Log.Errorf("move build output error: %v", err)
			return nil, err
		}
		if req.ShellcodeType != "" {
			req.Stager = "shellcode"
			shellCodeName, srdiPath, _ := db.SaveBuilderFromGenerate(req)
			data, err := build.MaleficSRDI(&clientpb.MutantFile{
				Id:       shellCodeName,
				Type:     req.ShellcodeType,
				Arch:     req.Target,
				Platform: req.Platform,
			}, srcPath, srdiPath)
			if err != nil {
				return nil, err
			}
			return &clientpb.Bin{
				Bin:  data,
				Name: strings.TrimSuffix(shellCodeName, filepath.Ext(shellCodeName)),
			}, err
		}
	} else {
		logs.Log.Errorf("save builder error: %v, you can find build output in ./malice/build/target/%s/", err,
			req.Target)
	}
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return nil, err
	}
	return &clientpb.Bin{
		Bin:  data,
		Name: fileName,
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
	filePath, err := db.GetBuilderPath(req.FileId)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return &clientpb.SyncResp{
		Name:    filepath.Base(filePath),
		Content: data,
	}, nil
}
