package listener

import (
	"errors"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/listener/encryption"
	"google.golang.org/protobuf/proto"
	"net/http"
	"net/url"
)

type Website struct {
	port        int
	server      *http.Server
	rootPath    string
	websiteName string
	TlsConfig   *configs.TlsConfig
	Content     map[string]*lispb.WebContent
}

func StartWebsite(cfg *configs.WebsiteConfig, content map[string]*lispb.WebContent) (*Website, error) {
	web := &Website{
		port:        int(cfg.Port),
		rootPath:    cfg.RootPath,
		websiteName: cfg.WebsiteName,
		TlsConfig:   cfg.TlsConfig,
		Content:     content,
	}
	err := web.Start()
	if err != nil {
		return nil, err
	}
	return web, nil
}

func (w *Website) ID() string {
	return fmt.Sprintf("%s", w.websiteName)
}

func (w *Website) Addr() string {
	return ""
}

func (w *Website) Start() error {
	http.HandleFunc(w.rootPath, w.websiteContentHandler)
	var err error
	tlsConfig, err := encryption.WrapToTlsConfig(w.TlsConfig)
	if err != nil {
		return err
	}
	w.server = &http.Server{Addr: fmt.Sprintf(":%d", w.port),
		TLSConfig: tlsConfig,
	}
	go func() {
		logs.Log.Importantf("HTTP Server is running on port %d", w.port)
		if err = w.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logs.Log.Errorf("HTTP Server failed to start: %v", err)
		}
	}()
	if err != nil {
		return err
	}
	return nil
}

func (w *Website) Close() error {
	if w.server != nil {
		logs.Log.Importantf("Stopping server")
		err := w.server.Shutdown(nil)
		if err != nil {
			return err
		}
		return nil
	} else {
		return errors.New("server is not running")
	}
}

func (w *Website) ToProtobuf() proto.Message {
	return &lispb.Website{
		ID:       fmt.Sprintf("%s_%d", w.websiteName, w.port),
		Port:     uint32(w.port),
		Name:     w.websiteName,
		RootPath: w.rootPath,
	}
}

func (w *Website) ToTLSProtobuf() proto.Message {
	return &lispb.TLS{
		Cert: w.TlsConfig.CertFile,
		Key:  w.TlsConfig.KeyFile,
	}
}

func ToWebsiteConfig(w *lispb.Website, tls *lispb.TLS) *configs.WebsiteConfig {
	return &configs.WebsiteConfig{
		Port:        uint16(w.Port),
		RootPath:    w.RootPath,
		WebsiteName: w.Name,
		TlsConfig: &configs.TlsConfig{
			Name:     fmt.Sprintf("%s_%v", w.Name, uint16(w.Port)),
			Enable:   true,
			CertFile: tls.Cert,
			KeyFile:  tls.Key,
		},
	}
}

func (w *Website) AddFileRoute(routePath, localFilePath string) {
	http.Handle(routePath, http.FileServer(http.Dir(localFilePath)))
}

func (w *Website) DeleteFileRoute(routePath string) {
	http.DefaultServeMux.Handle(routePath, nil)
	http.DefaultServeMux.HandleFunc(routePath, nil)
}

func (w *Website) websiteContentHandler(resp http.ResponseWriter, req *http.Request) {
	u, err := url.Parse(req.URL.Path)
	if err != nil {
		logs.Log.Errorf("Failed to parse URL: %v", err)
		return
	}
	content, ok := w.Content[u.Path]
	if !ok {
		logs.Log.Errorf("Failed to get content ")
		return
	}
	resp.Header().Set("Content-Type", content.ContentType)
	resp.Header().Add("Cache-Control", "no-store, no-cache, must-revalidate")
	resp.Write(content.Content)
}
