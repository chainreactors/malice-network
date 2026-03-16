package rpc

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	errs "github.com/chainreactors/IoM-go/types"

	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"google.golang.org/protobuf/proto"
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
		query.ByTask(fmt.Sprintf("%s-%d", req.Task.SessionId, req.Task.TaskId))
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
	if req == nil || req.Task == nil {
		return nil, nil
	}

	sessionID := ""
	if req.Session != nil {
		sessionID = req.Session.SessionId
	}
	if sessionID == "" {
		sessionID = req.Task.SessionId
	}
	if sessionID == "" {
		return nil, errors.New("task session id is required")
	}

	sess, err := core.Sessions.Get(sessionID)
	if err != nil {
		return nil, err
	}

	task := sess.Tasks.Get(req.Task.TaskId)
	if task == nil {
		return nil, errs.ErrNotFoundTask
	}

	return task, nil
}

func bindResolvedTask(req *clientpb.Context, task *core.Task) *clientpb.Context {
	if req == nil || task == nil || task.Session == nil {
		return req
	}

	clone := proto.Clone(req).(*clientpb.Context)
	clone.Task = task.ToProtobuf()
	clone.Session = task.Session.ToProtobufLite()
	return clone
}

func (rpc *Server) AddScreenShot(ctx context.Context, req *clientpb.Context) (*clientpb.Empty, error) {
	task, err := getTaskFromContext(req)
	if err != nil {
		return nil, err
	}

	content := req.Content
	if len(content) == 0 {
		screenshot, err := output.NewScreenShot(req.Value)
		if err != nil {
			return nil, err
		}
		content = screenshot.Content
	}

	err = core.HandleScreenshot(content, task)
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

func (rpc *Server) AddUpload(ctx context.Context, req *clientpb.Context) (*clientpb.Empty, error) {
	ictx, err := db.SaveContext(req)
	if err != nil {
		return nil, err
	}
	core.PushContextEvent(consts.ContextUpload, ictx)
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
	_ = download

	_, err = db.SaveContext(bindResolvedTask(req, task))
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
	_ = cred

	dctx, err := db.SaveContext(bindResolvedTask(req, task))
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
	_ = port

	dctx, err := db.SaveContext(bindResolvedTask(req, task))
	if err != nil {
		return nil, err
	}
	core.PushContextEvent(consts.CtrlContextPort, dctx)
	return &clientpb.Empty{}, nil
}

func (rpc *Server) AddKeylogger(ctx context.Context, req *clientpb.Context) (*clientpb.Empty, error) {
	task, err := getTaskFromContext(req)
	if err != nil {
		return nil, err
	}

	content := req.Content
	if len(content) == 0 {
		content = req.Value
	}
	if err := core.HandleKeylogger(content, task, "", "", req.Nonce); err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, nil
}

func (rpc *Server) DeleteContext(ctx context.Context, req *clientpb.Context) (*clientpb.Empty, error) {
	if req.Id == "" {
		return nil, fmt.Errorf("context id is required")
	}

	ictx, err := db.FindContext(req.Id)
	if err != nil {
		return nil, fmt.Errorf("context not found: %w", err)
	}

	// delete associated file if exists
	if ictx.Context != nil {
		switch c := ictx.Context.(type) {
		case *output.ScreenShotContext:
			os.Remove(c.FilePath)
		case *output.DownloadContext:
			os.Remove(c.FilePath)
		case *output.KeyLoggerContext:
			os.Remove(c.FilePath)
		case *output.UploadContext:
			os.Remove(c.FilePath)
		case *output.MediaContext:
			os.Remove(c.FilePath)
		}
	}

	if err := db.DeleteContext(ictx.ID.String()); err != nil {
		return nil, fmt.Errorf("failed to delete context: %w", err)
	}

	return &clientpb.Empty{}, nil
}

func (rpc *Server) Sync(ctx context.Context, req *clientpb.Sync) (*clientpb.Context, error) {
	if req.TaskId == "" && req.ContextId == "" {
		return nil, fmt.Errorf("context id or task id is required")
	}

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

	data, err := core.ReadFileForContext(ictx.Context)
	if err != nil {
		return ictx.ToProtobuf(), nil
	} else {
		result := ictx.ToProtobuf()
		result.Content = data
		return result, nil
	}
}
