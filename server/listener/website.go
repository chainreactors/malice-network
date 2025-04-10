package listener

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/core"
)

type Website struct {
	port     int
	server   *http.Server
	rpc      listenerrpc.ListenerRPCClient
	rootPath string
	Name     string
	Enable   bool
	*core.PipelineConfig
	Content map[string]*clientpb.WebContent
}

func StartWebsite(rpc listenerrpc.ListenerRPCClient, pipeline *clientpb.Pipeline, content map[string]*clientpb.WebContent) (*Website, error) {
	websitePp := pipeline.GetWeb()
	web := &Website{
		port:           int(websitePp.Port),
		rootPath:       websitePp.Root,
		rpc:            rpc,
		PipelineConfig: core.FromProtobuf(pipeline),
		Content:        content,
	}

	if web.Content == nil {
		web.Content = make(map[string]*clientpb.WebContent)
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
	var err error
	tlsConfig, err := certutils.WrapToTlsConfig(w.Tls)
	if err != nil {
		return err
	}
	w.server = &http.Server{Addr: fmt.Sprintf("0.0.0.0:%d", w.port),
		TLSConfig: tlsConfig,
		Handler:   mux,
	}
	go func() {
		if err = w.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logs.Log.Errorf("HTTP Server failed to start: %v", err)
		}
	}()
	w.Enable = true
	return nil
}

func (w *Website) Close() error {
	if w.server != nil {
		logs.Log.Importantf("Stopping server")
		err := w.server.Shutdown(nil)
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
		Body: &clientpb.Pipeline_Web{
			Web: &clientpb.Website{
				Name:       w.Name,
				ListenerId: w.ListenerID,
				Port:       uint32(w.port),
				Root:       w.rootPath,
			},
		},
		Tls: w.Tls.ToProtobuf(),
	}
	return p
}

func (w *Website) websiteContentHandler(resp http.ResponseWriter, req *http.Request) {
	contentPath := strings.TrimPrefix(req.URL.Path, w.rootPath)
	content, ok := w.Content[strings.Trim(contentPath, "/")]
	if !ok {
		logs.Log.Debugf("%s Failed to get content ", req.URL)
		return
	}

	resp.Header().Add("Content-Type", content.ContentType)
	resp.Header().Add("Cache-Control", "no-store, no-cache, must-revalidate")
	resp.Write(content.Content)
}

func (w *Website) AddContent(content *clientpb.WebContent) {
	w.Content[strings.Trim(content.Path, "/")] = content
}
