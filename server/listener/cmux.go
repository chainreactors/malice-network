package listener

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/soheilhy/cmux"
)

var serveCMux = func(m cmux.CMux) error {
	return m.Serve()
}

var serveHTTP = func(server *http.Server, ln net.Listener) error {
	return server.Serve(ln)
}

func NewHTTPServer(handler http.Handler) *http.Server {
	return &http.Server{
		Handler: handler,
	}
}

func SpiltTLSListener(ln net.Listener) (cmux.CMux, net.Listener, net.Listener) {
	m := cmux.New(ln)
	tlsL := m.Match(cmux.TLS())
	plainL := m.Match(cmux.Any())
	return m, tlsL, plainL
}

// StartCmuxTCPListener 启动支持 TLS 和非 TLS 端口复用的 TCP listener
func StartCmuxTCPListener(ln net.Listener, tlsConfig *tls.Config, handleConn func(net.Conn), onError core.GoErrorHandler) (net.Listener, error) {
	m, tlsL, plainL := SpiltTLSListener(ln)

	// 为 TLS 连接创建 TLS listener
	if tlsConfig != nil {
		tlsL = tls.NewListener(tlsL, tlsConfig)
	}

	// 启动 TLS 连接处理
	core.GoGuarded("cmux-tcp-tls-accept", func() error {
		return acceptConnLoop("cmux tcp tls", tlsL, handleConn)
	}, core.CombineErrorHandlers(core.LogGuardedError("cmux-tcp-tls-accept"), onError))

	// 启动非 TLS 连接处理
	core.GoGuarded("cmux-tcp-plain-accept", func() error {
		return acceptConnLoop("cmux tcp plain", plainL, handleConn)
	}, core.CombineErrorHandlers(core.LogGuardedError("cmux-tcp-plain-accept"), onError))

	// 启动 cmux
	core.GoGuarded("cmux-tcp-serve", func() error {
		if err := serveCMux(m); err != nil && !errors.Is(err, net.ErrClosed) {
			return fmt.Errorf("cmux tcp serve: %w", err)
		}
		return nil
	}, core.CombineErrorHandlers(core.LogGuardedError("cmux-tcp-serve"), onError))

	return ln, nil
}

// StartCmuxHTTPListener 启动支持 TLS 和非 TLS 端口复用的 HTTP listener
func StartCmuxHTTPListener(ln net.Listener, tlsConfig *tls.Config, handler http.Handler, onError core.GoErrorHandler) (net.Listener, error) {
	m, tlsL, httpL := SpiltTLSListener(ln)

	httpsServer := NewHTTPServer(handler)
	httpServer := NewHTTPServer(handler)

	// 为 TLS 连接包装 TLS listener
	if tlsConfig != nil {
		tlsL = tls.NewListener(tlsL, tlsConfig)
	}

	// 对于 HTTPS，使用包装后的 TLS listener
	core.GoGuarded("cmux-http-https-serve", func() error {
		if err := serveHTTP(httpsServer, tlsL); err != nil && err != http.ErrServerClosed && !errors.Is(err, net.ErrClosed) {
			return fmt.Errorf("cmux https serve: %w", err)
		}
		return nil
	}, core.CombineErrorHandlers(core.LogGuardedError("cmux-http-https-serve"), onError))

	// 对于 HTTP，使用普通 listener
	core.GoGuarded("cmux-http-serve", func() error {
		if err := serveHTTP(httpServer, httpL); err != nil && err != http.ErrServerClosed && !errors.Is(err, net.ErrClosed) {
			return fmt.Errorf("cmux http serve: %w", err)
		}
		return nil
	}, core.CombineErrorHandlers(core.LogGuardedError("cmux-http-serve"), onError))

	core.GoGuarded("cmux-http-mux-serve", func() error {
		if err := serveCMux(m); err != nil && !errors.Is(err, net.ErrClosed) {
			return fmt.Errorf("cmux http mux serve: %w", err)
		}
		return nil
	}, core.CombineErrorHandlers(core.LogGuardedError("cmux-http-mux-serve"), onError))

	return ln, nil
}

func acceptConnLoop(label string, ln net.Listener, handleConn func(net.Conn)) error {
	for {
		conn, err := ln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return fmt.Errorf("%s accept: %w", label, err)
		}
		core.GoGuarded(label+"-conn", func() error {
			handleConn(conn)
			return nil
		}, core.LogGuardedError(label+"-conn"))
	}
}
