package rpc

import (
	"context"

	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/db"
)

func (rpc *Server) GetContexts(ctx context.Context, req *clientpb.Context) (*clientpb.Contexts, error) {
	contexts, err := db.GetAllContext()
	if err != nil {
		return nil, err
	}

	result := &clientpb.Contexts{
		Contexts: make([]*clientpb.Context, 0),
	}
	for _, c := range contexts {
		result.Contexts = append(result.Contexts, c.ToProtobuf())
	}
	return result, nil
}

func (rpc *Server) GetScreenShot(ctx context.Context, req *clientpb.Empty) (*clientpb.Contexts, error) {
	contexts, err := db.GetContextsByType(consts.ContextScreenShot)
	if err != nil {
		return nil, err
	}
	return &clientpb.Contexts{Contexts: contexts}, nil
}

func (rpc *Server) GetCredential(ctx context.Context, req *clientpb.Empty) (*clientpb.Contexts, error) {
	contexts, err := db.GetContextsByType(consts.ContextCredential)
	if err != nil {
		return nil, err
	}
	return &clientpb.Contexts{Contexts: contexts}, nil
}

func (rpc *Server) GetKeylogger(ctx context.Context, req *clientpb.Empty) (*clientpb.Contexts, error) {
	contexts, err := db.GetContextsByType(consts.ContextKeyLogger)
	if err != nil {
		return nil, err
	}
	return &clientpb.Contexts{Contexts: contexts}, nil
}

func (rpc *Server) GetPivoting(ctx context.Context, req *clientpb.Empty) (*clientpb.Contexts, error) {
	contexts, err := db.GetContextsByType(consts.ContextPivoting)
	if err != nil {
		return nil, err
	}
	return &clientpb.Contexts{Contexts: contexts}, nil
}

func (rpc *Server) AddContext(ctx context.Context, req *clientpb.Context) (*clientpb.Empty, error) {
	err := db.SaveContext(req)
	if err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, nil
}
