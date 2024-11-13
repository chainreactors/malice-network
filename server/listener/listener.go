package listener

import (
	"context"
	"errors"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/helper/utils/mtls"
	"github.com/chainreactors/malice-network/helper/utils/webutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"google.golang.org/grpc"
	"os"
	"path/filepath"
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
		websites:  make(core.Websites),
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
		tls, err := newWebsite.TlsConfig.ReadCert()
		if err != nil {
			return err
		}
		addWeb := &clientpb.WebsiteAddContent{
			Name:     newWebsite.WebsiteName,
			Contents: map[string]*clientpb.WebContent{},
		}
		cPath, _ := filepath.Abs(newWebsite.ContentPath)
		fileIfo, err := os.Stat(cPath)

		if fileIfo.IsDir() {
			_ = webutils.WebAddDirectory(addWeb, newWebsite.RootPath, cPath)
		} else {
			file, err := os.Open(cPath)
			webutils.WebAddFile(addWeb, newWebsite.RootPath, webutils.SniffContentType(file), cPath)
			if err != nil {
				return err
			}
			err = file.Close()
			if err != nil {
				return err
			}
		}
		webProtobuf := &clientpb.Pipeline{
			Name:       newWebsite.WebsiteName,
			ListenerId: lis.Name,
			Body: &clientpb.Pipeline_Web{
				Web: &clientpb.Website{
					Root:     newWebsite.RootPath,
					Port:     uint32(newWebsite.Port),
					Contents: addWeb.Contents,
				},
			},
			Tls: tls.ToProtobuf(),
		}
		_, err = lis.Rpc.RegisterWebsite(context.Background(), webProtobuf)
		if err != nil {
			return err
		}
		if !newWebsite.Enable {
			continue
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
	websites  core.Websites
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
		Name:       pipeline.Name,
		ListenerId: lns.ID(),
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
		var resp *clientpb.JobStatus
		switch msg.Ctrl {
		case consts.CtrlPipelineStart:
			resp = lns.startHandler(msg.Job)
		case consts.CtrlPipelineStop:
			resp = lns.stopHandler(msg.Job)
		case consts.CtrlWebsiteStart:
			resp = lns.startWebsite(msg.Job)
		case consts.CtrlWebsiteStop:
			resp = lns.stopWebsite(msg.Job)
		case consts.CtrlWebsiteRegister:
			resp = lns.registerWebsite(msg.Job)
		}
		err = stream.Send(resp)
		if err != nil {
			logs.Log.Errorf(err.Error())
			continue
		}
	}
}

func (lns *listener) startHandler(job *clientpb.Job) *clientpb.JobStatus {
	pipeline, err := lns.startPipeline(job.GetPipeline())
	if err != nil {
		return &clientpb.JobStatus{
			ListenerId: lns.ID(),
			Ctrl:       consts.CtrlJobStart,
			Status:     consts.CtrlStatusFailed,
			Error:      err.Error(),
			Job:        job,
		}
	}
	job.Name = pipeline.ID()
	return &clientpb.JobStatus{
		ListenerId: lns.ID(),
		Ctrl:       consts.CtrlJobStart,
		Status:     consts.CtrlStatusSuccess,
		Job:        job,
	}
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
	return p, nil
}

func (lns *listener) stopHandler(job *clientpb.Job) *clientpb.JobStatus {
	var err error
	pipeline := job.GetPipeline()
	switch pipeline.Body.(type) {
	case *clientpb.Pipeline_Tcp:
		p := lns.pipelines.Get(pipeline.Name)
		job.Name = p.ID()
		if p == nil {
			return &clientpb.JobStatus{
				ListenerId: lns.ID(),
				Ctrl:       consts.CtrlJobStop,
				Status:     consts.CtrlStatusFailed,
				Error:      errors.New("pipeline not found").Error(),
				Job:        job,
			}
		}
		err = p.Close()
		if err != nil {
			break
		}
		coreJob := core.Jobs.Get(pipeline.Name)
		if coreJob != nil {
			core.Jobs.Remove(coreJob)
		}
	}
	if err != nil {
		return &clientpb.JobStatus{
			ListenerId: lns.ID(),
			Ctrl:       consts.CtrlJobStop,
			Status:     consts.CtrlStatusFailed,
			Error:      err.Error(),
			Job:        job,
		}
	}
	return &clientpb.JobStatus{
		ListenerId: lns.ID(),
		Ctrl:       consts.CtrlJobStop,
		Status:     consts.CtrlStatusSuccess,
		Job:        job,
	}
}

func (lns *listener) startWebsite(job *clientpb.Job) *clientpb.JobStatus {
	var err error
	getWeb := job.GetPipeline().GetWeb()
	job.Name = getWeb.ID
	w := lns.websites.Get(getWeb.ID)
	if w == nil {
		starResult, err := StartWebsite(job.GetPipeline(), getWeb.Contents)
		if err != nil {
			return &clientpb.JobStatus{
				ListenerId: lns.ID(),
				Ctrl:       consts.CtrlJobStart,
				Status:     consts.CtrlStatusFailed,
				Error:      err.Error(),
				Job:        job,
			}
		}
		lns.websites.Add(starResult)
	} else {
		err = w.Start()
		if err != nil {
			return &clientpb.JobStatus{
				ListenerId: lns.ID(),
				Ctrl:       consts.CtrlJobStart,
				Status:     consts.CtrlStatusFailed,
				Error:      err.Error(),
				Job:        job,
			}
		}
	}
	job.GetPipeline().Enable = true
	return &clientpb.JobStatus{
		ListenerId: lns.ID(),
		Ctrl:       consts.CtrlJobStart,
		Status:     consts.CtrlStatusSuccess,
		Job:        job,
	}
}

func (lns *listener) stopWebsite(job *clientpb.Job) *clientpb.JobStatus {
	var err error
	getWeb := job.GetPipeline().GetWeb()
	job.Name = getWeb.ID
	w := lns.websites.Get(getWeb.ID)
	if w == nil {
		return &clientpb.JobStatus{
			ListenerId: lns.ID(),
			Ctrl:       consts.CtrlJobStop,
			Status:     consts.CtrlStatusFailed,
			Error:      errors.New("website not found").Error(),
			Job:        job,
		}
	}
	err = w.Close()
	if err != nil {
		return &clientpb.JobStatus{
			ListenerId: lns.ID(),
			Ctrl:       consts.CtrlJobStop,
			Status:     consts.CtrlStatusFailed,
			Error:      err.Error(),
			Job:        job,
		}
	}
	coreJob := core.Jobs.Get(getWeb.ID)
	if coreJob != nil {
		core.Jobs.Remove(coreJob)
	}
	return &clientpb.JobStatus{
		ListenerId: lns.ID(),
		Ctrl:       consts.CtrlJobStop,
		Status:     consts.CtrlStatusSuccess,
		Job:        job,
	}
}

func (lns *listener) registerWebsite(job *clientpb.Job) *clientpb.JobStatus {
	webAssets := job.GetWebsiteAssets().GetAssets()
	for _, asset := range webAssets {
		filePath := filepath.Join(configs.WebsitePath, asset.FileName)
		err := os.WriteFile(filePath, asset.Content, os.ModePerm)
		if err != nil {
			return &clientpb.JobStatus{
				ListenerId: lns.ID(),
				Status:     consts.CtrlStatusFailed,
				Ctrl:       consts.CtrlWebUpload,
				Error:      err.Error(),
			}
		}
	}
	return &clientpb.JobStatus{
		ListenerId: lns.ID(),
		Status:     consts.CtrlStatusSuccess,
		Ctrl:       consts.CtrlWebUpload,
	}
}
