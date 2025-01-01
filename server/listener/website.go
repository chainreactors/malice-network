package listener

import (
	"errors"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/core"
	"net/http"
	"net/url"
	"path"
	"strings"
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
	mux.HandleFunc(path.Join(w.rootPath, "/"), w.websiteContentHandler)
	var err error
	tlsConfig, err := certutils.WrapToTlsConfig(w.Tls)
	if err != nil {
		return err
	}
	w.server = &http.Server{Addr: fmt.Sprintf(":%d", w.port),
		TLSConfig: tlsConfig,
		Handler:   mux,
	}
	go func() {
		logs.Log.Importantf("HTTP Server is running on port %d", w.port)
		if err = w.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logs.Log.Errorf("HTTP Server failed to start: %v", err)
		}
	}()
	return nil
}

func (w *Website) Close() error {
	if w.server != nil {
		logs.Log.Importantf("Stopping server")
		//http.HandleFunc(w.rootPath, nil)
		//http.HandleFunc(w.rootPath+"/", nil)
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
		Body: &clientpb.Pipeline_Web{
			Web: &clientpb.Website{
				Port: uint32(w.port),
				Root: w.rootPath,
			},
		},
		Tls: w.Tls.ToProtobuf(),
	}
	return p
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
	contentPath := strings.TrimRight(u.Path, "/")
	content, ok := w.Content[contentPath]
	if !ok {
		logs.Log.Debugf("%s Failed to get content ", req.URL)
		return
	}
	switch content.ContentType {
	case consts.ImplantPulse:
	default:

	}
	resp.Header().Add("Content-Type", content.ContentType)
	resp.Header().Add("Cache-Control", "no-store, no-cache, must-revalidate")
	resp.Write(content.Content)
}

func (w *Website) AddContent(content *clientpb.WebContent) {
	w.Content[content.Path] = content
}

//func (w *Website) handlePulse() {
//	cry, err := w.Encryption.NewCrypto()
//	if err != nil {
//		logs.Log.Errorf("Failed to create crypto: %v", err)
//		return
//	}
//	parserContent, err := parser.NewParser(content.Parser)
//	magic, artifactId, err := parserContent.ReadHeader(&peek.Conn{Conn: nil, Reader: bufio.NewReader(req.Body)})
//	if err != nil {
//		logs.Log.Errorf("Failed to read header: %v", err)
//		return
//	}
//	builder, err := w.rpc.GetArtifact(context.Background(), &clientpb.Builder{
//		Id: artifactId,
//	})
//	if err != nil {
//		logs.Log.Errorf("not found artifact %d ,%s ", artifactId, err.Error())
//		return
//	} else {
//		logs.Log.Infof("send artifact %d %s", builder.Id, builder.Name)
//	}
//	err = parserContent.WritePacket(&peek.Conn{Conn: nil, Reader: bufio.NewReader(resp)}, types.BuildOneSpites(&implantpb.Spite{
//		Name: consts.ModuleInit,
//		Body: &implantpb.Spite_Init{
//			Init: &implantpb.Init{Data: content.Content},
//		},
//	}), magic)
//}
