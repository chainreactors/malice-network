package listener

import (
	"context"
	"errors"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/helper/utils/mtls"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"github.com/chainreactors/malice-network/proto/services/listenerrpc"
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

	_, err = lis.Rpc.RegisterListener(context.Background(), &lispb.RegisterListener{
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
		pipeline := tcpPipeline.ToProtobuf(lis.Name)
		_, err = lis.Rpc.RegisterPipeline(context.Background(), pipeline)
		if err != nil {
			return err
		}
		_, err = lis.Rpc.StartTcpPipeline(context.Background(), &lispb.CtrlPipeline{
			Name:       tcpPipeline.Name,
			ListenerId: lis.Name,
		})
		if err != nil {
			return err
		}
	}

	for _, newWebsite := range cfg.Websites {
		tls, err := newWebsite.TlsConfig.ReadCert()
		if err != nil {
			return err
		}
		addWeb := &lispb.WebsiteAddContent{
			Name:     newWebsite.WebsiteName,
			Contents: map[string]*lispb.WebContent{},
		}
		cPath, _ := filepath.Abs(newWebsite.ContentPath)
		fileIfo, err := os.Stat(cPath)

		if fileIfo.IsDir() {
			_ = types.WebAddDirectory(addWeb, newWebsite.RootPath, cPath)
		} else {
			file, err := os.Open(cPath)
			types.WebAddFile(addWeb, newWebsite.RootPath, types.SniffContentType(file), cPath)
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
			Tls: tls.ToProtobuf(),
		}
		_, err = lis.Rpc.RegisterWebsite(context.Background(), webProtobuf)
		if err != nil {
			return err
		}
		_, err = lis.Rpc.StartWebsite(context.Background(), &lispb.CtrlPipeline{
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

func (lns *listener) ID() string {
	return fmt.Sprintf("%s_%s", lns.Name, lns.Host)
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
	var err error
	pipeline := job.GetPipeline()
	switch pipeline.Body.(type) {
	case *lispb.Pipeline_Tcp:
		p := lns.pipelines.Get(pipeline.GetTcp().Name)
		if p == nil {
			tcpPipeline, err := StartTcpPipeline(lns.conn, pipeline)
			if err != nil {
				return &clientpb.JobStatus{
					ListenerId: lns.ID(),
					Ctrl:       consts.CtrlJobStart,
					Status:     consts.CtrlStatusFailed,
					Error:      err.Error(),
					Job:        job,
				}
			}
			job.Name = tcpPipeline.Name
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
	job.GetPipeline().GetWeb().Enable = true
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
