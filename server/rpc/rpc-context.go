package rpc

import (
	"context"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/errs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/server/internal/core"
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

func (rpc *Server) GetScreenShots(ctx context.Context, req *clientpb.Empty) (*clientpb.Contexts, error) {
	contexts, err := db.GetContextsByType(consts.ContextScreenShot)
	if err != nil {
		return nil, err
	}
	return &clientpb.Contexts{Contexts: contexts}, nil
}

func (rpc *Server) GetCredentials(ctx context.Context, req *clientpb.Empty) (*clientpb.Contexts, error) {
	contexts, err := db.GetContextsByType(consts.ContextCredential)
	if err != nil {
		return nil, err
	}
	return &clientpb.Contexts{Contexts: contexts}, nil
}

func (rpc *Server) GetKeyloggers(ctx context.Context, req *clientpb.Empty) (*clientpb.Contexts, error) {
	contexts, err := db.GetContextsByType(consts.ContextKeyLogger)
	if err != nil {
		return nil, err
	}
	return &clientpb.Contexts{Contexts: contexts}, nil
}

func (rpc *Server) AddScreenShot(ctx context.Context, req *clientpb.Context) (*clientpb.Empty, error) {
	var sess *core.Session
	var task *core.Task
	var err error
	if req.Session != nil {
		sess, err = core.Sessions.Get(req.Session.SessionId)
		if err != nil {
			return nil, err
		}
		if req.Task != nil {
			task = sess.Tasks.Get(req.Task.TaskId)
			if task == nil {
				return nil, errs.ErrNotFoundTask
			}
		}
	}
	screenshot, err := types.NewScreenShot([]byte(req.Value))
	if err != nil {
		return nil, err
	}
	err = core.HandleScreenshot(screenshot.Content, task)
	if err != nil {
		return nil, err
	}

	return &clientpb.Empty{}, nil
}

func (rpc *Server) AddContext(ctx context.Context, req *clientpb.Context) (*clientpb.Empty, error) {
	_, err := db.SaveContext(req)
	if err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, nil
}

func (rpc *Server) Sync(ctx context.Context, req *clientpb.Sync) (*clientpb.Context, error) {
	ictx, err := db.FindContext(req.ContextId)
	if err != nil {
		return nil, err
	}
	//if !file.Exist(td.Path + td.Name) {
	//	return nil, os.ErrExist
	//}

	return ictx.ToProtobuf(), nil
}
