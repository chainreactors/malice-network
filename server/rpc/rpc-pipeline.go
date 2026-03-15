package rpc

import (
	"context"
	"fmt"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/implanttypes"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
)

func waitForCtrlStatus(action, name string, status *clientpb.JobStatus) error {
	if status == nil {
		return fmt.Errorf("%s %s failed: unknown error", action, name)
	}
	if status.Status == consts.CtrlStatusSuccess {
		return nil
	}
	if status.Error != "" {
		return fmt.Errorf("%s %s failed: %s", action, name, status.Error)
	}
	return fmt.Errorf("%s %s failed: unknown error", action, name)
}

// resolveListenerID resolves the listener ID from a CtrlPipeline request.
// If listener_id is not provided, it queries the database by pipeline name.
// Returns an error if no pipeline is found, or if multiple pipelines share the same name.
func resolveListenerID(req *clientpb.CtrlPipeline) (string, error) {
	listenerID := req.GetListenerId()
	if listenerID == "" && req.Pipeline != nil {
		listenerID = req.Pipeline.ListenerId
	}
	if listenerID != "" {
		return listenerID, nil
	}

	// No listener_id provided, try to resolve by pipeline name
	if req.Name == "" {
		return "", fmt.Errorf("pipeline name required")
	}
	pipelines, err := db.NewPipelineQuery().WhereName(req.Name).Find()
	if err != nil {
		return "", err
	}
	switch len(pipelines) {
	case 0:
		return "", fmt.Errorf("pipeline '%s' not found", req.Name)
	case 1:
		return pipelines[0].ListenerId, nil
	default:
		return "", fmt.Errorf("multiple pipelines named '%s' found across listeners, please specify listener_id", req.Name)
	}
}

func (rpc *Server) RegisterPipeline(ctx context.Context, req *clientpb.Pipeline) (*clientpb.Empty, error) {
	lns, err := core.Listeners.Get(req.ListenerId)
	if err != nil {
		return nil, err
	}
	req.Ip = lns.IP
	pipelineModel := models.FromPipelinePb(req)
	_, err = db.SavePipeline(pipelineModel)
	if err != nil {
		return nil, err
	}
	profileReq := &clientpb.Profile{
		Name:       req.Name + "_default",
		PipelineId: req.Name,
	}
	err = db.NewProfile(profileReq)
	if err != nil {
		logs.Log.Errorf("new profile %s failed %v", req.Name, err)
	}
	return &clientpb.Empty{}, nil
}

func (rpc *Server) SyncPipeline(ctx context.Context, req *clientpb.Pipeline) (*clientpb.Empty, error) {
	_, err := db.SavePipeline(models.FromPipelinePb(req))
	if err != nil {
		return nil, err
	}
	job := core.Jobs.AddPipeline(req)
	core.EventBroker.Publish(core.Event{
		EventType: consts.EventJob,
		Op:        consts.CtrlPipelineSync,
		Important: true,
		Job:       job.ToProtobuf(),
	})
	return &clientpb.Empty{}, nil
}

func (rpc *Server) ListPipelines(ctx context.Context, req *clientpb.Listener) (*clientpb.Pipelines, error) {
	pipelines, err := db.ListPipelinesByListener(req.Id)
	if err != nil {
		return nil, err
	}
	return pipelines.ToProtobuf(), nil
}

func (rpc *Server) StartPipeline(ctx context.Context, req *clientpb.CtrlPipeline) (*clientpb.Empty, error) {
	listenerID, err := resolveListenerID(req)
	if err != nil {
		return nil, err
	}

	pipelineDB, err := db.FindPipelineByListener(req.Name, listenerID)
	if err != nil {
		return nil, err
	}

	// REM pipelines have their own start path; delegate transparently so
	// callers (e.g. WebUI) that always use StartPipeline still work.
	if pipelineDB.Type == consts.RemPipeline {
		req.ListenerId = listenerID
		return rpc.StartRem(ctx, req)
	}

	if pipelineDB.PipelineParams == nil {
		pipelineDB.PipelineParams = &implanttypes.PipelineParams{}
	}
	if req.CertName != "" {
		_, err := db.FindCertificate(req.CertName)
		if err != nil {
			return nil, err
		}
		pipelineDB, err = db.UpdatePipelineCert(req.CertName, pipelineDB)
		if err != nil {
			return nil, err
		}
	} else if req.Pipeline != nil && req.Pipeline.Tls != nil {
		if req.Pipeline.Tls.Cert != nil && req.Pipeline.Tls.Cert.Cert != "" && req.Pipeline.Tls.Cert.Key != "" {
			pipelineDB.PipelineParams.Tls = implanttypes.FromTls(req.Pipeline.Tls)
		}
	}
	lns, err := core.Listeners.Get(listenerID)
	if err != nil {
		return nil, err
	}

	if existing := lns.GetPipeline(req.Name); existing != nil && existing.Enable {
		_ = db.EnablePipelineByListener(req.Name, listenerID)
		return &clientpb.Empty{}, nil
	}

	pipelineProto := pipelineDB.ToProtobuf()
	job := &core.Job{
		ID:       core.NextJobID(),
		Pipeline: pipelineProto,
		Name:     req.Name,
	}

	ctrlID := lns.PushCtrl(&clientpb.JobCtrl{
		Ctrl: consts.CtrlPipelineStart,
		Job:  job.ToProtobuf()})

	status := lns.WaitCtrl(ctrlID)
	if err := waitForCtrlStatus("start pipeline", req.Name, status); err != nil {
		_ = db.DisablePipelineByListener(pipelineDB.Name, listenerID)
		return nil, err
	}

	pipeline := pipelineDB.ToProtobuf()
	if err := db.EnablePipelineByListener(pipeline.Name, listenerID); err != nil {
		return nil, err
	}
	core.Jobs.AddPipeline(pipeline)
	return &clientpb.Empty{}, nil
}

func (rpc *Server) StopPipeline(ctx context.Context, req *clientpb.CtrlPipeline) (*clientpb.Empty, error) {
	listenerID, err := resolveListenerID(req)
	if err != nil {
		return nil, err
	}

	pipelineDB, err := db.FindPipelineByListener(req.Name, listenerID)
	if err != nil {
		return nil, err
	}

	// Delegate REM pipelines to their dedicated handler.
	if pipelineDB.Type == consts.RemPipeline {
		req.ListenerId = listenerID
		return rpc.StopRem(ctx, req)
	}

	lns, err := core.Listeners.Get(listenerID)
	if err != nil {
		return nil, err
	}

	if _, err := db.FindPipelineByListener(req.Name, listenerID); err != nil {
		return nil, err
	}

	pipe := lns.GetPipeline(req.Name)
	if pipe != nil {
		job := &core.Job{
			ID:       core.NextJobID(),
			Name:     req.Name,
			Pipeline: pipe,
		}
		ctrlID := lns.PushCtrl(&clientpb.JobCtrl{
			Ctrl: consts.CtrlPipelineStop,
			Job:  job.ToProtobuf(),
		})
		status := lns.WaitCtrl(ctrlID)
		if err := waitForCtrlStatus("stop pipeline", req.Name, status); err != nil {
			return nil, err
		}
	}

	if err := db.DisablePipelineByListener(req.Name, listenerID); err != nil {
		return nil, err
	}
	if pipe != nil {
		lns.RemovePipeline(pipe)
	}
	return &clientpb.Empty{}, nil
}

func (rpc *Server) DeletePipeline(ctx context.Context, req *clientpb.CtrlPipeline) (*clientpb.Empty, error) {
	listenerID, err := resolveListenerID(req)
	if err != nil {
		return nil, err
	}

	pipelineDB, err := db.FindPipelineByListener(req.Name, listenerID)
	if err != nil {
		return nil, err
	}

	// Delegate REM pipelines to their dedicated handler.
	if pipelineDB.Type == consts.RemPipeline {
		req.ListenerId = listenerID
		return rpc.DeleteRem(ctx, req)
	}

	lns, err := core.Listeners.Get(listenerID)
	if err != nil {
		return nil, err
	}

	if pipe := lns.GetPipeline(req.Name); pipe != nil {
		ctrlID := lns.PushCtrl(&clientpb.JobCtrl{
			Ctrl: consts.CtrlPipelineStop,
			Job: &clientpb.Job{
				Id:       core.NextJobID(),
				Name:     req.Name,
				Pipeline: pipe,
			},
		})
		status := lns.WaitCtrl(ctrlID)
		if err := waitForCtrlStatus("delete pipeline", req.Name, status); err != nil {
			return nil, err
		}
		lns.RemovePipeline(pipe)
	}

	err = db.DeletePipelineByListener(pipelineDB.Name, listenerID)
	if err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, nil
}
