package repl

import (
	"context"
	"fmt"
	"net"
	"strconv"

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

// InitMCPServer 在命令注册完成后初始化 MCP 服务器
// 该函数应该在所有命令注册完成后调用，避免并发映射访问错误
// MCP 服务器在后台 goroutine 中启动，不会阻塞主流程
func (con *Console) InitMCPServer() {
	go func() {
		// 加载配置
		setting, err := assets.GetSetting()
		if err != nil {
			logs.Log.Errorf("Failed to get setting: %v\n", err)
			return
		}

		// 检查 MCP 是否启用
		if !setting.McpEnable {
			logs.Log.Debug("MCP server is disabled in settings")
			return
		}

		// 解析地址
		host, port, err := parseAddr(setting.McpAddr)
		if err != nil {
			logs.Log.Errorf("Failed to parse MCP address: %v\n", err)
			return
		}

		// 创建并启动 MCP 服务器
		con.NewMCPServer()
		if err = con.MCP.Start(host, port); err != nil {
			logs.Log.Errorf("Failed to start MCP server: %v\n", err)
			return
		}

		logs.Log.Importantf("MCP server started at http://%s:%d/mcp\n", host, port)
	}()
}

// parseAddr 解析 host:port 格式的地址字符串
// 返回主机名、端口号和可能的错误
func parseAddr(addr string) (string, int, error) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return "", 0, fmt.Errorf("invalid address format: %w", err)
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return "", 0, fmt.Errorf("invalid port number: %w", err)
	}

	return host, port, nil
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
