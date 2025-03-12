package listener

import (
	"context"
	"errors"
	"fmt"
	"github.com/chainreactors/malice-network/helper/errs"
	"github.com/chainreactors/rem/agent"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"os"
	"path/filepath"
	"strings"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/codenames"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/helper/utils/mtls"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
)

var (
	Listener *listener
)

func NewListener(clientConf *mtls.ClientConfig, cfg *configs.ListenerConfig) error {
	options, err := mtls.GetGrpcOptions([]byte(clientConf.CACertificate), []byte(clientConf.Certificate), []byte(clientConf.PrivateKey), clientConf.Type)
	if err != nil {
		return err
	}
	listenerCfg, err := mtls.ReadConfig(cfg.Auth)
	if err != nil {
		return err
	}
	conn, err := grpc.Dial(listenerCfg.Address(), options...)
	if err != nil {
		return err
	}

	lns := &listener{
		Rpc:       listenerrpc.NewListenerRPCClient(conn),
		Name:      cfg.Name,
		IP:        cfg.IP,
		pipelines: make(core.Pipelines),
		conn:      conn,
		cfg:       cfg,
		websites:  make(map[string]*Website),
	}

	_, err = lns.Rpc.RegisterListener(lns.Context(), &clientpb.RegisterListener{
		Name: lns.Name,
		Host: cfg.IP,
	})
	if err != nil {
		return err
	}
	go lns.Handler()
	Listener = lns

	for _, tcpPipeline := range cfg.TcpPipelines {
		pipeline, err := tcpPipeline.ToProtobuf(lns.Name)
		if err != nil {
			return err
		}
		err = lns.RegisterAndStart(pipeline)
		if err != nil {
			return err
		}
	}

	for _, bindPipeline := range cfg.BindPipelineConfig {
		pipeline, err := bindPipeline.ToProtobuf(lns.Name)
		if err != nil {
			return err
		}
		err = lns.RegisterAndStart(pipeline)
		if err != nil {
			return err
		}
	}

	for _, rem := range cfg.REMs {
		if !rem.Enable {
			continue
		}
		pipeline, err := rem.ToProtobuf(lns.Name)
		if err != nil {
			return err
		}

		_, err = lns.Rpc.RegisterRem(lns.Context(), pipeline)
		if err != nil {
			return err
		}

		_, err = lns.Rpc.StartRem(lns.Context(), &clientpb.CtrlPipeline{
			Name:       pipeline.Name,
			ListenerId: lns.ID(),
		})
		if err != nil {
			return err
		}
	}

	for _, newWebsite := range cfg.Websites {
		if !newWebsite.Enable {
			continue
		}
		tls, err := newWebsite.TlsConfig.ReadCert()
		if err != nil {
			return err
		}

		web := &clientpb.Website{
			Root: newWebsite.RootPath,
			Port: uint32(newWebsite.Port),
		}
		pipe := &clientpb.Pipeline{
			Name:       newWebsite.WebsiteName,
			ListenerId: lns.Name,
			Body: &clientpb.Pipeline_Web{
				Web: web,
			},
			Tls: tls.ToProtobuf(),
		}

		contents := map[string]*clientpb.WebContent{}
		for _, content := range newWebsite.WebContents {
			contents[content.Path], err = content.ToProtobuf()
			if err != nil {
				return err
			}
		}
		web.Contents = contents
		_, err = lns.Rpc.RegisterWebsite(lns.Context(), pipe)
		if err != nil {
			return err
		}

		_, err = lns.Rpc.StartWebsite(lns.Context(), &clientpb.CtrlPipeline{
			Name:       newWebsite.WebsiteName,
			ListenerId: lns.Name,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

type listener struct {
	Rpc       listenerrpc.ListenerRPCClient
	Name      string
	IP        string
	pipelines core.Pipelines
	conn      *grpc.ClientConn
	cfg       *configs.ListenerConfig
	websites  map[string]*Website
}

func (lns *listener) RegisterAndStart(pipeline *clientpb.Pipeline) error {
	if !pipeline.Enable {
		return nil
	}
	_, err := lns.Rpc.RegisterPipeline(lns.Context(), pipeline)
	if err != nil {
		return err
	}

	_, err = lns.Rpc.StartPipeline(lns.Context(), &clientpb.CtrlPipeline{
		Name:           pipeline.Name,
		ListenerId:     lns.ID(),
		BeaconPipeline: pipeline.BeaconPipeline,
		Target:         pipeline.Target,
	})
	if err != nil {
		return err
	}
	return nil
}

func (lns *listener) ID() string {
	return lns.Name
}

func (lns *listener) ToProtobuf() *clientpb.Listener {
	return &clientpb.Listener{
		Id: lns.ID(),
	}
}

func (lns *listener) Context() context.Context {
	return metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
		"listener_id", lns.Name,
		"listener_ip", lns.IP,
	),
	)
}

func (lns *listener) Handler() {
	stream, err := lns.Rpc.JobStream(lns.Context())
	if err != nil {
		return
	}

	for {
		msg, err := stream.Recv()
		if err != nil {
			logs.Log.Errorf(err.Error())
			continue
		}

		var handlerErr error
		switch msg.Ctrl {
		case consts.CtrlPipelineStart:
			handlerErr = lns.handlerStart(msg.Job)
		case consts.CtrlPipelineStop, consts.CtrlRemStop:
			handlerErr = lns.handlerStop(msg.Job)
		case consts.CtrlPipelineSync:
			handlerErr = lns.syncPipeline(msg.Job)
		case consts.CtrlWebsiteStart:
			handlerErr = lns.handleStartWebsite(msg.Job)
		case consts.CtrlWebsiteStop:
			handlerErr = lns.handleStopWebsite(msg.Job)
		case consts.CtrlWebsiteRegister:
			handlerErr = lns.handleRegisterWebsite(msg.Job)
		case consts.CtrlWebContentAdd:
			handlerErr = lns.handleWebContentAdd(msg.Job)
		case consts.CtrlWebContentUpdate:
			handlerErr = lns.handleWebContentUpdate(msg.Job)
		case consts.CtrlWebContentRemove:
			handlerErr = lns.handleWebContentRemove(msg.Job)
		case consts.CtrlRemStart:
			handlerErr = lns.handleStartRem(msg.Job)
		case consts.CtrlRemCtrl:
			handlerErr = lns.handlerRemCtrl(msg.Job)
		case consts.CtrlRemLog:
			handlerErr = lns.handlerRemLog(msg.Job)
		}

		status := &clientpb.JobStatus{
			ListenerId: lns.ID(),
			Ctrl:       msg.Ctrl,
			CtrlId:     msg.Id,
			Job:        msg.Job,
		}
		if handlerErr != nil {
			status.Status = consts.CtrlStatusFailed
			status.Error = handlerErr.Error()
			logs.Log.Errorf("[listener.%s] job ctrl %d %s %s failed: %s", lns.ID(), msg.Id, msg.Job.Name, msg.Ctrl, handlerErr.Error())
		} else {
			status.Status = consts.CtrlStatusSuccess
			logs.Log.Importantf("[listener.%s] job ctrl %d %s %s success", lns.ID(), msg.Id, msg.Job.Name, msg.Ctrl)
		}
		if err := stream.Send(status); err != nil {
			logs.Log.Errorf(err.Error())
			return
		}
	}
}

func (lns *listener) handlerStart(job *clientpb.Job) error {
	pipeline, err := lns.startPipeline(job.GetPipeline())
	if err != nil {
		return err
	}
	_, err = lns.Rpc.SyncPipeline(lns.Context(), pipeline.ToProtobuf())
	if err != nil {
		return err
	}
	job.Name = pipeline.ID()
	err = lns.autoBuild(job.GetPipeline())
	if err != nil {
		logs.Log.Warn(err)
	}
	return nil
}

func (lns *listener) autoBuild(pipeline *clientpb.Pipeline) error {
	if len(pipeline.Target) == 0 {
		return fmt.Errorf("pipeline %s target is empty, auto build canceled", pipeline.Name)
	}
	var buildType string
	var beaconPipeline string
	var pulsePipeline string
	var input map[string]string

	_, workflowErr := lns.Rpc.WorkflowStatus(lns.Context(), &clientpb.GithubWorkflowRequest{})
	_, dockerErr := lns.Rpc.DockerStatus(lns.Context(), &clientpb.Empty{})
	if workflowErr != nil && dockerErr != nil {
		return fmt.Errorf("workflow and docker not worked: %s, %s", workflowErr.Error(), dockerErr.Error())
	}

	for _, target := range pipeline.Target {
		if pipeline.Parser == consts.CommandBuildPulse {
			if !strings.Contains(target, "windows") {
				logs.Log.Warnf("pulse build target must be windows, %s is not supported", target)
				continue
			}
			buildType = consts.CommandBuildPulse
			beaconPipeline = pipeline.BeaconPipeline
			pulsePipeline = pipeline.Name
			input = map[string]string{
				"package": consts.CommandBuildPulse,
				"targets": target,
			}
		} else {
			buildType = consts.CommandBuildBeacon
			beaconPipeline = pipeline.Name
			input = map[string]string{
				"package": consts.CommandBuildBeacon,
				"targets": target,
			}
		}
		targetMap, ok := consts.GetBuildTarget(target)
		if !ok {
			fmt.Printf("Error getting build target for %s\n", target)
			continue
		}
		_, err := lns.Rpc.FindArtifact(lns.Context(), &clientpb.Artifact{
			Pipeline: pipeline.Name,
			Target:   target,
			Type:     buildType,
			Platform: targetMap.OS,
			Arch:     targetMap.Arch,
		})
		if !errors.Is(err, errs.ErrNotFoundArtifact) && err != nil {
			logs.Log.Errorf("Error finding artifact for %s: %v\n", target, err)
			continue
		} else if err == nil {
			continue
		}
		profileName := codenames.GetCodename()
		_, err = lns.Rpc.NewProfile(lns.Context(), &clientpb.Profile{
			Name:            profileName,
			PipelineId:      beaconPipeline,
			PulsePipelineId: pulsePipeline,
		})
		if err != nil {
			return err
		}
		if workflowErr == nil {
			_, err = lns.Rpc.TriggerWorkflowDispatch(lns.Context(), &clientpb.GithubWorkflowRequest{
				Inputs:  input,
				Profile: profileName,
			})
			if err != nil {
				return err
			}
		} else if dockerErr == nil {
			_, err = lns.Rpc.Build(lns.Context(), &clientpb.Generate{
				Target:      target,
				ProfileName: profileName,
				Type:        buildType,
				Srdi:        true,
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (lns *listener) syncPipeline(pipeline *clientpb.Job) error {
	p := lns.pipelines.Get(pipeline.Name)
	if p == nil {
		return fmt.Errorf("pipeline %s not found", pipeline.Name)
	}

	_, err := lns.Rpc.SyncPipeline(lns.Context(), p.ToProtobuf())
	if err != nil {
		return err
	}

	return nil
}

func (lns *listener) startPipeline(pipelinepb *clientpb.Pipeline) (core.Pipeline, error) {
	var err error
	p := lns.pipelines.Get(pipelinepb.Name)
	switch pipelinepb.Body.(type) {
	case *clientpb.Pipeline_Tcp:
		p, err = NewTcpPipeline(lns.Rpc, pipelinepb)
	case *clientpb.Pipeline_Bind:
		p, err = NewBindPipeline(lns.Rpc, pipelinepb)
	default:
		return nil, fmt.Errorf("not impl")
	}
	if err != nil {
		return nil, err
	}
	err = p.Start()
	if err != nil {
		return nil, err
	}
	lns.pipelines.Add(p)
	return p, nil
}

func (lns *listener) handlerStop(job *clientpb.Job) error {
	pipeline := job.GetPipeline()
	p := lns.pipelines.Get(pipeline.Name)
	if p == nil {
		return errors.New("pipeline not found")
	}
	job.Name = p.ID()
	if err := p.Close(); err != nil {
		return err
	}
	delete(lns.pipelines, p.ID())
	return nil
}

func (lns *listener) handleStartWebsite(job *clientpb.Job) error {
	pipe := job.GetPipeline()
	web := pipe.GetWeb()
	w := lns.websites[pipe.Name]
	if w == nil {
		starResult, err := StartWebsite(lns.Rpc, job.GetPipeline(), web.Contents)
		if err != nil {
			return err
		}
		lns.websites[pipe.Name] = starResult
	} else {
		if err := w.Start(); err != nil {
			return err
		}
	}
	job.GetPipeline().Enable = true
	return nil
}

func (lns *listener) handleStopWebsite(job *clientpb.Job) error {
	pipe := job.GetPipeline()
	w := lns.websites[pipe.Name]
	if w == nil {
		return errors.New("website not found")
	}
	if err := w.Close(); err != nil {
		return err
	}
	delete(lns.websites, pipe.Name)
	return nil
}

func (lns *listener) handleRegisterWebsite(job *clientpb.Job) error {
	webContents := job.GetPipeline().GetWeb().Contents
	for _, content := range webContents {
		filePath := filepath.Join(configs.WebsitePath, content.File)
		if err := os.WriteFile(filePath, content.Content, os.ModePerm); err != nil {
			return err
		}
	}
	return nil
}

func (lns *listener) handleWebContentAdd(job *clientpb.Job) error {
	pipe := job.GetPipeline()
	web := pipe.GetWeb()
	w := lns.websites[pipe.Name]
	if w == nil {
		return errors.New("website not found")
	}
	for _, content := range web.Contents {
		w.AddContent(content)
	}
	return nil
}

func (lns *listener) handleWebContentUpdate(job *clientpb.Job) error {
	pipe := job.GetPipeline()
	web := pipe.GetWeb()
	w := lns.websites[pipe.Name]
	if w == nil {
		return errors.New("website not found")
	}
	for _, content := range web.Contents {
		w.AddContent(content)
	}
	return nil
}

func (lns *listener) handleWebContentRemove(job *clientpb.Job) error {
	pipe := job.GetPipeline()
	web := pipe.GetWeb()
	w := lns.websites[pipe.Name]
	if w == nil {
		return errors.New("website not found")
	}
	for path := range web.Contents {
		delete(w.Content, path)
	}
	return nil
}

func (lns *listener) handleStartRem(job *clientpb.Job) error {
	pipe := job.GetPipeline()
	pipe.Ip = lns.IP
	rem, err := NewRem(lns.Rpc, pipe)
	if err != nil {
		return err
	}

	err = rem.Start()
	if err != nil {
		return err
	}

	_, err = lns.Rpc.SyncPipeline(lns.Context(), rem.ToProtobuf())
	if err != nil {
		return err
	}

	lns.pipelines.Add(rem)
	job.Name = rem.ID()
	return nil
}

func (lns *listener) handlerRemCtrl(job *clientpb.Job) error {
	rem := lns.pipelines.Get(job.Name)
	if rem == nil {
		return errors.New("rem not found")
	}

	body := job.GetRemAgent()
	if body == nil {
		return errors.New("agent not found")
	}
	_, err := rem.(*REM).con.Fork(body.Id, body.Args)
	if err != nil {
		return err
	}
	return nil
}

func (lns *listener) handlerRemLog(job *clientpb.Job) error {
	rem := lns.pipelines.Get(job.Name)
	if rem == nil {
		return errors.New("rem not found")
	}

	body := job.GetRemAgent()
	if body == nil {
		return errors.New("agent not found")
	}
	a, ok := agent.Agents.Get(body.Id)
	if ok {
		job.Body = &clientpb.Job_RemLog{
			RemLog: &clientpb.RemLog{
				PipelineId: job.Name,
				AgentId:    body.Id,
				Log:        a.HistoryLog(),
			},
		}
		return nil
	} else {
		return errors.New("agent not found")
	}
}
