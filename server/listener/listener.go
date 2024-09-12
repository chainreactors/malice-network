package listener

import (
	"context"
	"errors"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/mtls"
	"github.com/chainreactors/malice-network/helper/website"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"github.com/chainreactors/malice-network/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/server/internal/certs"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"google.golang.org/grpc"
	"os"
	"path"
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

	l := &lispb.Pipelines{}
	for _, tcpPipeline := range cfg.TcpPipelines {
		if tcpPipeline.TlsConfig.CertFile == "" || tcpPipeline.TlsConfig.KeyFile == "" {
			cert, key, err := certs.GeneratePipelineCert(tcpPipeline.TlsConfig)
			if err != nil {
				return err
			}
			tcpPipeline.TlsConfig.CertFile = string(cert)
			tcpPipeline.TlsConfig.KeyFile = string(key)
		}
		pipeline := &lispb.Pipeline{
			Body: &lispb.Pipeline_Tcp{
				Tcp: &lispb.TCPPipeline{
					Name:       tcpPipeline.Name,
					Host:       tcpPipeline.Host,
					Port:       uint32(tcpPipeline.Port),
					ListenerId: lis.Name,
				},
			},
			Tls: tcpPipeline.TlsConfig.ToProtobuf(),
		}
		_, err = lis.Rpc.RegisterPipeline(context.Background(), pipeline)
		if err != nil {
			return err
		}
		l.Pipelines = append(l.Pipelines, pipeline)
	}
	for _, newWebsite := range cfg.Websites {
		if newWebsite.TlsConfig.CertFile == "" || newWebsite.TlsConfig.KeyFile == "" {
			cert, key, err := certs.GeneratePipelineCert(newWebsite.TlsConfig)
			if err != nil {
				return err
			}
			newWebsite.TlsConfig.CertFile = string(cert)
			newWebsite.TlsConfig.KeyFile = string(key)
		}
		addWeb := &lispb.WebsiteAddContent{
			Name:     newWebsite.WebsiteName,
			Contents: map[string]*lispb.WebContent{},
		}
		cPath, _ := filepath.Abs(newWebsite.ContentPath)

		fileIfo, err := os.Stat(cPath)

		if fileIfo.IsDir() {
			_ = website.WebAddDirectory(addWeb, newWebsite.RootPath, cPath)
		} else {
			file, err := os.Open(cPath)
			website.WebAddFile(addWeb, newWebsite.RootPath, website.SniffContentType(file), cPath)
			if err != nil {
				return err
			}
			err = file.Close()
			if err != nil {
				return err
			}
		}
		webProtobuf := &lispb.Pipeline{
			Body: &lispb.Pipeline_Web{
				Web: &lispb.Website{
					RootPath:   newWebsite.RootPath,
					Port:       uint32(newWebsite.Port),
					Name:       newWebsite.WebsiteName,
					ListenerId: lis.Name,
					Contents:   addWeb.Contents,
				},
			},
			Tls: newWebsite.TlsConfig.ToProtobuf(),
		}
		_, err = lis.Rpc.RegisterWebsite(context.Background(), webProtobuf)
		if err != nil {
			return err
		}
	}
	_, err = lis.Rpc.RegisterListener(context.Background(), &lispb.RegisterListener{
		Id:        fmt.Sprintf("%s_%s", lis.Name, lis.Host),
		Name:      lis.Name,
		Host:      conn.Target(),
		Addr:      serverAddress,
		Pipelines: l,
	})
	if err != nil {
		return err
	}
	lis.Start()
	Listener = lis
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

func (lns *listener) ID() string {
	return fmt.Sprintf("%s_%s", lns.Name, lns.Host)
}

func (lns *listener) Start() {
	go lns.Handler()
	for _, tcp := range lns.cfg.TcpPipelines {
		if !tcp.Enable {
			continue
		}
		pipeline, err := StartTcpPipeline(lns.conn, tcp)
		if err != nil {
			logs.Log.Errorf("Failed to start tcp pipeline %s", err)
			continue
		}
		logs.Log.Importantf("Started tcp pipeline %s, encryption: %t, tls: %t", pipeline.ID(), pipeline.Encryption.Enable, pipeline.TlsConfig.Enable)
		ch := make(chan bool)
		tcpPipeline := pipeline.ToProtobuf().(*lispb.TCPPipeline)
		tcpPipeline.ListenerId = lns.Name
		job := &core.Job{
			ID: core.CurrentJobID(),
			Message: &lispb.Pipeline{
				Body: &lispb.Pipeline_Tcp{
					Tcp: tcpPipeline,
				},
			},
			JobCtrl: ch,
			Name:    pipeline.Name,
		}
		core.Jobs.Add(job)
		l := core.Listeners.Get(lns.Name)
		l.Pipelines.Add(pipeline)
	}
	for _, newWebsite := range lns.cfg.Websites {
		if !newWebsite.Enable {
			continue
		}
		addWeb := &lispb.WebsiteAddContent{
			Name:     newWebsite.WebsiteName,
			Contents: map[string]*lispb.WebContent{},
		}
		cPath, _ := filepath.Abs(newWebsite.ContentPath)

		fileIfo, err := os.Stat(cPath)

		if fileIfo.IsDir() {
			_ = website.WebAddDirectory(addWeb, newWebsite.RootPath, cPath)
		} else {
			file, err := os.Open(cPath)
			website.WebAddFile(addWeb, newWebsite.RootPath, website.SniffContentType(file), cPath)
			if err != nil {
				logs.Log.Error(err)
				continue
			}
			err = file.Close()
			if err != nil {
				logs.Log.Error(err)
				continue
			}
		}
		startWebsite, err := StartWebsite(newWebsite, addWeb.Contents)
		if err != nil {
			logs.Log.Errorf("Failed to start website %s", err)
			continue
		}
		ch := make(chan bool)
		websitePipeline := startWebsite.ToProtobuf().(*lispb.Website)
		websitePipeline.ListenerId = lns.Name
		job := &core.Job{
			ID: core.CurrentJobID(),
			Message: &lispb.Pipeline{
				Body: &lispb.Pipeline_Web{
					Web: websitePipeline,
				},
			},
			JobCtrl: ch,
			Name:    startWebsite.websiteName,
		}
		core.Jobs.Add(job)
		l := core.Listeners.Get(lns.Name)
		l.Pipelines.Add(startWebsite)
		go startWebsite.Start()
	}
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
		case consts.RegisterWebsite:
			resp = lns.registerWebsite(msg.Job)
		}
		err = stream.Send(resp)
		if err != nil {
			return
		}
	}
}

func (lns *listener) startHandler(job *clientpb.Job) *clientpb.JobStatus {
	var err error
	pipeline := job.GetPipeline()
	switch pipeline.Body.(type) {
	case *lispb.Pipeline_Tcp:
		p := lns.pipelines.Get(pipeline.GetTcp().Name)
		if p == nil {
			tcpPipeline, err := StartTcpPipeline(lns.conn, ToTcpConfig(pipeline.GetTcp(), pipeline.GetTls()))
			job.Name = tcpPipeline.Name
			if err != nil {
				return &clientpb.JobStatus{
					ListenerId: lns.ID(),
					Ctrl:       consts.CtrlJobStart,
					Status:     consts.CtrlStatusFailed,
					Error:      err.Error(),
					Job:        job,
				}
			}
			lns.pipelines.Add(tcpPipeline)
		} else {
			err = p.Start()
			job.Name = p.ID()
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
	}
	return &clientpb.JobStatus{
		ListenerId: lns.ID(),
		Ctrl:       consts.CtrlJobStart,
		Status:     consts.CtrlStatusSuccess,
		Job:        job,
	}
}

func (lns *listener) stopHandler(job *clientpb.Job) *clientpb.JobStatus {
	var err error
	pipeline := job.GetPipeline()
	switch pipeline.Body.(type) {
	case *lispb.Pipeline_Tcp:
		p := lns.pipelines.Get(pipeline.GetTcp().Name)
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
		coreJob := core.Jobs.Get(pipeline.GetTcp().Name)
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
	job.Name = getWeb.Name
	w := lns.websites.Get(getWeb.Name)
	if w == nil {
		starResult, err := StartWebsite(ToWebsiteConfig(getWeb, job.GetPipeline().GetTls()), getWeb.Contents)
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
	job.Name = getWeb.Name
	w := lns.websites.Get(getWeb.Name)
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
	coreJob := core.Jobs.Get(getWeb.Name)
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
	folderPath := filepath.Join(configs.WebsitePath, webAssets[0].WebName)
	err := os.MkdirAll(folderPath, 0755)
	if err != nil {
		return &clientpb.JobStatus{
			ListenerId: lns.ID(),
			Status:     consts.CtrlStatusFailed,
			Error:      err.Error(),
			Job:        job,
		}
	}
	for _, asset := range webAssets {
		filePath := filepath.Join(folderPath, asset.FileName)
		fullWebpath := path.Join(folderPath, filepath.ToSlash(filePath[len(folderPath):]))
		err := os.MkdirAll(filepath.Dir(fullWebpath), os.ModePerm)
		if err != nil {
			return &clientpb.JobStatus{
				ListenerId: lns.ID(),
				Status:     consts.CtrlStatusFailed,
				Ctrl:       consts.CtrlWebUpload,
				Error:      err.Error(),
			}
		}
		err = os.WriteFile(filePath, asset.Content, os.ModePerm)
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
