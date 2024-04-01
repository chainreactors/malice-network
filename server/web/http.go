package web

import (
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/internal/website"
	"net/http"
)

type HTTPServer struct {
	port        int
	server      *http.Server
	rootPath    string
	websiteName string
}

func NewHTTPServer(port int, rootPath, websiteName string) *HTTPServer {
	return &HTTPServer{
		port:        port,
		rootPath:    rootPath,
		websiteName: websiteName,
	}
}

func (s *HTTPServer) Start() {
	// 定义 HTTP 请求处理函数
	http.HandleFunc(s.rootPath, s.websiteContentHandler)

	// 启动 HTTP 服务器并监听在指定端口
	s.server = &http.Server{Addr: fmt.Sprintf(":%d", s.port)}
	go func() {
		logs.Log.Importantf("HTTP Server is running on port %d", s.port)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logs.Log.Errorf("HTTP Server failed to start: %v", err)
		}
	}()
}

func (s *HTTPServer) Stop() {
	if s.server != nil {
		logs.Log.Importantf("Stopping server")
		err := s.server.Shutdown(nil)
		if err != nil {
			logs.Log.Errorf("Error shutting down server: %v", err)
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

func (s *HTTPServer) websiteContentHandler(resp http.ResponseWriter, req *http.Request) {
	content, err := website.GetContent(s.websiteName, req.URL.Path)
	if err != nil {
		logs.Log.Errorf("Failed to get content %s", err)
		return
	}
	resp.Header().Set("Content-Type", content.ContentType)
	resp.Header().Add("Cache-Control", "no-store, no-cache, must-revalidate")
	resp.Write(content.Content)
}
