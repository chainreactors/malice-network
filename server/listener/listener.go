package listener

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	mtls "github.com/chainreactors/IoM-go/mtls"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/helper/utils/output"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/malice-network/helper/utils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var (
	Listener *listener
	// ListenerSessions 在 listener 层维护的 Sessions map (rawID -> Session)
)

var openListenerJobStream = func(client listenerrpc.ListenerRPCClient, ctx context.Context) (listenerrpc.ListenerRPC_JobStreamClient, error) {
	return client.JobStream(ctx)
}

func NewListener(clientConf *mtls.ClientConfig, cfg *configs.ListenerConfig, serverEnable bool) error {
	options, err := mtls.GetGrpcOptions([]byte(clientConf.CACertificate), []byte(clientConf.Certificate), []byte(clientConf.PrivateKey), clientConf.Type)
	if err != nil {
		return err
	}
	listenerCfg, err := mtls.ReadConfig(cfg.Auth)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var address = listenerCfg.Address()
	if serverEnable && cfg.Enable {
		address = fmt.Sprintf("%s:%d", "127.0.0.1", listenerCfg.Port)
	}
	conn, err := grpc.DialContext(ctx, address, options...)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %v", err)
	}

	lns := &listener{
		Rpc:       listenerrpc.NewListenerRPCClient(conn),
		Name:      cfg.Name,
		IP:        cfg.IP,
		pipelines: core.NewPipelines(),
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
	core.GoGuarded("listener-job-stream:"+lns.ID(), lns.Handler, core.LogGuardedError("listener-job-stream:"+lns.ID()))
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

func (lns *listener) Close() error {
	if lns == nil {
		return nil
	}

	var errs []error

	for _, pipeline := range lns.pipelines.ToProtobuf().GetPipelines() {
		if pipeline == nil {
			continue
		}
		runtime := lns.pipelines.Get(pipeline.Name)
		if runtime == nil {
			continue
		}
		if err := runtime.Close(); err != nil {
			errs = append(errs, err)
		}
		lns.pipelines.Delete(pipeline.Name)
	}

	for name, website := range lns.websites {
		if website == nil {
			continue
		}
		if err := website.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close website %s: %w", name, err))
		}
		delete(lns.websites, name)
	}

	if lns.conn != nil {
		if err := lns.conn.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if Listener == lns {
		Listener = nil
	}

	return errors.Join(errs...)
}

func (lns *listener) RegisterAndStart(pipeline *clientpb.Pipeline) error {
	if !pipeline.Enable {
		return nil
	}

	var err error
	// 如果启用了安全模式，生成密钥对
	//if pipeline.Secure != nil && pipeline.Secure.Enable {
	//	err = lns.generateSecureKeyPair(pipeline)
	//	if err != nil {
	//		return err
	//	}
	//}

	_, err = lns.Rpc.RegisterPipeline(lns.Context(), pipeline)
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

func (lns *listener) Handler() error {
	stream, err := openListenerJobStream(lns.Rpc, lns.Context())
	if err != nil {
		return fmt.Errorf("open listener job stream: %w", err)
	}

	for {
		msg, err := stream.Recv()
		if err != nil {
			return fmt.Errorf("listener %s job stream recv: %w", lns.ID(), err)
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
		case consts.CtrlRemAgentReconfigure:
			handlerErr = lns.handlerRemAgentReconfigure(msg.Job)
		case consts.CtrlListenerSyncSession:
			core.ListenerSessions.Add(msg.Session)
			continue
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
			return fmt.Errorf("listener %s job stream send: %w", lns.ID(), err)
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

	resp, err := lns.Rpc.CheckSource(lns.Context(), &clientpb.BuildConfig{
		Target: artifact.Target,
	})
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
		BuildType:   artifact.Type,
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
	if pipelinepb == nil {
		return nil, fmt.Errorf("pipeline is nil")
	}

	// Idempotency: if pipeline already exists locally, treat start as a no-op.
	if existing := lns.pipelines.Get(pipelinepb.Name); existing != nil {
		return existing, nil
	}

	var p core.Pipeline
	switch pipelinepb.Body.(type) {
	case *clientpb.Pipeline_Tcp:
		p, err = NewTcpPipeline(lns.Rpc, pipelinepb)
	case *clientpb.Pipeline_Bind:
		p, err = NewBindPipeline(lns.Rpc, pipelinepb)
	case *clientpb.Pipeline_Http:
		p, err = NewHttpPipeline(lns.Rpc, pipelinepb)
	case *clientpb.Pipeline_Custom:
		p = NewCustomPipeline(pipelinepb)
	default:
		// Fallback: treat any unknown body as custom pipeline.
		p = NewCustomPipeline(pipelinepb)
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
	lns.pipelines.Delete(p.ID())
	return nil
}

func (lns *listener) handleStartWebsite(job *clientpb.Job) error {
	pipe := job.GetPipeline()
	if pipe == nil {
		return errors.New("pipeline is nil")
	}

	// Idempotency: website already started in this listener process.
	if existing := lns.websites[pipe.Name]; existing != nil && existing.Enable {
		_, err := lns.Rpc.SyncPipeline(lns.Context(), existing.ToProtobuf())
		return err
	}

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
	pipe := job.GetPipeline()
	if pipe == nil {
		return errors.New("pipeline is nil")
	}
	web := pipe.GetWeb()
	if web == nil {
		return errors.New("website is nil")
	}

	websiteDir, err := fileutils.SafeJoin(configs.WebsitePath, pipe.Name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(websiteDir, 0o700); err != nil {
		return err
	}

	for _, content := range web.Contents {
		if content == nil {
			continue
		}

		nameHint := content.File
		if nameHint == "" {
			switch {
			case content.Id != "":
				nameHint = content.Id
			case content.Path != "":
				nameHint = content.Path
			default:
				return errors.New("web content missing file/path/id")
			}
		}

		fileName, err := fileutils.SanitizeBasename(nameHint)
		if err != nil {
			return err
		}
		filePath, err := fileutils.SafeJoin(websiteDir, fileName)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filePath, content.Content, 0o600); err != nil {
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
	if err := w.AddContent(job.Content); err != nil {
		return err
	}
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

	en := output.Encode(job.Content.Path)

	w.mu.Lock()
	w.Artifact[en] = &clientpb.WebContent{
		Path: job.Content.Path,
	}
	w.mu.Unlock()

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
	return w.AddContent(job.Content)
}

func (lns *listener) handleWebContentRemove(job *clientpb.Job) error {
	pipe := job.GetPipeline()
	web := pipe.GetWeb()
	w := lns.websites[pipe.Name]
	if w == nil {
		return errors.New("website not found")
	}
	w.mu.Lock()
	for path := range web.Contents {
		delete(w.Content, path)
	}
	w.mu.Unlock()
	return nil
}

func (lns *listener) handleStartRem(job *clientpb.Job) error {
	pipe := job.GetPipeline()
	if pipe == nil {
		return errors.New("pipeline is nil")
	}
	pipe.Ip = lns.IP

	// Idempotency: REM already started in this listener process.
	if existing := lns.pipelines.Get(pipe.Name); existing != nil {
		remPipeline, ok := existing.(*REM)
		if ok && remPipeline.Enable {
			// Still healthy — just sync its current state.
			_, err := lns.Rpc.SyncPipeline(lns.Context(), existing.ToProtobuf())
			return err
		}
		// Dead pipeline (crashed via runtimeErrorHandler) — remove the stale
		// entry so we can create a fresh one below.
		lns.pipelines.Delete(existing.ID())
	}

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

// generateSecureKeyPair 为pipeline生成安全密钥对
// 生成两对密钥：server密钥对和implant密钥对，然后进行交换分发
func (lns *listener) generateSecureKeyPair(pipeline *clientpb.Pipeline) error {
	// 检查是否已经有密钥对
	if pipeline.Secure != nil &&
		pipeline.Secure.ServerKeypair != nil &&
		pipeline.Secure.ImplantKeypair != nil &&
		pipeline.Secure.ServerKeypair.PrivateKey != "" &&
		pipeline.Secure.ImplantKeypair.PublicKey != "" {
		logs.Log.Infof("[secure] pipeline %s already has keypair, skipping generation", pipeline.Name)
		return nil
	}

	logs.Log.Infof("[secure] generating two keypairs for pipeline %s", pipeline.Name)

	// 生成Server密钥对
	serverKeyPair, err := cryptography.RandomAgeKeyPair()
	if err != nil {
		return fmt.Errorf("failed to generate server keypair: %v", err)
	}

	// 生成Implant密钥对
	implantKeyPair, err := cryptography.RandomAgeKeyPair()
	if err != nil {
		return fmt.Errorf("failed to generate implant keypair: %v", err)
	}

	// 确保SecureConfig存在
	if pipeline.Secure == nil {
		pipeline.Secure = &clientpb.Secure{
			Enable: true,
		}
	}

	// 创建Server密钥对
	pipeline.Secure.ServerKeypair = &clientpb.KeyPair{
		PublicKey:  serverKeyPair.Public,
		PrivateKey: serverKeyPair.Private, // Pipeline保存server私钥，用于解密implant发来的数据
	}

	// 创建Implant密钥对
	pipeline.Secure.ImplantKeypair = &clientpb.KeyPair{
		PublicKey:  implantKeyPair.Public, // Pipeline保存implant公钥，用于加密发给implant的数据
		PrivateKey: implantKeyPair.Private,
	}

	logs.Log.Infof("[secure] generated keypairs for pipeline %s", pipeline.Name)
	logs.Log.Infof("[secure] pipeline stores: server_private_key + implant_public_key")

	return nil
}
