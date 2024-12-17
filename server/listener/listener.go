package listener

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/helper/utils/mtls"
	"github.com/chainreactors/malice-network/helper/utils/webutils"
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

	for _, rem := range cfg.REMs {
		if !rem.Enable {
			continue
		}
		pipeline, err := rem.ToProtobuf(lis.Name)
		if err != nil {
			return err
		}

		_, err = lis.Rpc.RegisterRem(context.Background(), pipeline)
		if err != nil {
			return err
		}

		_, err = lis.Rpc.StartRem(context.Background(), &clientpb.CtrlPipeline{
			Name:       pipeline.Name,
			ListenerId: lis.ID(),
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
		for _, content := range newWebsite.WebContents {
			addWeb := &clientpb.WebsiteAddContent{
				Name:     newWebsite.WebsiteName,
				Contents: map[string]*clientpb.WebContent{},
			}
			cPath, _ := filepath.Abs(content.Path)
			fileIfo, err := os.Stat(cPath)
			var path string
			if err != nil {
				logs.Log.Errorf(err.Error())
				continue
			}
			if fileIfo.IsDir() {
				logs.Log.Errorf("file is a directory")
				continue
			} else {
				file, err := os.Open(cPath)
				path = filepath.Join(newWebsite.RootPath, filepath.Base(cPath))
				path = filepath.ToSlash(path)
				webutils.WebAddFile(addWeb, path, webutils.SniffContentType(file), cPath, content.EncryptionConfig.Type, content.Parser)
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
			resp, err := lis.Rpc.RegisterWebsite(context.Background(), webProtobuf)
			if err != nil {
				return err
			}
			webProtobuf.GetWeb().ID = resp.ID
			_, err = lis.Rpc.UploadWebsite(context.Background(), webProtobuf.GetWeb())
			if err != nil {
				return err
			}
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

func (lns *listener) Handler() {
	stream, err := lns.Rpc.JobStream(context.Background())
	if err != nil {
		return
	}

	for {
		msg, err := stream.Recv()
		if err != nil {
			logs.Log.Errorf(err.Error())
			return
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
		case consts.CtrlRemStart:
			resp = lns.startRem(msg.Job)
		case consts.CtrlRemStop:
			resp = lns.stopRem(msg.Job)
		}
		err = stream.Send(resp)
		if err != nil {
			logs.Log.Errorf(err.Error())
			continue
		}
	}
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
	lns.pipelines.Add(p)
	return p, nil
}

func (lns *listener) stopHandler(job *clientpb.Job) *clientpb.JobStatus {
	var err error
	pipeline := job.GetPipeline()
	switch pipeline.Body.(type) {
	case *clientpb.Pipeline_Tcp:
		p := lns.pipelines.Get(pipeline.Name)
		if p == nil {
			return &clientpb.JobStatus{
				ListenerId: lns.ID(),
				Ctrl:       consts.CtrlJobStop,
				Status:     consts.CtrlStatusFailed,
				Error:      errors.New("pipeline not found").Error(),
				Job:        job,
			}
		}
		job.Name = p.ID()
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
	web := job.GetPipeline().GetWeb()
	job.Name = web.ID
	w := lns.websites.Get(web.ID)
	if w == nil {
		starResult, err := StartWebsite(lns.Rpc, job.GetPipeline(), web.Contents)
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
	webContents := job.GetPipeline().GetWeb().Contents
	for _, content := range webContents {
		filePath := filepath.Join(configs.WebsitePath, job.GetPipeline().GetWeb().ID)
		err := os.WriteFile(filePath, content.Content, os.ModePerm)
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

func (lns *listener) startRem(job *clientpb.Job) *clientpb.JobStatus {
	rem, err := NewRem(lns.Rpc, job.GetPipeline())
	if err != nil {
		return &clientpb.JobStatus{
			ListenerId: lns.ID(),
			Ctrl:       consts.CtrlJobStart,
			Status:     consts.CtrlStatusFailed,
			Error:      err.Error(),
			Job:        job,
		}
	}

	err = rem.Start()
	if err != nil {
		return &clientpb.JobStatus{
			ListenerId: lns.ID(),
			Ctrl:       consts.CtrlJobStart,
			Status:     consts.CtrlStatusFailed,
			Error:      err.Error(),
			Job:        job,
		}
	}

	lns.pipelines.Add(rem)
	job.Name = rem.ID()
	return &clientpb.JobStatus{
		ListenerId: lns.ID(),
		Ctrl:       consts.CtrlJobStart,
		Status:     consts.CtrlStatusSuccess,
		Job:        job,
	}
}

func (lns *listener) stopRem(job *clientpb.Job) *clientpb.JobStatus {
	p := lns.pipelines.Get(job.GetPipeline().Name)
	if p == nil {
		return &clientpb.JobStatus{
			ListenerId: lns.ID(),
			Ctrl:       consts.CtrlJobStop,
			Status:     consts.CtrlStatusFailed,
			Error:      "rem not found",
			Job:        job,
		}
	}

	job.Name = p.ID()
	err := p.Close()
	if err != nil {
		return &clientpb.JobStatus{
			ListenerId: lns.ID(),
			Ctrl:       consts.CtrlJobStop,
			Status:     consts.CtrlStatusFailed,
			Error:      err.Error(),
			Job:        job,
		}
	}

	coreJob := core.Jobs.Get(job.GetPipeline().Name)
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
