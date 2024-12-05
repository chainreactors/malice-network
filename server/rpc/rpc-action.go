package rpc

import (
	"context"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/build"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"strings"
)

// TriggerWorkflowDispatch triggers a workflow dispatch event
func (rpc *Server) TriggerWorkflowDispatch(ctx context.Context, req *clientpb.WorkflowRequest) (*clientpb.Builder, error) {
	var modules []string
	if req.Inputs["malefic_modules_features"] != "" {
		modules = strings.Split(req.Inputs["malefic_modules_features"], ",")
	}
	if req.Owner == "" || req.Repo == "" || req.Token == "" {
		config := configs.GetServerConfig()
		req.Owner = config.GithubConfig.Owner
		req.Repo = config.GithubConfig.Repo
		req.Token = config.GithubConfig.Token
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

// EnableWorkflow enables a GitHub Actions workflow
func (rpc *Server) EnableWorkflow(ctx context.Context, req *clientpb.WorkflowRequest) (*clientpb.Empty, error) {
	err := build.EnableWorkflow(req.Owner, req.Repo, req.WorkflowId, req.Token)
	if err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, nil
}

// DisableWorkflow disables a GitHub Actions workflow
func (rpc *Server) DisableWorkflow(ctx context.Context, req *clientpb.WorkflowRequest) (*clientpb.Empty, error) {
	err := build.DisableWorkflow(req.Owner, req.Repo, req.WorkflowId, req.Token)
	if err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, nil
}

// ListRepositoryWorkflows fetches the workflows for a given repository
func (rpc *Server) ListRepositoryWorkflows(ctx context.Context, req *clientpb.WorkflowRequest) (*clientpb.ListWorkflowsResponse, error) {
	workflows, err := build.ListRepositoryWorkflows(req.Owner, req.Repo, req.Token)
	if err != nil {
		return nil, err
	}

	protoWorkflows := make([]*clientpb.Workflow, len(workflows))
	for i, wf := range workflows {
		protoWorkflows[i] = wf.ToProtoBuf()
	}

	response := &clientpb.ListWorkflowsResponse{
		Workflows: protoWorkflows,
	}

	return response, nil
}
