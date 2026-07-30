package rpc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/implanttypes"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"google.golang.org/protobuf/proto"
)

func waitForCtrlStatus(action, name string, status *clientpb.JobStatus) error {
	if status == nil {
		return fmt.Errorf("%s %s failed: timeout waiting for listener response", action, name)
	}
	if status.Status == consts.CtrlStatusSuccess {
		return nil
	}
	if status.Error != "" {
		return fmt.Errorf("%s %s failed: %s", action, name, status.Error)
	}
	return fmt.Errorf("%s %s failed: unknown error", action, name)
}

func publishPipelineLifecycleEvent(operation string, pipeline *clientpb.Pipeline) {
	if pipeline == nil {
		return
	}
	core.EventBroker.Publish(core.Event{
		EventType: consts.EventJob,
		Op:        operation,
		Job: &clientpb.Job{
			Name:     pipeline.Name,
			Pipeline: pipeline,
		},
		Important: true,
	})
}

func lockPipelineLifecycle(listenerID, name string) func() {
	value, _ := pipelineLifecycleLocks.LoadOrStore(listenerID+":"+name, &sync.Mutex{})
	lock := value.(*sync.Mutex)
	lock.Lock()
	return lock.Unlock
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
	if req == nil {
		return nil, types.ErrMissingRequestField
	}
	if err := validatePipelineIdentity(req); err != nil {
		return nil, err
	}
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
	if err := registerDefaultProfileForPipeline(req); err != nil {
		logs.Log.Errorf("new profile %s failed %v", req.Name, err)
	}
	publishPipelineLifecycleEvent(consts.CtrlPipelineRegister, req)
	return &clientpb.Empty{}, nil
}

func registerDefaultProfileForPipeline(req *clientpb.Pipeline) error {
	if req.GetName() == "" {
		return nil
	}
	profileName := req.Name + "_default"
	pipelines, err := db.NewPipelineQuery().WhereName(req.Name).Find()
	if err != nil {
		return err
	}
	if len(pipelines) > 1 {
		profileName = req.ListenerId + "_" + req.Name + "_default"
	}
	pipelineID := req.Name
	if req.ListenerId != "" {
		pipelineID = req.ListenerId + ":" + req.Name
	}
	return db.NewProfile(&clientpb.Profile{
		Name:       profileName,
		PipelineId: pipelineID,
	})
}

func (rpc *Server) SyncPipeline(ctx context.Context, req *clientpb.Pipeline) (*clientpb.Empty, error) {
	if req == nil {
		return nil, types.ErrMissingRequestField
	}
	if err := validatePipelineIdentity(req); err != nil {
		return nil, err
	}
	_, err := db.SavePipeline(models.FromPipelinePb(req))
	if err != nil {
		return nil, err
	}
	if req.GetRem() != nil {
		if err := configs.SyncREMConfigFromPipeline(req); err != nil {
			return nil, err
		}
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

type pipelineTLSSnapshot struct {
	certName string
	tls      *clientpb.TLS
	enabled  bool
}

type preparedPipelineTLS struct {
	certName string
	tls      *clientpb.TLS
	saveName string
	comment  string
}

func (rpc *Server) UpdatePipelineTLS(ctx context.Context, req *clientpb.PipelineTLSUpdate) (*clientpb.Pipeline, error) {
	if req == nil {
		return nil, types.ErrMissingRequestField
	}
	if strings.TrimSpace(req.GetName()) == "" {
		return nil, fmt.Errorf("pipeline name is required")
	}
	listenerID, err := resolveListenerID(&clientpb.CtrlPipeline{
		Name:       req.GetName(),
		ListenerId: req.GetListenerId(),
	})
	if err != nil {
		return nil, err
	}
	unlock := lockPipelineLifecycle(listenerID, req.GetName())
	defer unlock()
	return rpc.updatePipelineTLSLocked(ctx, req, listenerID)
}

func (rpc *Server) updatePipelineTLSLocked(ctx context.Context, req *clientpb.PipelineTLSUpdate, listenerID string) (*clientpb.Pipeline, error) {
	pipeline, err := db.FindPipelineByListener(req.GetName(), listenerID)
	if err != nil {
		return nil, err
	}
	if pipeline.Type != consts.HTTPPipeline && pipeline.Type != consts.TCPPipeline {
		return nil, fmt.Errorf("pipeline %s/%s type %s does not support TLS certificate binding", listenerID, pipeline.Name, pipeline.Type)
	}

	prepared, err := preparePipelineTLSUpdate(pipeline, req)
	if err != nil {
		return nil, err
	}
	snapshot := pipelineTLSSnapshot{
		certName: pipeline.CertName,
		tls:      cloneTLS(pipeline.ToProtobuf().GetTls()),
		enabled:  pipeline.Enable,
	}

	running := false
	if listener, listenerErr := core.Listeners.Get(listenerID); listenerErr == nil {
		runtime := listener.GetPipeline(pipeline.Name)
		running = runtime != nil && runtime.GetEnable()
	}
	if running {
		if _, err := rpc.stopPipelineLocked(ctx, &clientpb.CtrlPipeline{Name: pipeline.Name, ListenerId: listenerID}, listenerID); err != nil {
			return nil, err
		}
	}

	pipelineName := pipeline.Name
	pipeline, err = db.FindPipelineByListener(pipeline.Name, listenerID)
	if err != nil {
		return nil, rpc.rollbackPipelineTLS(ctx, &models.Pipeline{Name: pipelineName}, listenerID, snapshot, running, nil, err)
	}
	updated, createdCert, err := persistPreparedPipelineTLS(pipeline, prepared)
	if err != nil {
		return nil, rpc.rollbackPipelineTLS(ctx, pipeline, listenerID, snapshot, running, createdCert, err)
	}

	if running {
		if _, err := rpc.startPipelineLocked(ctx, &clientpb.CtrlPipeline{Name: updated.Name, ListenerId: listenerID}, listenerID); err != nil {
			return nil, rpc.rollbackPipelineTLS(ctx, updated, listenerID, snapshot, true, createdCert, err)
		}
	}

	updated, err = db.FindPipelineByListener(updated.Name, listenerID)
	if err != nil {
		return nil, err
	}
	if createdCert != nil {
		publishCertificateLifecycleEvent(consts.CtrlCertCreate, createdCert.Name, "")
	}
	pb := updated.ToProtobuf()
	publishPipelineLifecycleEvent(consts.CtrlPipelineSync, pb)
	return pb, nil
}

func preparePipelineTLSUpdate(pipeline *models.Pipeline, req *clientpb.PipelineTLSUpdate) (*preparedPipelineTLS, error) {
	switch req.GetMode() {
	case clientpb.TLSUpdateMode_TLS_UPDATE_MODE_DISABLE:
		return &preparedPipelineTLS{tls: &clientpb.TLS{Enable: false}}, nil
	case clientpb.TLSUpdateMode_TLS_UPDATE_MODE_EXISTING_CERT:
		certName := strings.TrimSpace(req.GetCertName())
		if certName == "" {
			return nil, fmt.Errorf("cert_name is required")
		}
		cert, err := db.FindCertificate(certName)
		if err != nil {
			return nil, err
		}
		tlsConfig := cert.ToProtobuf()
		if tlsConfig == nil {
			return nil, fmt.Errorf("certificate %s is invalid", certName)
		}
		if _, err := prepareWebsiteTLS(pipeline.Name, tlsConfig); err != nil {
			return nil, fmt.Errorf("certificate %s is invalid: %w", certName, err)
		}
		return &preparedPipelineTLS{certName: certName, tls: tlsConfig}, nil
	case clientpb.TLSUpdateMode_TLS_UPDATE_MODE_INLINE_CERT:
		tlsConfig := cloneTLS(req.GetTls())
		if tlsConfig == nil {
			tlsConfig = &clientpb.TLS{}
		}
		tlsConfig.Enable = true
		prepared, err := prepareWebsiteTLS(pipeline.Name, tlsConfig)
		if err != nil {
			return nil, err
		}
		result := &preparedPipelineTLS{tls: prepared, comment: req.GetCertComment()}
		if req.GetSaveCert() {
			result.saveName = strings.TrimSpace(req.GetSaveCertName())
			if result.saveName == "" {
				return nil, fmt.Errorf("save_cert_name is required when save_cert is enabled")
			}
		}
		return result, nil
	default:
		return nil, fmt.Errorf("tls update mode is required")
	}
}

func persistPreparedPipelineTLS(pipeline *models.Pipeline, prepared *preparedPipelineTLS) (*models.Pipeline, *models.Certificate, error) {
	if prepared.saveName != "" {
		certModel, err := db.SaveCertFromTLSWithOptions(prepared.tls, "", "", db.SaveCertOptions{
			Name:    prepared.saveName,
			Comment: prepared.comment,
		})
		if err != nil {
			return nil, nil, err
		}
		updated, err := db.UpdatePipelineCert(certModel.Name, pipeline)
		return updated, certModel, err
	}
	if prepared.certName != "" {
		updated, err := db.UpdatePipelineCert(prepared.certName, pipeline)
		return updated, nil, err
	}
	tlsConfig := cloneTLS(prepared.tls)
	if tlsConfig.GetCert() != nil {
		tlsConfig.Cert.Name = ""
		tlsConfig.Cert.Comment = prepared.comment
	}
	updated, err := db.SetPipelineTLS(pipeline, tlsConfig, "")
	return updated, nil, err
}

func (rpc *Server) rollbackPipelineTLS(ctx context.Context, pipeline *models.Pipeline, listenerID string, snapshot pipelineTLSSnapshot, wasRunning bool, createdCert *models.Certificate, cause error) error {
	rollbackErrors := []error{cause}
	name := ""
	if pipeline != nil {
		name = pipeline.Name
	}
	if wasRunning && name != "" {
		if listener, listenerErr := core.Listeners.Get(listenerID); listenerErr == nil {
			if runtime := listener.GetPipeline(name); runtime != nil && runtime.GetEnable() {
				if _, cleanupErr := rpc.stopPipelineLocked(ctx, &clientpb.CtrlPipeline{Name: name, ListenerId: listenerID}, listenerID); cleanupErr != nil {
					rollbackErrors = append(rollbackErrors, fmt.Errorf("stop partially started pipeline: %w", cleanupErr))
				}
			}
		}
	}
	if name == "" {
		rollbackErrors = append(rollbackErrors, fmt.Errorf("rollback failed: pipeline identity is unavailable"))
	} else if restoreErr := restorePipelineTLSSnapshot(name, listenerID, snapshot); restoreErr != nil {
		rollbackErrors = append(rollbackErrors, fmt.Errorf("restore previous TLS configuration: %w", restoreErr))
	} else if wasRunning {
		if _, restartErr := rpc.startPipelineLocked(ctx, &clientpb.CtrlPipeline{Name: name, ListenerId: listenerID}, listenerID); restartErr != nil {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("restart pipeline with previous TLS configuration: %w", restartErr))
		}
	}
	if createdCert != nil {
		if deleteErr := db.DeleteCertificate(createdCert.Name); deleteErr != nil {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("remove generated certificate %s: %w", createdCert.Name, deleteErr))
		}
	}
	return errors.Join(rollbackErrors...)
}

func restorePipelineTLSSnapshot(name, listenerID string, snapshot pipelineTLSSnapshot) error {
	pipeline, err := db.FindPipelineByListener(name, listenerID)
	if err != nil {
		return err
	}
	if snapshot.certName != "" {
		if _, err := db.UpdatePipelineCert(snapshot.certName, pipeline); err != nil {
			return err
		}
	} else {
		tlsConfig := cloneTLS(snapshot.tls)
		if tlsConfig == nil {
			tlsConfig = &clientpb.TLS{Enable: false}
		}
		if _, err := db.SetPipelineTLS(pipeline, tlsConfig, ""); err != nil {
			return err
		}
	}
	if snapshot.enabled {
		return db.EnablePipelineByListener(name, listenerID)
	}
	return db.DisablePipelineByListener(name, listenerID)
}

func cloneTLS(tlsConfig *clientpb.TLS) *clientpb.TLS {
	if tlsConfig == nil {
		return nil
	}
	return proto.Clone(tlsConfig).(*clientpb.TLS)
}

func (rpc *Server) ListPipelines(ctx context.Context, req *clientpb.Listener) (*clientpb.Pipelines, error) {
	if req == nil {
		return nil, types.ErrMissingRequestField
	}
	pipelines, err := db.ListPipelinesByListener(req.Id)
	if err != nil {
		return nil, err
	}
	result := pipelines.ToProtobuf()
	for _, pipeline := range result.GetPipelines() {
		runtime, ok := core.Listeners.FindByListener(pipeline.GetListenerId(), pipeline.GetName())
		pipeline.Enable = pipeline.GetEnable() && ok && runtime.GetEnable()
	}
	return result, nil
}

func (rpc *Server) StartPipeline(ctx context.Context, req *clientpb.CtrlPipeline) (*clientpb.Empty, error) {
	if req == nil {
		return nil, types.ErrMissingRequestField
	}
	listenerID, err := resolveListenerID(req)
	if err != nil {
		return nil, err
	}
	unlock := lockPipelineLifecycle(listenerID, req.GetName())
	defer unlock()
	return rpc.startPipelineLocked(ctx, req, listenerID)
}

func (rpc *Server) startPipelineLocked(ctx context.Context, req *clientpb.CtrlPipeline, listenerID string) (*clientpb.Empty, error) {
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
	certificateHandled := false
	if req.GetCertName() != "" && (pipelineDB.Type == consts.HTTPPipeline || pipelineDB.Type == consts.TCPPipeline) {
		running := false
		if listener, listenerErr := core.Listeners.Get(listenerID); listenerErr == nil {
			if runtime := listener.GetPipeline(req.Name); runtime != nil && runtime.GetEnable() {
				running = true
			}
		}
		_, updateErr := rpc.updatePipelineTLSLocked(ctx, &clientpb.PipelineTLSUpdate{
			ListenerId: listenerID,
			Name:       req.Name,
			Mode:       clientpb.TLSUpdateMode_TLS_UPDATE_MODE_EXISTING_CERT,
			CertName:   req.CertName,
		}, listenerID)
		if updateErr != nil {
			return nil, updateErr
		}
		if running {
			return &clientpb.Empty{}, nil
		}
		pipelineDB, err = db.FindPipelineByListener(req.Name, listenerID)
		if err != nil {
			return nil, err
		}
		certificateHandled = true
	}

	if pipelineDB.PipelineParams == nil {
		pipelineDB.PipelineParams = &implanttypes.PipelineParams{}
	}
	if req.CertName != "" {
		if !certificateHandled {
			_, err := db.FindCertificate(req.CertName)
			if err != nil {
				return nil, err
			}
			pipelineDB, err = db.UpdatePipelineCert(req.CertName, pipelineDB)
			if err != nil {
				return nil, err
			}
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
		if err := db.EnablePipelineByListener(req.Name, listenerID); err != nil {
			return nil, err
		}
		existing.Enable = true
		publishPipelineLifecycleEvent(consts.CtrlPipelineStart, existing)
		return &clientpb.Empty{}, nil
	}

	pipelineProto := pipelineDB.ToProtobuf()
	job := &core.Job{
		ID:       core.NextJobID(),
		Pipeline: pipelineProto,
		Name:     req.Name,
	}

	ctrlID := lns.PushCtrlDeferredEvent(ctx, &clientpb.JobCtrl{
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
	pipeline.Enable = true
	core.Jobs.AddPipeline(pipeline)
	publishPipelineLifecycleEvent(consts.CtrlPipelineStart, pipeline)
	return &clientpb.Empty{}, nil
}

func (rpc *Server) StopPipeline(ctx context.Context, req *clientpb.CtrlPipeline) (*clientpb.Empty, error) {
	if req == nil {
		return nil, types.ErrMissingRequestField
	}
	listenerID, err := resolveListenerID(req)
	if err != nil {
		return nil, err
	}
	unlock := lockPipelineLifecycle(listenerID, req.GetName())
	defer unlock()
	return rpc.stopPipelineLocked(ctx, req, listenerID)
}

func (rpc *Server) stopPipelineLocked(ctx context.Context, req *clientpb.CtrlPipeline, listenerID string) (*clientpb.Empty, error) {
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
		ctrlID := lns.PushCtrlDeferredEvent(ctx, &clientpb.JobCtrl{
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
	persisted := pipelineDB.ToProtobuf()
	persisted.Enable = false
	publishPipelineLifecycleEvent(consts.CtrlPipelineStop, persisted)
	return &clientpb.Empty{}, nil
}

func (rpc *Server) DeletePipeline(ctx context.Context, req *clientpb.CtrlPipeline) (*clientpb.Empty, error) {
	if req == nil {
		return nil, types.ErrMissingRequestField
	}
	listenerID, err := resolveListenerID(req)
	if err != nil {
		return nil, err
	}
	unlock := lockPipelineLifecycle(listenerID, req.GetName())
	defer unlock()
	return rpc.deletePipelineLocked(ctx, req, listenerID)
}

func (rpc *Server) deletePipelineLocked(ctx context.Context, req *clientpb.CtrlPipeline, listenerID string) (*clientpb.Empty, error) {
	pipelineDB, err := db.FindPipelineByListener(req.Name, listenerID)
	if err != nil {
		return nil, err
	}

	// Delegate REM pipelines to their dedicated handler.
	if pipelineDB.Type == consts.RemPipeline {
		req.ListenerId = listenerID
		return rpc.DeleteRem(ctx, req)
	}

	if lns, runtimeErr := core.Listeners.Get(listenerID); runtimeErr == nil {
		if pipe := lns.GetPipeline(req.Name); pipe != nil {
			ctrlID := lns.PushCtrlDeferredEvent(ctx, &clientpb.JobCtrl{
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
	}

	err = db.DeletePipelineByListener(pipelineDB.Name, listenerID)
	if err != nil {
		return nil, err
	}
	publishPipelineLifecycleEvent(consts.CtrlPipelineDelete, pipelineDB.ToProtobuf())
	return &clientpb.Empty{}, nil
}
