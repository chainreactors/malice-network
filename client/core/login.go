package core

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	mtls "github.com/chainreactors/IoM-go/mtls"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"google.golang.org/grpc"
)

func Login(con *Console, config *mtls.ClientConfig) error {
	conn, err := mtls.Connect(config)
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
	con.Server, err = NewServer(conn, config)
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
// MCP 默认关闭，需要通过 --mcp 参数或配置文件中设置 mcp_enable: true 来启用
func (con *Console) InitMCPServer() {
	go func() {
		var addr string

		// 优先使用命令行参数
		if con.MCPAddr != "" {
			addr = con.MCPAddr
		} else {
			// 加载配置
			setting, err := assets.GetSetting()
			if err != nil {
				logs.Log.Errorf("Failed to get setting: %v\n", err)
				return
			}

			// 检查 MCP 是否启用
			if !setting.McpEnable {
				logs.Log.Debugf("MCP server is disabled (use --mcp <addr> to enable)\n")
				return
			}
			addr = setting.McpAddr
		}

		// 解析地址
		host, port, err := parseAddr(addr)
		if err != nil {
			logs.Log.Errorf("Failed to parse MCP address: %v\n", err)
			return
		}

		// 查找可用端口
		finalPort, err := findAvailableMCPPort(host, port)
		if err != nil {
			if errors.Is(err, ErrMCPAlreadyRunning) {
				return
			}
			logs.Log.Errorf("Failed to find available port for MCP server: %v\n", err)
			return
		}

		if finalPort != port {
			logs.Log.Warnf("Port %d is occupied, using port %d instead\n", port, finalPort)
		}

		// 创建并启动 MCP 服务器
		con.MCP = NewMCP(con)
		if err = con.MCP.Start(host, finalPort); err != nil {
			logs.Log.Errorf("Failed to start MCP server: %v\n", err)
			return
		}

		logs.Log.Importantf("MCP server started at http://%s:%d/mcp\n", host, finalPort)
	}()
}

// InitLocalRPCServer 在命令注册完成后初始化 Local RPC 服务器
// 该函数应该在所有命令注册完成后调用，避免并发映射访问错误
// Local RPC 服务器在后台 goroutine 中启动，不会阻塞主流程
// Local RPC 默认关闭，需要通过 --rpc 参数或配置文件中设置 localrpc_enable: true 来启用
func (con *Console) InitLocalRPCServer() {
	go func() {
		var addr string

		// 优先使用命令行参数
		if con.RPCAddr != "" {
			addr = con.RPCAddr
		} else {
			// 加载配置
			setting, err := assets.GetSetting()
			if err != nil {
				logs.Log.Errorf("Failed to get setting: %v\n", err)
				return
			}

			// 检查 Local RPC 是否启用
			if !setting.LocalRPCEnable {
				logs.Log.Debugf("Local RPC server is disabled (use --rpc <addr> to enable)\n")
				return
			}
			addr = setting.LocalRPCAddr
		}

		// 启动 Local RPC 服务器
		var err error
		con.LocalRPC, err = NewLocalRPC(con, addr)
		if err != nil {
			logs.Log.Errorf("Failed to start Local RPC server: %v\n", err)
			return
		}

		if con.LocalRPC != nil {
			logs.Log.Importantf("Local RPC server started at %s\n", addr)
		}
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

var ErrMCPAlreadyRunning = errors.New("mcp already running")

// findAvailableMCPPort 查找可用的 MCP 端口
func findAvailableMCPPort(host string, startPort int) (int, error) {
	const maxAttempts = 10

	for i := 0; i < maxAttempts; i++ {
		port := startPort + i
		addr := fmt.Sprintf("%s:%d", host, port)

		listener, err := net.Listen("tcp", addr)
		if err == nil {
			listener.Close()
			return port, nil
		}

		if checkMCPHealth(host, port) {
			logs.Log.Infof("MCP server already running at http://%s:%d/mcp, skipping startup\n", host, port)
			return port, ErrMCPAlreadyRunning
		}

		logs.Log.Debugf("Port %d is occupied but MCP service is not available, trying next port\n", port)
	}

	return 0, fmt.Errorf("failed to find available port after %d attempts", maxAttempts)
}

// checkMCPHealth 检查指定端口上的 MCP 服务是否健康
// 通过 HTTP GET 请求检查 /mcp/sse 端点是否响应
func checkMCPHealth(host string, port int) bool {
	url := fmt.Sprintf("http://%s:%d/mcp/sse", host, port)

	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	return strings.Contains(contentType, "text/event-stream")
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
