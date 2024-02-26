package listener

import (
	"context"
	"encoding/pem"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/mtls"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"github.com/chainreactors/malice-network/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/server/internal/certs"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"strconv"
)

var (
	Listener *listener
)

func NewListener(cfg *configs.ListenerConfig) error {
	clientCert, clientKey, err := certs.ClientGenerateCertificate("", cfg.Name, 0, certs.ListenerCA, cfg)
	caCertX509, _, err := certs.GetCertificateAuthority()
	caCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCertX509.Raw})
	if err != nil {
		return err
	}
	tlsConfig, err := mtls.GetTLSConfig(string(caCert), string(clientCert), string(clientKey))
	tlsConfig.ServerName = certs.ListenerNamespace
	transportCreds := credentials.NewTLS(tlsConfig)
	options := []grpc.DialOption{
		grpc.WithTransportCredentials(transportCreds),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(consts.ClientMaxReceiveMessageSize)),
	}
	listenerCfg, err := mtls.ReadConfig(cfg.Auth)
	if err != nil {
		return err
	}
	serverAddress := listenerCfg.LHost + ":" + strconv.Itoa(listenerCfg.LPort)
	conn, err := grpc.Dial(serverAddress, options...)
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
	}

	_, err = lis.Rpc.RegisterListener(context.Background(), &lispb.RegisterListener{
		Id: fmt.Sprintf("%s_%s", lis.Name, lis.Host),
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
}

func (lns *listener) ID() string {
	return fmt.Sprintf("%s_%s", lns.Name, lns.Host)
}

func (lns *listener) Start() {
	go lns.Handler()
	for _, tcp := range lns.cfg.TcpPipelines {
		pipeline, err := StartTcpPipeline(lns.conn, tcp)
		if err != nil {
			logs.Log.Errorf("Failed to start tcp pipeline %s", err)
			continue
		}
		logs.Log.Importantf("Started tcp pipeline %s", pipeline.ID())
		lns.registerPipeline(pipeline)
	}
}

func (lns *listener) ToProtobuf() *clientpb.Listener {
	return &clientpb.Listener{
		Id:        lns.ID(),
		Pipelines: lns.pipelines.ToProtobuf(),
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
		}
		err = stream.Send(resp)
		if err != nil {
			return
		}
	}
}

func (lns *listener) registerPipeline(pipeline core.Pipeline) {
	lns.pipelines.Add(pipeline)
	lns.Rpc.RegisterPipeline(context.Background(), types.BuildPipeline(pipeline.ToProtobuf()))
}

func (lns *listener) startHandler(job *clientpb.Job) *clientpb.JobStatus {
	var err error
	pipeline := job.GetPipeline()
	switch pipeline.Body.(type) {
	case *lispb.Pipeline_Tcp:
		p := lns.pipelines.Get(pipeline.GetTcp().Name)
		if p == nil {
			tcpPipeline, err := StartTcpPipeline(lns.conn, ToTcpConfig(pipeline.GetTcp()))
			if err != nil {
				break
			}
			lns.registerPipeline(tcpPipeline)
		} else {
			err = p.Start()
			if err != nil {
				break
			}
		}
	}
	if err != nil {
		return &clientpb.JobStatus{
			ListenerId: lns.ID(),
			Ctrl:       consts.CtrlPipelineStart,
			Status:     consts.CtrlStatusFailed,
			Error:      err.Error(),
		}
	}
	return &clientpb.JobStatus{
		ListenerId: lns.ID(),
		Ctrl:       consts.CtrlPipelineStart,
		Status:     consts.CtrlStatusSuccess,
	}
}

func (lns *listener) stopHandler(job *clientpb.Job) *clientpb.JobStatus {
	var err error
	pipeline := job.GetPipeline()
	switch pipeline.Body.(type) {
	case *lispb.Pipeline_Tcp:
		p := lns.pipelines.Get(pipeline.GetTcp().Name)
		err = p.Close()
		if err != nil {
			break
		}

	}
	if err != nil {
		return &clientpb.JobStatus{
			ListenerId: lns.ID(),
			Ctrl:       consts.CtrlPipelineStop,
			Status:     consts.CtrlStatusFailed,
			Error:      err.Error(),
		}
	}
	return &clientpb.JobStatus{
		ListenerId: lns.ID(),
		Ctrl:       consts.CtrlPipelineStop,
		Status:     consts.CtrlStatusSuccess,
	}
}
