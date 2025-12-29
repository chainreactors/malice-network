package rpc

import (
	"context"
	"fmt"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/mutant"
)

// MutantSrdi converts DLL to shellcode using malefic-mutant srdi tool
func (rpc *Server) MutantSrdi(ctx context.Context, req *clientpb.MutantSrdiRequest) (*clientpb.Bin, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	if len(req.Bin) == 0 {
		return nil, fmt.Errorf("input binary is required")
	}

	// Convert protobuf request to internal request
	mutantReq := &mutant.SrdiRequest{
		Bin:          req.Bin,
		Arch:         req.Arch,
		FunctionName: req.FunctionName,
		Platform:     req.Platform,
		Type:         req.Type,
		Userdata:     req.Userdata,
	}

	// Call the srdi tool
	shellcode, err := mutant.Srdi(mutantReq)
	if err != nil {
		return nil, fmt.Errorf("srdi failed: %w", err)
	}

	return &clientpb.Bin{Bin: shellcode}, nil
}

// MutantStrip removes paths from binary files using malefic-mutant strip tool
func (rpc *Server) MutantStrip(ctx context.Context, req *clientpb.MutantStripRequest) (*clientpb.Bin, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	if len(req.Bin) == 0 {
		return nil, fmt.Errorf("input binary is required")
	}

	// Convert protobuf request to internal request
	mutantReq := &mutant.StripRequest{
		Bin:         req.Bin,
		CustomPaths: req.CustomPaths,
	}

	// Call the strip tool
	stripped, err := mutant.Strip(mutantReq)
	if err != nil {
		return nil, fmt.Errorf("strip failed: %w", err)
	}

	return &clientpb.Bin{Bin: stripped}, nil
}

// MutantSigforge manipulates PE file signatures using malefic-mutant sigforge tool
func (rpc *Server) MutantSigforge(ctx context.Context, req *clientpb.MutantSigforgeRequest) (*clientpb.Bin, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	if req.Operation == "" {
		return nil, fmt.Errorf("operation is required")
	}

	if len(req.SourceBin) == 0 {
		return nil, fmt.Errorf("source binary is required")
	}

	// Convert protobuf request to internal request
	mutantReq := &mutant.SigforgeRequest{
		Operation: req.Operation,
		SourceBin: req.SourceBin,
		TargetBin: req.TargetBin,
		Signature: req.Signature,
	}

	// Call the sigforge tool
	result, err := mutant.Sigforge(mutantReq)
	if err != nil {
		return nil, fmt.Errorf("sigforge failed: %w", err)
	}

	return &clientpb.Bin{Bin: result}, nil
}
