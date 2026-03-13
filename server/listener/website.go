package listener

import (
	"context"
	"errors"
	"fmt"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/chainreactors/logs"
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
	mu       sync.RWMutex
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
		w.server = ln
		core.GoGuarded("website-serve:"+w.Name, func() error {
			if err := serveHTTP(server, ln); err != nil && err != http.ErrServerClosed {
				return fmt.Errorf("website %s serve: %w", w.Name, err)
			}
			return nil
		}, core.LogGuardedError("website-serve:"+w.Name))
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

	_, err = StartCmuxHTTPListener(ln, tlsConfig, mux, core.LogGuardedError("website-cmux:"+w.Name))
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
		Parser:     w.Parser,
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
		Tls:        w.TLSConfig.ToProtobuf(),
		Encryption: w.Encryption.ToProtobuf(),
		Secure:     w.SecureConfig.ToProtobuf(),
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
	w.mu.RLock()
	_, ok := w.Artifact[artifactName]
	w.mu.RUnlock()
	if ok {
		name, err := output.Decode(artifactName)
		if err != nil {
			logs.Log.Errorf("failed to decode: %s", err)
			return
		}

		format, _ := output.Decode(formatted)

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
		w.mu.RLock()
		content, ok := w.Content[strings.Trim(contentPath, "/")]
		w.mu.RUnlock()
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
	if content == nil {
		return errors.New("content is nil")
	}
	if content.WebsiteId == "" || content.Id == "" {
		return errors.New("website_id and id are required")
	}

	websiteID, err := fileutils.SanitizeBasename(content.WebsiteId)
	if err != nil {
		return err
	}
	websiteRoot, err := fileutils.SafeJoin(configs.WebsitePath, websiteID)
	if err != nil {
		return err
	}
	contentPath, err := fileutils.SafeJoin(websiteRoot, content.Id)
	if err != nil {
		return err
	}
	if !fileutils.Exist(contentPath) {
		if err := os.MkdirAll(filepath.Dir(contentPath), 0o700); err != nil {
			return err
		}
		if err := os.WriteFile(contentPath, content.Content, 0o600); err != nil {
			return err
		}
	}
	w.mu.Lock()
	w.Content[strings.Trim(content.Path, "/")] = &clientpb.WebContent{
		Path:        content.Path,
		File:        contentPath,
		ContentType: content.ContentType,
	}
	w.mu.Unlock()
	return nil
}
