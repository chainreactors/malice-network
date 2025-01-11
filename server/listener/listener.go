package listener

import (
	"context"
	"errors"
	"fmt"
	"github.com/chainreactors/malice-network/helper/codenames"
	"github.com/chainreactors/malice-network/helper/errs"
	"os"
	"path/filepath"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/helper/utils/mtls"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"google.golang.org/grpc"
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
	serverAddress := listenerCfg.Address()
	conn, err := grpc.NewClient(serverAddress, options...)
	if err != nil {
		return err
	}

	lis := &listener{
		Rpc:       listenerrpc.NewListenerRPCClient(conn),
		Name:      cfg.Name,
		Host:      serverAddress,
		pipelines: make(core.Pipelines),
		conn:      conn,
		cfg:       cfg,
		websites:  make(map[string]*Website),
	}

	_, err = lis.Rpc.RegisterListener(context.Background(), &clientpb.RegisterListener{
		Id:   fmt.Sprintf("%s_%s", lis.Name, lis.Host),
		Name: lis.Name,
		Host: conn.Target(),
		Addr: serverAddress,
	})
	if err != nil {
		return err
	}
	go lis.Handler()
	Listener = lis

	for _, tcpPipeline := range cfg.TcpPipelines {
		pipeline, err := tcpPipeline.ToProtobuf(lis.Name)
		if err != nil {
			return err
		}
		err = lis.RegisterAndStart(pipeline)
		if err != nil {
			return err
		}
	}

	for _, bindPipeline := range cfg.BindPipelineConfig {
		pipeline, err := bindPipeline.ToProtobuf(lis.Name)
		if err != nil {
			return err
		}
		err = lis.RegisterAndStart(pipeline)
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
			ListenerId: lis.Name,
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
		_, err = lis.Rpc.RegisterWebsite(context.Background(), pipe)
		if err != nil {
			return err
		}

		_, err = lis.Rpc.StartWebsite(context.Background(), &clientpb.CtrlPipeline{
			Name:       newWebsite.WebsiteName,
			ListenerId: lis.Name,
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
	Host      string
	pipelines core.Pipelines
	conn      *grpc.ClientConn
	cfg       *configs.ListenerConfig
	websites  map[string]*Website
}

func (lns *listener) RegisterAndStart(pipeline *clientpb.Pipeline) error {
	if !pipeline.Enable {
		return nil
	}
	_, err := lns.Rpc.RegisterPipeline(context.Background(), pipeline)
	if err != nil {
		return err
	}

	_, err = lns.Rpc.StartPipeline(context.Background(), &clientpb.CtrlPipeline{
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

func (lns *listener) Handler() {
	stream, err := lns.Rpc.JobStream(context.Background())
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
			handlerErr = lns.startHandler(msg.Job)
		case consts.CtrlPipelineStop:
			handlerErr = lns.stopHandler(msg.Job)
		case consts.CtrlWebsiteStart:
			handlerErr = lns.startWebsite(msg.Job)
		case consts.CtrlWebsiteStop:
			handlerErr = lns.stopWebsite(msg.Job)
		case consts.CtrlWebsiteRegister:
			handlerErr = lns.registerWebsite(msg.Job)
		case consts.CtrlWebContentAdd:
			handlerErr = lns.handleWebContentAdd(msg.Job)
		case consts.CtrlWebContentUpdate:
			handlerErr = lns.handleWebContentUpdate(msg.Job)
		case consts.CtrlWebContentRemove:
			handlerErr = lns.handleWebContentRemove(msg.Job)
		}

		status := &clientpb.JobStatus{
			ListenerId: lns.ID(),
			Ctrl:       msg.Ctrl,
			Job:        msg.Job,
		}
		if handlerErr != nil {
			status.Status = consts.CtrlStatusFailed
			status.Error = handlerErr.Error()
		} else {
			status.Status = consts.CtrlStatusSuccess
		}

		if err := stream.Send(status); err != nil {
			logs.Log.Errorf(err.Error())
			continue
		}
	}
}

func (lns *listener) startHandler(job *clientpb.Job) error {
	pipeline, err := lns.startPipeline(job.GetPipeline())
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
	if pipeline.Target == "" {
		return fmt.Errorf("pipeline %s target is empty, auto build canceled", pipeline.Name)
	}
	var buildType string
	var beaconPipeline string
	var pulsePipeline string
	var input map[string]string
	if pipeline.Parser == consts.ImplantPulse {
		buildType = consts.CommandBuildPulse
		beaconPipeline = pipeline.BeaconPipeline
		pulsePipeline = pipeline.Name
		input = map[string]string{
			"package": consts.CommandBuildPulse,
			"targets": pipeline.Target,
		}
	} else {
		buildType = consts.CommandBuildBeacon
		beaconPipeline = pipeline.Name
		input = map[string]string{
			"package": consts.CommandBuildBeacon,
			"targets": pipeline.Target,
		}
	}
	target, _ := consts.GetBuildTarget(pipeline.Target)
	_, err := lns.Rpc.FindArtifact(context.Background(), &clientpb.Artifact{
		Pipeline: pipeline.Name,
		Target:   pipeline.Target,
		Type:     buildType,
		Platform: target.OS,
		Arch:     target.Arch,
	})
	if !errors.Is(err, errs.ErrNotFoundArtifact) && err != nil {
		return err
	} else if err == nil {
		return nil
	}
	_, workflowErr := lns.Rpc.WorkflowStatus(context.Background(), &clientpb.GithubWorkflowRequest{})
	_, dockerErr := lns.Rpc.DockerStatus(context.Background(), &clientpb.Empty{})
	if workflowErr != nil && dockerErr != nil {
		return fmt.Errorf("workflow and docker not worked: %s, %s", workflowErr.Error(), dockerErr.Error())
	}
	profileName := codenames.GetCodename()
	_, err = lns.Rpc.NewProfile(context.Background(), &clientpb.Profile{
		Name:            profileName,
		PipelineId:      beaconPipeline,
		PulsePipelineId: pulsePipeline,
	})
	if err != nil {
		return err
	}
	if workflowErr == nil {
		_, err = lns.Rpc.TriggerWorkflowDispatch(context.Background(), &clientpb.GithubWorkflowRequest{
			Inputs:  input,
			Profile: profileName,
		})
		if err != nil {
			return err
		}
	} else if dockerErr == nil {
		_, err = lns.Rpc.Build(context.Background(), &clientpb.Generate{
			Target:      pipeline.Target,
			ProfileName: profileName,
			Type:        buildType,
			Srdi:        true,
		})
		if err != nil {
			return err
		}
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

func (lns *listener) stopHandler(job *clientpb.Job) error {
	pipeline := job.GetPipeline()
	switch pipeline.Body.(type) {
	case *clientpb.Pipeline_Tcp:
		p := lns.pipelines.Get(pipeline.Name)
		if p == nil {
			return errors.New("pipeline not found")
		}
		job.Name = p.ID()
		if err := p.Close(); err != nil {
			return err
		}
		if coreJob := core.Jobs.Get(pipeline.Name); coreJob != nil {
			core.Jobs.Remove(coreJob)
		}
	}
	return nil
}

func (lns *listener) startWebsite(job *clientpb.Job) error {
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

func (lns *listener) stopWebsite(job *clientpb.Job) error {
	pipe := job.GetPipeline()
	w := lns.websites[pipe.Name]
	if w == nil {
		return errors.New("website not found")
	}
	if err := w.Close(); err != nil {
		return err
	}
	if coreJob := core.Jobs.Get(pipe.Name); coreJob != nil {
		core.Jobs.Remove(coreJob)
	}
	return nil
}

func (lns *listener) registerWebsite(job *clientpb.Job) error {
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
