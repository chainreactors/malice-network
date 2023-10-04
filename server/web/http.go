package web

import (
	"fmt"
	"net/http"
)

type HTTPServer struct {
	port     int
	server   *http.Server
	rootPath string
}

func NewHTTPServer(port int) *HTTPServer {
	return &HTTPServer{
		port: port,
	}
}

func (s *HTTPServer) Start() {
	// 定义 HTTP 请求处理函数
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, World!") // 在浏览器中显示 "Hello, World!"
	})

	// 启动 HTTP 服务器并监听在指定端口
	s.server = &http.Server{Addr: fmt.Sprintf(":%d", s.port)}
	go func() {
		fmt.Printf("Server is running on port %d...\n", s.port)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Error: %v\n", err)
		}
	}()
}

func (s *HTTPServer) Stop() {
	if s.server != nil {
		fmt.Println("Stopping server...")
		err := s.server.Shutdown(nil)
		if err != nil {
			fmt.Printf("Error shutting down server: %v\n", err)
		}
	}
}

func (s *HTTPServer) AddFileRoute(routePath, localFilePath string) {
	http.Handle(routePath, http.FileServer(http.Dir(localFilePath)))
}

func (s *HTTPServer) DeleteFileRoute(routePath string) {
	// 取消注册指定路由的处理器
	http.DefaultServeMux.Handle(routePath, nil)
	http.DefaultServeMux.HandleFunc(routePath, nil)
}
