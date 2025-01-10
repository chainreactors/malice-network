package rpc

import (
	"context"
	"errors"
	"fmt"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/build"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"os"
	"strings"
)

// TriggerWorkflowDispatch triggers a workflow dispatch event
func (rpc *Server) TriggerWorkflowDispatch(ctx context.Context, req *clientpb.GithubWorkflowRequest) (*clientpb.Builder, error) {
	var modules []string
	if req.Inputs["malefic_modules_features"] != "" {
		modules = strings.Split(req.Inputs["malefic_modules_features"], ",")
	}
	if req.Owner == "" || req.Repo == "" || req.Token == "" {
		config := configs.GetGithubConfig()
		if config == nil {
			return nil, fmt.Errorf("please set github config use flag or server config")
		}
		req.Owner = config.Owner
		req.Repo = config.Repo
		req.Token = config.Token
		req.WorkflowId = config.Workflow
	}
	if req.Inputs["package"] == consts.CommandBuildModules {
		moduleBuilder, err := db.GetBuilderByModules(req.Inputs["targets"], modules)
		if err != nil && !errors.Is(err, db.ErrRecordNotFound) {
			return nil, err
		}
		if moduleBuilder.Path != "" {
			bin, err := os.ReadFile(moduleBuilder.Path)
			if err != nil {
				return nil, err
			}
			moduleBuilder.Name = build.GetFilePath(moduleBuilder.Name, moduleBuilder.Target, moduleBuilder.Type, moduleBuilder.IsSRDI)
			result := moduleBuilder.ToProtobuf()
			result.Bin = bin
			return result, nil
		}
	}
	generateReq := &clientpb.Generate{
		ProfileName: req.Profile,
		Address:     req.Address,
		Type:        req.Inputs["package"],
		Modules:     modules,
		Ca:          req.Ca,
		Params:      req.Params,
	}
	builder, err := build.TriggerWorkflowDispatch(req.Owner, req.Repo, req.WorkflowId, req.Token, req.Inputs, generateReq)
	if err != nil {
		return nil, err
	}
	return builder, nil
}

func (rpc *Server) WorkflowStatus(ctx context.Context, req *clientpb.GithubWorkflowRequest) (*clientpb.Empty, error) {
	if req.Owner == "" || req.Repo == "" || req.Token == "" {
		config := configs.GetGithubConfig()
		if config == nil {
			return nil, fmt.Errorf("please set github config use flag or server config")
		}
		req.Owner = config.Owner
		req.Repo = config.Repo
		req.Token = config.Token
		req.WorkflowId = config.Workflow
	}
	err := build.GetWorkflowStatus(req.Owner, req.Repo, req.WorkflowId, req.Token)
	if err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, nil
}
