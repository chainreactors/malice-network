package listener

import (
	"context"
	"errors"
	"fmt"
	"github.com/chainreactors/malice-network/helper/utils/formatutils"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"strings"
)

type Website struct {
	port     int
	server   net.Listener
	rpc      listenerrpc.ListenerRPCClient
	rootPath string
	Name     string
	Enable   bool
	CertName string
	*core.PipelineConfig
	Content  map[string]*clientpb.WebContent
	Artifact map[string]*clientpb.WebContent
}

func StartWebsite(rpc listenerrpc.ListenerRPCClient, pipeline *clientpb.Pipeline, content map[string]*clientpb.WebContent) (*Website, error) {
	websitePp := pipeline.GetWeb()
	web := &Website{
		Name:           pipeline.Name,
		port:           int(websitePp.Port),
		rootPath:       websitePp.Root,
		rpc:            rpc,
		CertName:       pipeline.CertName,
		PipelineConfig: core.FromPipeline(pipeline),
		Content:        make(map[string]*clientpb.WebContent),
		Artifact:       make(map[string]*clientpb.WebContent),
	}
	for _, c := range content {
		err := web.AddContent(c)
		if err != nil {
			return nil, err
		}
	}
	err := web.Start()
	if err != nil {
		return nil, err
	}
	return web, nil
}

func (w *Website) ID() string {
	return w.Name
}

func (w *Website) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc(certutils.ACMERootPath, w.acmeChallengeHandler)
	mux.HandleFunc(w.rootPath, w.websiteContentHandler)

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", w.port))
	if err != nil {
		return err
	}
	// 如果启用了 TLS，使用 cmux 实现 TLS 和非 TLS 的端口复用
	if w.TLSConfig != nil && w.TLSConfig.Enable && w.TLSConfig.Cert != nil {
		err := w.startWithCmux(ln, mux)
		if err != nil {
			return err
		}
	} else {
		server := NewHTTPServer(mux)
		go func() {
			if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
				logs.Log.Errorf("HTTP Server failed to start: %v", err)
			}
		}()
	}

	w.Enable = true
	return nil
}

// startWithCmux 使用 cmux 实现 Website TLS 和非 TLS 的端口复用
func (w *Website) startWithCmux(ln net.Listener, mux *http.ServeMux) error {
	// 获取 TLS 配置
	tlsConfig, err := certutils.GetTlsConfig(w.TLSConfig.Cert)
	if err != nil {
		return err
	}

	_, err = StartCmuxHTTPListener(ln, tlsConfig, mux)
	if err != nil {
		return err
	}

	// 保存服务器引用用于关闭
	w.server = ln

	return nil
}

func (w *Website) Close() error {
	if w.server != nil {
		logs.Log.Importantf("Stopping server")
		err := w.server.Close()
		if err != nil {
			return err
		}
		w.server = nil
		return nil
	} else {
		return errors.New("server is not running")
	}
}

func (w *Website) ToProtobuf() *clientpb.Pipeline {
	p := &clientpb.Pipeline{
		Name:       w.Name,
		Enable:     w.Enable,
		ListenerId: w.ListenerID,
		Type:       consts.WebsitePipeline,
		CertName:   w.CertName,
		Body: &clientpb.Pipeline_Web{
			Web: &clientpb.Website{
				Name:       w.Name,
				ListenerId: w.ListenerID,
				Port:       uint32(w.port),
				Root:       w.rootPath,
			},
		},
		Tls: w.TLSConfig.ToProtobuf(),
	}
	return p
}

func (w *Website) websiteContentHandler(resp http.ResponseWriter, req *http.Request) {
	contentPath := strings.TrimPrefix(req.URL.Path, w.rootPath)
	parts := strings.SplitN(contentPath, "/", 2)
	var artifactName string
	var formatted string
	artifactName = parts[0]
	if len(parts) == 2 {
		formatted = parts[1]
	}
	_, ok := w.Artifact[artifactName]
	if ok {
		name, err := formatutils.Decode(artifactName)
		if err != nil {
			logs.Log.Errorf("failed to decode: %s", err)
			return
		}

		format, _ := formatutils.Decode(formatted)

		artifact, err := w.rpc.FindArtifact(context.Background(), &clientpb.Artifact{
			Name:   name,
			Format: format,
		})

		if err != nil {
			logs.Log.Errorf("failed to find artifact: %s", err)
			return
		}

		if len(artifact.Bin) > 0 {
			resp.Write(artifact.Bin)
		}
	} else {
		content, ok := w.Content[strings.Trim(contentPath, "/")]
		if !ok {
			logs.Log.Debugf("%s failed to get content ", req.URL)
			return
		}

		resp.Header().Add("Content-Type", content.ContentType)
		resp.Header().Add("Cache-Control", "no-store, no-cache, must-revalidate")
		data, err := os.ReadFile(content.File)
		if err != nil {
			return
		}

		resp.Write(data)
	}
}

func (w *Website) AddContent(content *clientpb.WebContent) error {
	contentPath := filepath.Join(configs.WebsitePath, content.WebsiteId, content.Id)
	if !fileutils.Exist(contentPath) {
		err := os.WriteFile(contentPath, content.Content, 0644)
		if err != nil {
			return err
		}
	}
	w.Content[strings.Trim(content.Path, "/")] = &clientpb.WebContent{
		Path:        content.Path,
		File:        contentPath,
		ContentType: content.ContentType,
	}
	return nil
}

// acmeChallengeHandler
func (w *Website) acmeChallengeHandler(resp http.ResponseWriter, req *http.Request) {
	if certutils.GetACMEManager() == nil {
		http.Error(resp, "ACME not enabled", http.StatusNotFound)
		return
	}
	certutils.GetACMEManager().GetManager().HTTPHandler(nil)
}
