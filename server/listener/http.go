package listener

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	types "github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/malice-network/helper/implanttypes"
	"io"
	"net"
	"net/http"
	"strconv"
	"sync"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/parser/pulse"
	cryptostream "github.com/chainreactors/malice-network/server/internal/stream"
)

var (
	httpListen        = net.Listen
	httpStartWithCmux = func(pipeline *HTTPPipeline, ln net.Listener, mux *http.ServeMux) error {
		return pipeline.startWithCmux(ln, mux)
	}
)

func NewHttpPipeline(rpc pipelineRPCClient, pipeline *clientpb.Pipeline) (*HTTPPipeline, error) {
	http := pipeline.GetHttp()

	params, err := implanttypes.UnmarshalPipelineParams(http.Params)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal pipeline params: %v", err)
	}

	pp := &HTTPPipeline{
		rpc:            rpc,
		Name:           pipeline.Name,
		Port:           uint16(http.Port),
		Host:           http.Host,
		PipelineConfig: core.FromPipeline(pipeline),
		Headers:        params.Headers,
		CertName:       pipeline.CertName,
		ErrorPage:      []byte(params.ErrorPage),
		BodyPrefix:     []byte(params.BodyPrefix),
		BodySuffix:     []byte(params.BodySuffix),
	}

	return pp, nil
}

type HTTPPipeline struct {
	stateMu   sync.RWMutex
	starting  bool
	startDone chan struct{}
	srv       net.Listener
	rpc       pipelineRPCClient
	Name      string
	Port      uint16
	Host      string
	Enable    bool
	Target    []string
	CertName  string
	*core.PipelineConfig
	Headers    map[string][]string
	ErrorPage  []byte
	BodyPrefix []byte
	BodySuffix []byte
}

func (pipeline *HTTPPipeline) ToProtobuf() *clientpb.Pipeline {
	pipeline.stateMu.RLock()
	enabled := pipeline.Enable
	pipeline.stateMu.RUnlock()
	params := (&implanttypes.PipelineParams{
		Headers:    pipeline.Headers,
		ErrorPage:  string(pipeline.ErrorPage),
		BodyPrefix: string(pipeline.BodyPrefix),
		BodySuffix: string(pipeline.BodySuffix),
	}).String()

	p := &clientpb.Pipeline{
		Name:       pipeline.Name,
		Enable:     enabled,
		Type:       consts.HTTPPipeline,
		ListenerId: pipeline.ListenerID,
		Parser:     pipeline.Parser,
		CertName:   pipeline.CertName,
		Body: &clientpb.Pipeline_Http{
			Http: &clientpb.HTTPPipeline{
				Name:       pipeline.Name,
				ListenerId: pipeline.ListenerID,
				Port:       uint32(pipeline.Port),
				Host:       pipeline.Host,
				Params:     params,
			},
		},
		Tls:        pipeline.TLSConfig.ToProtobuf(),
		Encryption: pipeline.Encryption.ToProtobuf(),
		Secure:     pipeline.SecureConfig.ToProtobuf(),
	}
	return p
}

func (pipeline *HTTPPipeline) ID() string {
	return pipeline.Name
}

func (pipeline *HTTPPipeline) Close() error {
	pipeline.stateMu.Lock()
	pipeline.Enable = false
	if pipeline.starting {
		done := pipeline.startDone
		pipeline.stateMu.Unlock()
		<-done
		return nil
	}
	ln := pipeline.srv
	pipeline.srv = nil
	pipeline.stateMu.Unlock()
	if ln == nil {
		return nil
	}
	if err := ln.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		return err
	}
	return nil
}

func (pipeline *HTTPPipeline) Start() error {
	if !pipeline.beginStart() {
		return nil
	}

	forward, err := core.NewForward(pipeline.rpc, pipeline)
	if err != nil {
		pipeline.abortStart()
		return err
	}
	forward.ListenerId = pipeline.ListenerID
	committed := false
	defer func() {
		if !committed {
			_ = forward.Abort()
			pipeline.abortStart()
		}
	}()

	ln, err := httpListen("tcp", fmt.Sprintf("%s:%d", pipeline.Host, pipeline.Port))
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", pipeline.handler)

	if pipeline.TLSConfig != nil && pipeline.TLSConfig.Enable && pipeline.TLSConfig.Cert != nil {
		err := httpStartWithCmux(pipeline, ln, mux)
		if err != nil {
			_ = ln.Close()
			return err
		}
	} else {
		// 非 TLS 模式，使用原有逻辑
		server := NewHTTPServer(mux)
		core.GoGuarded("http-serve:"+pipeline.Name, func() error {
			if err := serveHTTP(server, ln); err != nil && err != http.ErrServerClosed && !errors.Is(err, net.ErrClosed) {
				return fmt.Errorf("http pipeline %s serve: %w", pipeline.Name, err)
			}
			return nil
		}, pipeline.runtimeErrorHandler("serve loop"))
	}
	if !pipeline.commitStart(ln, forward) {
		_ = ln.Close()
		return nil
	}
	committed = true

	logs.Log.Infof("pipeline.http - start host=%s port=%d parser=%s tls=%t",
		pipeline.Host, pipeline.Port, pipeline.Parser, pipeline.TLSConfig.Enable)
	return nil
}

func (pipeline *HTTPPipeline) enabled() bool {
	pipeline.stateMu.RLock()
	defer pipeline.stateMu.RUnlock()
	return pipeline.Enable
}

func (pipeline *HTTPPipeline) beginStart() bool {
	for {
		pipeline.stateMu.Lock()
		if pipeline.starting {
			done := pipeline.startDone
			pipeline.stateMu.Unlock()
			<-done
			continue
		}
		if pipeline.Enable {
			pipeline.stateMu.Unlock()
			return false
		}
		pipeline.Enable = true
		pipeline.starting = true
		pipeline.startDone = make(chan struct{})
		pipeline.stateMu.Unlock()
		return true
	}
}

func (pipeline *HTTPPipeline) abortStart() {
	pipeline.stateMu.Lock()
	pipeline.starting = false
	pipeline.Enable = false
	done := pipeline.startDone
	pipeline.startDone = nil
	if done != nil {
		close(done)
	}
	pipeline.stateMu.Unlock()
}

func (pipeline *HTTPPipeline) commitStart(ln net.Listener, forward *core.Forward) bool {
	pipeline.stateMu.Lock()
	defer pipeline.stateMu.Unlock()
	if !pipeline.Enable {
		return false
	}
	pipeline.srv = ln
	core.Forwarders.Add(forward)
	core.GoGuarded("http-forward-recv:"+pipeline.Name, func() error {
		for {
			msg, err := forward.Stream.Recv()
			if err != nil {
				if !pipeline.enabled() {
					return nil
				}
				return fmt.Errorf("http pipeline %s forward recv: %w", pipeline.Name, err)
			}
			dispatchForwardTaskRequest("http", pipeline.Name, msg)
		}
	}, pipeline.runtimeErrorHandler("forward recv loop"))
	pipeline.starting = false
	done := pipeline.startDone
	pipeline.startDone = nil
	close(done)
	return true
}

// startWithCmux 使用 cmux 实现 HTTP TLS 和非 TLS 的端口复用
func (pipeline *HTTPPipeline) startWithCmux(ln net.Listener, mux *http.ServeMux) error {
	// 获取 TLS 配置
	tlsConfig, err := certutils.GetTlsConfig(pipeline.TLSConfig.Cert)
	if err != nil {
		return err
	}

	_, err = StartCmuxHTTPListener(ln, tlsConfig, mux, pipeline.runtimeErrorHandler("cmux"))
	if err != nil {
		return err
	}

	return nil
}

func (pipeline *HTTPPipeline) handlePulse(resp http.ResponseWriter, req *http.Request, conn *cryptostream.Conn) {
	p := conn.Parser
	magic, artifactId, err := p.ReadHeader(conn)
	if err != nil {
		logs.Log.Errorf("%v", err)
		return
	}

	builder, err := pipeline.rpc.GetArtifact(context.Background(), &clientpb.Artifact{
		Id:       artifactId,
		Pipeline: pipeline.Name,
		Format:   consts.FormatRaw,
	})
	if err != nil {
		logs.Log.Errorf("not found artifact %d ,%s ", artifactId, err.Error())
		return
	}
	resp.Header().Set("Content-Type", "application/octet-stream")
	resp.Header().Add("Cache-Control", "no-store, no-cache, must-revalidate")
	resp.Header().Set("Content-Length", fmt.Sprintf("%d", len(builder.Bin)+pulse.HeaderLength+1))
	logs.Log.Infof("send artifact %d %s", builder.Id, builder.Name)

	err = p.WritePacket(conn, types.BuildOneSpites(&implantpb.Spite{
		Name: consts.ModuleInit,
		Body: &implantpb.Spite_Init{
			Init: &implantpb.Init{Data: builder.Bin},
		},
	}), magic)
	if err != nil {
		logs.Log.Errorf("%s", err.Error())
		return
	}
}

func (pipeline *HTTPPipeline) handleMalefic(w http.ResponseWriter, r *http.Request, conn *cryptostream.Conn) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	connect, err := core.GetOrReuseConnection(conn, core.PipelineRuntimeKey(pipeline.ListenerID, pipeline.ID()), pipeline.SecureConfig)
	if err != nil {
		pipeline.writeError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	err = connect.HandlerSimplex(ctx, conn)
	if err != nil {
		if !errors.Is(err, io.EOF) {
			logs.Log.Warnf("http pipeline %s handler error from %s: %s", pipeline.Name, r.RemoteAddr, err.Error())
		}
		return
	}
}

func (pipeline *HTTPPipeline) handler(w http.ResponseWriter, r *http.Request) {
	err := core.RunGuarded("http-request:"+pipeline.Name, func() error {
		// 设置自定义响应头
		for key, values := range pipeline.Headers {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		rw := &httpReadWriter{
			body:       r.Body,
			writer:     w,
			remoteAddr: parseRemoteAddr(r.RemoteAddr),
			bodyPrefix: pipeline.BodyPrefix,
			bodySuffix: pipeline.BodySuffix,
		}

		conn, err := pipeline.WrapConn(rw)
		if err != nil {
			return fmt.Errorf("http pipeline %s wrap conn: %w", pipeline.Name, err)
		}

		logs.Log.Debugf("pipeline.http - accept pipeline=%s remote=%s", pipeline.Name, r.RemoteAddr)
		switch conn.Parser.Implant {
		case consts.ImplantMalefic:
			pipeline.handleMalefic(w, r, conn)
		case consts.ImplantPulse:
			pipeline.handlePulse(w, r, conn)
		default:
			return fmt.Errorf("http pipeline %s unsupported implant: %s", pipeline.Name, conn.Parser.Implant)
		}
		return nil
	}, core.LogGuardedError("http-request:"+pipeline.Name))
	if err != nil {
		pipeline.writeError(w, http.StatusInternalServerError, "Internal server error")
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
		if _, err := buf.Write(h.bodyPrefix); err != nil {
			return 0, err
		}
	}
	n, err = buf.Write(p)
	if err != nil {
		return n, err
	}
	if len(h.bodySuffix) > 0 {
		if _, err := buf.Write(h.bodySuffix); err != nil {
			return n, err
		}
	}
	h.writer.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	if _, err := h.writer.Write(buf.Bytes()); err != nil {
		return n, err
	}
	return n, nil
}

func (h *httpReadWriter) Close() error {
	return nil
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

func (pipeline *HTTPPipeline) runtimeErrorHandler(scope string) core.GoErrorHandler {
	label := fmt.Sprintf("http pipeline %s %s", pipeline.Name, scope)
	return core.CombineErrorHandlers(
		core.LogGuardedError(label),
		func(err error) {
			_ = pipeline.Close()
			if core.EventBroker != nil {
				core.EventBroker.Publish(core.Event{
					EventType: consts.EventListener,
					Op:        consts.CtrlPipelineStop,
					Listener:  &clientpb.Listener{Id: pipeline.ListenerID},
					Message:   label,
					Err:       core.ErrorText(err),
					Important: true,
				})
			}
		},
	)
}
