package rpc

import (
	"context"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"github.com/gofrs/uuid"
	"strconv"
)

func (rpc *Server) GetContexts(ctx context.Context, req *clientpb.Context) (*clientpb.Contexts, error) {
	contexts := make([]*clientpb.Context, 0)

	filterFunc := func(c *core.Context) bool {
		if req.Session != nil && req.Session.SessionId != "" && c.Session.ID != req.Session.SessionId {
			return false
		}
		if req.Task != nil && req.Task.TaskId != 0 && c.Task.TaskID() != req.Task.SessionId+"-"+strconv.Itoa(int(req.Task.TaskId)) {
			return false
		}
		if req.Pipeline != nil && req.Pipeline.Name != "" && c.Pipeline.Name != req.Pipeline.Name {
			return false
		}
		if req.Listener != nil && req.Listener.Id != "" && c.Listener.Name != req.Listener.Id {
			return false
		}
		return true
	}

	for _, c := range core.Contexts.All() {
		if filterFunc(c) {
			contexts = append(contexts, c.ToProtobuf())
		}
	}

	return &clientpb.Contexts{Contexts: contexts}, nil
}
func (rpc *Server) GetScreenShot(ctx context.Context, req *clientpb.Empty) (*clientpb.Contexts, error) {
	return core.Contexts.ScreenShot(), nil
}

func (rpc *Server) GetCredential(ctx context.Context, req *clientpb.Empty) (*clientpb.Contexts, error) {
	return core.Contexts.Credential(), nil
}

func (rpc *Server) GetKeylogger(ctx context.Context, req *clientpb.Empty) (*clientpb.Contexts, error) {
	return core.Contexts.KeyLogger(), nil
}

func (rpc *Server) AddContext(ctx context.Context, req *clientpb.Context) (*clientpb.Empty, error) {
	cID, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	req.Id = cID.String()
	newContext, err := core.NewContext(req)
	if err != nil {
		return nil, err
	}
	core.Contexts.Add(newContext)
	contextDB := models.ToContextDB(req)
	err = db.CreateContext(contextDB)
	if err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, nil
}
