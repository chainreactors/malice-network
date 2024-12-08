package rpc

import (
	"context"
	"fmt"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/build"
	"github.com/chainreactors/malice-network/server/internal/configs"
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
