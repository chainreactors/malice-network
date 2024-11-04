package repl

import (
	"context"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/utils/mtls"
	"github.com/chainreactors/tui"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"time"
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

	// 初次连接成功后，立即初始化并记录状态
	logs.Log.Info("Initial connection established, initializing state...")
	if err := restoreAndLogState(con, conn, config); err != nil {
		logs.Log.Errorf("Failed to initialize state: %v", err)
		return nil, err
	}

	// 启动协程持续监控连接状态变化
	go monitorConnectionState(con, conn, config)

	return conn, nil
}

func Login(con *Console, config *mtls.ClientConfig) error {
	_, err := Connect(con, config)
	if err != nil {
		logs.Log.Errorf("Failed to connect: %v", err)
		return err
	}
	logs.Log.Importantf("Connected to server %s", config.Address())
	return nil
}

// monitorConnectionState 监控连接状态变化并在自动重连后恢复状态
func monitorConnectionState(con *Console, conn *grpc.ClientConn, config *mtls.ClientConfig) {
	var previousState connectivity.State
	for {
		currentState := conn.GetState()
		if previousState != connectivity.Ready && currentState == connectivity.Ready {
			tui.Down(0)
			logs.Log.Info("Connection re-established, restoring state...")
			if err := restoreAndLogState(con, conn, config); err != nil {
				logs.Log.Errorf("Failed to restore state after reconnect: %v", err)
			}
		}
		// 更新前一个状态，并等待状态变化
		previousState = currentState
		conn.WaitForStateChange(context.Background(), currentState)

		// 等待一段时间以避免高频率的状态检查
		time.Sleep(100 * time.Millisecond)
	}
}

// restoreAndLogState 恢复并记录 Login 的状态
func restoreAndLogState(con *Console, conn *grpc.ClientConn, config *mtls.ClientConfig) error {
	var err error
	con.ServerStatus, err = core.InitServerStatus(conn, config)
	if err != nil {
		logs.Log.Errorf("init server failed : %v", err)
		return err
	}
	go con.EventHandler()

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
	logs.Log.Importantf("%d listeners, %d pipelines, %d clients, %d sessions (%d alive)",
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
