package listener

import (
	"context"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"github.com/chainreactors/malice-network/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/server/configs"
	"github.com/chainreactors/malice-network/server/core"
	"google.golang.org/grpc"
)

var Listener *listener

func NewListener(cfg *configs.ListenerConfig) error {
	conn, err := grpc.Dial(cfg.ServerAddr, grpc.WithInsecure())
	if err != nil {
		return err
	}

	lis := &listener{
		Rpc:  listenerrpc.NewListenerRPCClient(conn),
		Name: cfg.Name,
		Host: cfg.Host,
		conn: conn,
		cfg:  cfg,
	}

	_, err = lis.Rpc.RegisterListener(context.Background(), &lispb.RegisterListener{
		ListenerId: fmt.Sprintf("%s_%s", lis.Name, lis.Host),
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
	for _, tcp := range lns.cfg.TcpPipelines {
		pipeline, err := StartTcpPipeline(lns.conn, tcp)
		if err != nil {
			logs.Log.Errorf("Failed to start tcp pipeline %s", err)
			continue
		}
		lns.pipelines.Add(pipeline)
	}
}

func (lns *listener) ToProtobuf() *clientpb.Listener {
	return &clientpb.Listener{
		Id:        lns.ID(),
		Pipelines: lns.pipelines.ToProtobuf(),
	}
}
