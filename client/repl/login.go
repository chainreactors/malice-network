package repl

import (
	"context"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/utils/mtls"
	"google.golang.org/grpc"
)

func Connect(con *Console, config *mtls.ClientConfig) (*grpc.ClientConn, error) {
	options, err := mtls.GetGrpcOptions([]byte(config.CACertificate), []byte(config.Certificate), []byte(config.PrivateKey), config.Type)
	if err != nil {
		return nil, err
	}

	ctx, _ := context.WithTimeout(context.Background(), consts.DefaultTimeout)
	conn, err := grpc.DialContext(ctx, config.Address(), options...)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func Login(con *Console, config *mtls.ClientConfig) error {
	conn, err := Connect(con, config)
	if err != nil {
		logs.Log.Errorf("Failed to connect to server %s: %v\n", config.Address(), err)
		return err
	}
	logs.Log.Info("Initial connection established, initializing state...\n")
	if err := initState(con, conn, config); err != nil {
		return err
	}
	con.ActiveTarget.Background()
	con.App.SwitchMenu(consts.ClientMenu)
	logs.Log.Importantf("Connected to server %s\n", config.Address())
	return nil
}

func initState(con *Console, conn *grpc.ClientConn, config *mtls.ClientConfig) error {
	var err error
	con.ServerStatus, err = core.InitServerStatus(conn, config)
	if err != nil {
		logs.Log.Errorf("init server failed : %v\n", err)
		return err
	}

	// 记录状态信息
	var pipelineCount int
	for _, i := range con.Listeners {
		pipelineCount += len(i.Pipelines.Pipelines)
	}
	var alive int
	for _, i := range con.Sessions {
		if i.IsAlive {
			alive++
		}
	}
	logs.Log.Importantf("%d listeners, %d pipelines, %d clients, %d sessions (%d alive)\n",
		len(con.Listeners), pipelineCount, len(con.Clients), len(con.Sessions), alive)

	return nil
}

func NewConfigLogin(con *Console, yamlFile string) error {
	config, err := mtls.ReadConfig(yamlFile)
	if err != nil {
		return err
	}
	err = Login(con, config)
	if err != nil {
		return err
	}
	err = assets.MvConfig(yamlFile)
	if err != nil {
		return err
	}
	return nil
}
