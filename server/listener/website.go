package listener

import (
	"errors"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/core"
	"google.golang.org/protobuf/proto"
	"net/http"
	"net/url"
)

type Website struct {
	port        int
	server      *http.Server
	rpc         listenerrpc.ListenerRPCClient
	rootPath    string
	websiteName string
	*core.PipelineConfig
	Content map[string]*clientpb.WebContent
}

func StartWebsite(rpc listenerrpc.ListenerRPCClient, pipeline *clientpb.Pipeline, content map[string]*clientpb.WebContent) (*Website, error) {
	websitePp := pipeline.GetWeb()
	web := &Website{
		port:           int(websitePp.Port),
		rootPath:       websitePp.Root,
		websiteName:    websitePp.ID,
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
	return fmt.Sprintf("%s", w.websiteName)
}

func (w *Website) Addr() string {
	return ""
}

func (w *Website) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc(w.rootPath, w.websiteContentHandler)
	mux.HandleFunc(w.rootPath+"/", w.websiteContentHandler)
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

func (w *Website) ToProtobuf() proto.Message {
	return &clientpb.Website{
		ID:   fmt.Sprintf("%s_%d", w.websiteName, w.port),
		Port: uint32(w.port),
		Root: w.rootPath,
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
		logs.Log.Debugf("Failed to get content ")
		return
	}
	//parserContent.Marshal()
	//cry.Encrypt(content.Content)
	resp.Header().Set("Content-Type", content.ContentType)
	resp.Header().Add("Cache-Control", "no-store, no-cache, must-revalidate")
	resp.Write(content.Content)
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
