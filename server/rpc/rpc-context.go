package rpc

import (
	"context"
	"fmt"

	"github.com/chainreactors/malice-network/helper/consts"
	errs "github.com/chainreactors/malice-network/helper/errs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
)

func (rpc *Server) GetContexts(ctx context.Context, req *clientpb.Context) (*clientpb.Contexts, error) {
	query := db.NewContextQuery()

	if req.Type != "" {
		query.ByType(req.Type)
	}
	if req.Session != nil {
		query.BySession(req.Session.SessionId)
	}
	if req.Task != nil {
		query.ByTask(fmt.Sprintf("%s_%d", req.Task.SessionId, req.Task.TaskId))
	}
	if req.Pipeline != nil {
		query.ByPipeline(req.Pipeline.Name)
	}
	if req.Nonce != "" {
		query.ByNonce(req.Nonce)
	}

	contexts, err := query.Find()
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

// getTaskFromContext 从Context请求中获取Session和Task
func getTaskFromContext(req *clientpb.Context) (*core.Task, error) {
	if req.Session == nil {
		return nil, nil
	}

	sess, err := core.Sessions.Get(req.Session.SessionId)
	if err != nil {
		return nil, err
	}

	if req.Task == nil {
		return nil, nil
	}

	task := sess.Tasks.Get(req.Task.TaskId)
	if task == nil {
		return nil, errs.ErrNotFoundTask
	}

	return task, nil
}

func (rpc *Server) AddScreenShot(ctx context.Context, req *clientpb.Context) (*clientpb.Empty, error) {
	task, err := getTaskFromContext(req)
	if err != nil {
		return nil, err
	}

	screenshot, err := output.NewScreenShot(req.Value)
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

func (rpc *Server) AddDownload(ctx context.Context, req *clientpb.Context) (*clientpb.Empty, error) {
	task, err := getTaskFromContext(req)
	if err != nil {
		return nil, err
	}

	download, err := output.NewDownloadContext(req.Value)
	if err != nil {
		return nil, err
	}

	_, err = core.SaveContext(download, task)
	if err != nil {
		return nil, err
	}

	return &clientpb.Empty{}, nil
}

func (rpc *Server) AddCredential(ctx context.Context, req *clientpb.Context) (*clientpb.Empty, error) {
	task, err := getTaskFromContext(req)
	if err != nil {
		return nil, err
	}

	cred, err := output.NewCredential(req.Value)
	if err != nil {
		return nil, err
	}

	dctx, err := core.SaveContext(cred, task)
	if err != nil {
		return nil, err
	}
	core.PushContextEvent(consts.ContextCredential, dctx)
	return &clientpb.Empty{}, nil
}

func (rpc *Server) AddPort(ctx context.Context, req *clientpb.Context) (*clientpb.Empty, error) {
	task, err := getTaskFromContext(req)
	if err != nil {
		return nil, err
	}

	port, err := output.NewPortContext(req.Value)
	if err != nil {
		return nil, err
	}

	_, err = core.SaveContext(port, task)
	if err != nil {
		return nil, err
	}
	dctx, err := core.SaveContext(port, task)
	if err != nil {
		return nil, err
	}
	core.PushContextEvent(consts.CtrlContextCred, dctx)
	return &clientpb.Empty{}, nil
}

func (rpc *Server) Sync(ctx context.Context, req *clientpb.Sync) (*clientpb.Context, error) {
	var ictx *models.Context
	var err error
	if req.TaskId != "" {
		ictx, err = db.GetContextByTask(req.TaskId)
	} else {
		ictx, err = db.FindContext(req.ContextId)
	}
	if err != nil {
		return nil, err
	}

	c, err := output.ParseContext(ictx.Type, ictx.Value)
	if err != nil {
		return nil, err
	}
	data, err := core.ReadFileForContext(c)
	if err != nil {
		return ictx.ToProtobuf(), nil
	} else {
		result := ictx.ToProtobuf()
		result.Content = data
		return result, nil
	}
}
