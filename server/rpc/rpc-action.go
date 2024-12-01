package rpc

import (
	"context"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/build"
)

// TriggerWorkflowDispatch triggers a workflow dispatch event
func (rpc *Server) TriggerWorkflowDispatch(ctx context.Context, req *clientpb.WorkflowRequest) (*clientpb.Builder, error) {
	builder, err := build.TriggerWorkflowDispatch(req.Owner, req.Repo, req.WorkflowId, req.Token, req.Inputs)
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

func (rpc *Server) ListArtifacts(ctx context.Context, req *clientpb.WorkflowRequest) (*clientpb.ListArtifactsResponse, error) {
	artifacts, err := build.ListArtifacts(req.Owner, req.Repo, req.Token)
	if err != nil {
		return nil, err
	}
	protoArtifacts := make([]*clientpb.Artifact, len(artifacts))
	for i, artifact := range artifacts {
		protoArtifacts[i] = artifact.ToProtoBuf()
	}

	response := &clientpb.ListArtifactsResponse{
		Artifacts: protoArtifacts,
	}
	return response, nil
}

func (rpc *Server) DownloadGithubArtifact(ctx context.Context, req *clientpb.WorkflowRequest) (*clientpb.DownloadArtifactsResponse, error) {
	resp, err := build.DownloadArtifact(req.Owner, req.Repo, req.Token, req.BuildName)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
