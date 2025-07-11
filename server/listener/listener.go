package listener

import (
	"context"
	"errors"
	"fmt"
	"github.com/chainreactors/malice-network/helper/utils"
	"github.com/chainreactors/malice-network/helper/utils/formatutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"os"
	"path/filepath"
	"time"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/helper/utils/mtls"
	"github.com/chainreactors/malice-network/server/internal/certutils"
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
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	conn, err := grpc.DialContext(ctx, listenerCfg.Address(), options...)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %v", err)
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

	for _, httpPipeline := range cfg.HttpPipelines {
		pipeline, err := httpPipeline.ToProtobuf(lns.Name)
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

		if pipe.Tls.Enable && !pipe.Tls.Acme {
			_, err = lns.Rpc.GenerateSelfCert(lns.Context(), pipe)
		} else if pipe.Tls.Enable && pipe.Tls.Acme {
			_, err = lns.Rpc.GenerateAcmeCert(lns.Context(), pipe)
		}
		if err != nil {
			return err
		}

		_, err = lns.Rpc.StartWebsite(lns.Context(), &clientpb.CtrlPipeline{
			Name:       newWebsite.WebsiteName,
			ListenerId: lns.Name,
			Pipeline:   pipe,
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

	if pipeline.Tls.Enable && !pipeline.Tls.Acme {
		_, err = lns.Rpc.GenerateSelfCert(lns.Context(), pipeline)
	} else if pipeline.Tls.Enable && pipeline.Tls.Acme {
		_, err = lns.Rpc.GenerateAcmeCert(lns.Context(), pipeline)
	}
	if err != nil {
		return err
	}

	_, err = lns.Rpc.StartPipeline(lns.Context(), &clientpb.CtrlPipeline{
		Name:       pipeline.Name,
		ListenerId: lns.ID(),
		Pipeline:   pipeline,
	})
	if err != nil {
		return err
	}

	lns.autoBuild(lns.cfg.AutoBuildConfig, pipeline)
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
			handlerErr = lns.handleWebContentAdd(msg)
		case consts.CtrlWebContentUpdate:
			handlerErr = lns.handleWebContentUpdate(msg)
		case consts.CtrlWebContentRemove:
			handlerErr = lns.handleWebContentRemove(msg.Job)
		case consts.CtrlWebContentAddArtifact:
			handlerErr = lns.handleAmountArtifact(msg)
		case consts.CtrlRemStart:
			handlerErr = lns.handleStartRem(msg.Job)
		case consts.CtrlRemAgentCtrl:
			handlerErr = lns.handlerRemAgentCtrl(msg.Job)
		case consts.CtrlRemAgentLog:
			handlerErr = lns.handlerRemAgentLog(msg.Job)
		case consts.CtrlRemAgentStop:
			handlerErr = lns.handlerRemAgentStop(msg.Job)
		case consts.CtrlAcme:
			handlerErr = lns.handlerAcme(msg.Job)

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
	return nil
}

func (lns *listener) autoBuild(autoBuild *configs.AutoBuildConfig, pipeline *clientpb.Pipeline) {
	if autoBuild == nil || !autoBuild.Enable || len(autoBuild.Target) == 0 || len(autoBuild.Pipeline) == 0 {
		logs.Log.Debugf("not set auto_build/target/pipeline, skip auto build")
		return
	}

	if !utils.StringInSlice(pipeline.Name, autoBuild.Pipeline) {
		logs.Log.Debugf("%s pieline not auto build list", pipeline.Name)
		return
	}

	if !(pipeline.Type == consts.TCPPipeline || pipeline.Type == consts.HTTPPipeline || pipeline.Type == consts.RemPipeline) {
		logs.Log.Debugf("%s pieline not support auto build", pipeline.Type)
		return
	}

	for _, target := range autoBuild.Target {
		targetMap, ok := consts.GetBuildTarget(target)
		if !ok {
			logs.Log.Warnf("invalid build target: %s, skip auto build", target)
			continue
		}

		if autoBuild.BuildPulse {
			if err := lns.executeBuild(pipeline.Name+"_default", &clientpb.Artifact{
				Target:   target,
				Platform: targetMap.OS,
				Arch:     targetMap.Arch,
				Type:     consts.CommandBuildPulse,
				Pipeline: pipeline.Name,
			}); err != nil {
				logs.Log.Warnf("Error building %s: %v", target, err)
			}
		}

		if err := lns.executeBuild(pipeline.Name+"_default", &clientpb.Artifact{
			Target:   target,
			Platform: targetMap.OS,
			Arch:     targetMap.Arch,
			Type:     consts.CommandBuildBeacon,
			Pipeline: pipeline.Name,
		}); err != nil {
			logs.Log.Warnf("Error building %s: %v", target, err)
		}
	}
}

// 执行构建
func (lns *listener) executeBuild(profileName string, artifact *clientpb.Artifact) error {

	resp, err := lns.Rpc.CheckSource(lns.Context(), &clientpb.BuildConfig{})
	if err != nil {
		return err
	}
	_, err = lns.Rpc.FindArtifact(lns.Context(), artifact)
	if err == nil {
		return nil
	} else {
		err = nil
	}
	_, err = lns.Rpc.Build(lns.Context(), &clientpb.BuildConfig{
		Target:      artifact.Target,
		ProfileName: profileName,
		Type:        artifact.Type,
		Source:      resp.Source,
	})
	return err
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
	case *clientpb.Pipeline_Http:
		p, err = NewHttpPipeline(lns.Rpc, pipelinepb)
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

	website, err := StartWebsite(lns.Rpc, job.GetPipeline(), web.Contents)
	if err != nil {
		return err
	}
	lns.websites[pipe.Name] = website
	_, err = lns.Rpc.SyncPipeline(lns.Context(), website.ToProtobuf())
	if err != nil {
		return err
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

func (lns *listener) handleWebContentAdd(job *clientpb.JobCtrl) error {
	pipe := job.GetJob()
	w := lns.websites[pipe.Name]
	if w == nil {
		return errors.New("website not found")
	}
	w.AddContent(job.Content)
	job.Job.Contents = map[string]*clientpb.WebContent{
		job.Content.Path: &clientpb.WebContent{
			Id:   job.Content.Path,
			Path: job.Content.Path,
		},
	}
	return nil
}

func (lns *listener) handleAmountArtifact(job *clientpb.JobCtrl) error {
	pipe := job.GetJob()
	w := lns.websites[pipe.Pipeline.Name]
	if w == nil {
		return errors.New("website not found")
	}

	en := formatutils.Encode(job.Content.Path)

	w.Artifact[en] = &clientpb.WebContent{
		Path: job.Content.Path,
	}

	job.Job.Contents = map[string]*clientpb.WebContent{
		job.Content.Path: &clientpb.WebContent{
			Id:   job.Content.Path,
			Path: en,
		},
	}
	return nil
}

func (lns *listener) handleWebContentUpdate(job *clientpb.JobCtrl) error {
	pipe := job.GetJob()
	w := lns.websites[pipe.Name]
	if w == nil {
		return errors.New("website not found")
	}
	w.AddContent(job.Content)
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

func (lns *listener) handlerAcme(job *clientpb.Job) error {
	pipeline := job.GetPipeline()
	var has80 bool
	var website *Website
	var websiteName string

	for _, w := range lns.websites {
		if w.port == 80 && w.Enable {
			has80 = true
			break
		}
	}

	if !has80 {
		websiteName = pipeline.Tls.Domain + "_acme"
		web := &clientpb.Pipeline{
			Name:       websiteName,
			ListenerId: pipeline.ListenerId,
			Enable:     false,
			Ip:         lns.IP,
			Body: &clientpb.Pipeline_Web{
				Web: &clientpb.Website{
					Name:     websiteName,
					Root:     "/",
					Port:     80,
					Contents: make(map[string]*clientpb.WebContent),
				},
			},
			Tls: pipeline.Tls,
		}
		var err error
		website, err = StartWebsite(lns.Rpc, web, make(map[string]*clientpb.WebContent))
		if err != nil {
			return err
		}
		lns.websites[websiteName] = website
	}

	certutils.GetACMEManager().RegisterDomain(pipeline.Tls.Domain)
	go func() {
		//defer func() {
		//	if website != nil {
		//		website.Close()
		//		delete(lns.websites, websiteName)
		//	}
		//}()

		tls, err := certutils.GetAcmeTls(pipeline.GetTls())
		if err != nil {
			return
		}
		pipeline.Tls = tls
		_, err = lns.Rpc.SaveAcmeCert(lns.Context(), pipeline)
		if err != nil {
			return
		}
	}()

	return nil
}
