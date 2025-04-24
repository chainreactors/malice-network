package listener

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/encoders"
	"github.com/chainreactors/malice-network/helper/encoders/hash"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/helper/utils/peek"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/parser"
)

func NewHttpPipeline(rpc listenerrpc.ListenerRPCClient, pipeline *clientpb.Pipeline) (*HTTPPipeline, error) {
	http := pipeline.Body.(*clientpb.Pipeline_Http).Http

	// 解析额外参数
	var params types.PipelineParams
	if http.Params != "" {
		if err := json.Unmarshal([]byte(http.Params), &params); err != nil {
			return nil, fmt.Errorf("failed to unmarshal pipeline params: %v", err)
		}
	}

	pp := &HTTPPipeline{
		rpc:            rpc,
		Name:           pipeline.Name,
		Port:           uint16(http.Port),
		Host:           http.Host,
		Target:         pipeline.Target,
		BeaconPipeline: pipeline.BeaconPipeline,
		PipelineConfig: core.FromProtobuf(pipeline),
		Headers:        params.Headers,
		ErrorPage:      []byte(params.ErrorPage),
		BodyPrefix:     []byte(params.BodyPrefix),
		BodySuffix:     []byte(params.BodySuffix),
	}
	var err error
	pp.parser, err = parser.NewParser(pp.Parser)
	if err != nil {
		return nil, err
	}

	return pp, nil
}

type HTTPPipeline struct {
	srv            *http.Server
	rpc            listenerrpc.ListenerRPCClient
	Name           string
	Port           uint16
	Host           string
	Enable         bool
	Target         []string
	BeaconPipeline string
	parser         *parser.MessageParser
	*core.PipelineConfig
	Headers    map[string][]string
	ErrorPage  []byte
	BodyPrefix []byte
	BodySuffix []byte
}

func (pipeline *HTTPPipeline) ToProtobuf() *clientpb.Pipeline {
	p := &clientpb.Pipeline{
		Name:           pipeline.Name,
		Enable:         pipeline.Enable,
		Type:           consts.HTTPPipeline,
		ListenerId:     pipeline.ListenerID,
		Parser:         pipeline.Parser,
		Target:         pipeline.Target,
		BeaconPipeline: pipeline.BeaconPipeline,
		Body: &clientpb.Pipeline_Http{
			Http: &clientpb.HTTPPipeline{
				Name:       pipeline.Name,
				ListenerId: pipeline.ListenerID,
				Port:       uint32(pipeline.Port),
				Host:       pipeline.Host,
			},
		},
		Tls:        pipeline.Tls.ToProtobuf(),
		Encryption: pipeline.Encryption.ToProtobuf(),
	}
	return p
}

func (pipeline *HTTPPipeline) ID() string {
	return pipeline.Name
}

func (pipeline *HTTPPipeline) Close() error {
	pipeline.Enable = false
	if pipeline.srv != nil {
		return pipeline.srv.Close()
	}
	return nil
}

func (pipeline *HTTPPipeline) Start() error {
	if pipeline.Enable {
		return nil
	}
	forward, err := core.NewForward(pipeline.rpc, pipeline)
	if err != nil {
		return err
	}
	forward.ListenerId = pipeline.ListenerID
	core.Forwarders.Add(forward)
	go func() {
		defer logs.Log.Errorf("forwarder stream exit!!!")
		for {
			msg, err := forward.Stream.Recv()
			if err != nil {
				return
			}
			connect := core.Connections.Get(msg.Session.SessionId)
			if connect == nil {
				logs.Log.Errorf("connection %s not found", msg.Session.SessionId)
				continue
			}
			connect.C <- msg
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/", pipeline.handler)

	pipeline.srv = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", pipeline.Host, pipeline.Port),
		Handler: mux,
	}

	if pipeline.Tls != nil && pipeline.Tls.Enable {
		tlsConfig, err := certutils.GetTlsConfig(pipeline.Tls.ToProtobuf())
		if err != nil {
			return err
		}
		pipeline.srv.TLSConfig = tlsConfig
		go func() {
			if err := pipeline.srv.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
				logs.Log.Errorf("HTTPS server error: %v", err)
			}
		}()
	} else {
		go func() {
			if err := pipeline.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logs.Log.Errorf("HTTP server error: %v", err)
			}
		}()
	}

	logs.Log.Infof("[pipeline] starting HTTP pipeline on %s:%d, parser: %s, cryptor: %s, tls: %t",
		pipeline.Host, pipeline.Port, pipeline.Parser, pipeline.Encryption.Type, pipeline.Tls.Enable)
	pipeline.Enable = true
	return nil
}

func (pipeline *HTTPPipeline) handler(w http.ResponseWriter, r *http.Request) {
	// 设置自定义响应头
	for key, values := range pipeline.Headers {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	ctx, _ := context.WithCancel(r.Context())

	rw := &httpReadWriter{
		body:       r.Body,
		writer:     w,
		remoteAddr: parseRemoteAddr(r.RemoteAddr),
		bodyPrefix: pipeline.BodyPrefix,
		bodySuffix: pipeline.BodySuffix,
	}
	wrappedConn, err := pipeline.WrapConn(rw)
	if err != nil {
		pipeline.writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	connect, err := pipeline.getConnection(wrappedConn)
	if err != nil {
		pipeline.writeError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	err = connect.HandlerSimplex(ctx, wrappedConn)
	if err != nil {
		if !errors.Is(err, io.EOF) {
			logs.Log.Debugf("handler error: %s", err.Error())
		}
		return
	}
}

func (pipeline *HTTPPipeline) getConnection(conn *peek.Conn) (*core.Connection, error) {
	p := pipeline.parser
	sid, _, err := p.PeekHeader(conn)
	if err != nil {
		return nil, err
	}

	if newC := core.Connections.Get(hash.Md5Hash(encoders.Uint32ToBytes(sid))); newC != nil {
		return newC, nil
	} else {
		newC := core.NewConnection(p, sid, pipeline.ID())
		core.Connections.Add(newC)
		return newC, nil
	}
}

// httpReadWriter 实现了io.ReadWriteCloser接口，用于处理HTTP请求和响应
type httpReadWriter struct {
	body       io.Reader
	offset     int
	writer     http.ResponseWriter
	remoteAddr net.Addr
	bodyPrefix []byte
	bodySuffix []byte
}

func (h *httpReadWriter) Read(p []byte) (n int, err error) {
	return h.body.Read(p)
}

func (h *httpReadWriter) Write(p []byte) (n int, err error) {
	var buf bytes.Buffer
	if len(h.bodyPrefix) > 0 {
		buf.Write(h.bodyPrefix)
	}
	n, err = buf.Write(p)
	if err != nil {
		return n, err
	}

	if len(h.bodySuffix) > 0 {
		buf.Write(h.bodySuffix)
	}
	//h.writer.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	if _, err := h.writer.Write(buf.Bytes()); err != nil {
		return n, err
	}
	return n, nil
}

func (h *httpReadWriter) Close() error {
	return h.Close()
}

func (h *httpReadWriter) RemoteAddr() net.Addr {
	return h.remoteAddr
}

func parseRemoteAddr(remoteAddr string) net.Addr {
	// 分割 IP 和端口
	ipStr, portStr, _ := net.SplitHostPort(remoteAddr)

	ip := net.ParseIP(ipStr)

	port, _ := strconv.Atoi(portStr)

	// 创建 TCPAddr（实现了 net.Addr 接口）
	return &net.TCPAddr{
		IP:   ip,
		Port: port,
		Zone: "",
	}
}

// writeError 处理HTTP错误响应
func (pipeline *HTTPPipeline) writeError(w http.ResponseWriter, statusCode int, defaultMessage string) {
	if pipeline.ErrorPage != nil {
		w.WriteHeader(statusCode)
		w.Write(pipeline.ErrorPage)
	} else {
		http.Error(w, defaultMessage, statusCode)
	}
}
