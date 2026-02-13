package listener

import (
	"crypto/tls"
	"net"
	"net/http"

	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/soheilhy/cmux"
)

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
func StartCmuxTCPListener(ln net.Listener, tlsConfig *tls.Config, handleConn func(net.Conn)) (net.Listener, error) {
	m, tlsL, plainL := SpiltTLSListener(ln)

	// 为 TLS 连接创建 TLS listener
	if tlsConfig != nil {
		tlsL = tls.NewListener(tlsL, tlsConfig)
	}

	// 启动 TLS 连接处理
	core.SafeGo(func() {
		for {
			conn, err := tlsL.Accept()
			if err != nil {
				continue
			}
			core.SafeGo(func() { handleConn(conn) })
		}
	})

	// 启动非 TLS 连接处理
	core.SafeGo(func() {
		for {
			conn, err := plainL.Accept()
			if err != nil {
				continue
			}
			core.SafeGo(func() { handleConn(conn) })
		}
	})

	// 启动 cmux
	core.SafeGo(func() { m.Serve() })

	return ln, nil
}

// StartCmuxHTTPListener 启动支持 TLS 和非 TLS 端口复用的 HTTP listener
func StartCmuxHTTPListener(ln net.Listener, tlsConfig *tls.Config, handler http.Handler) (net.Listener, error) {
	m, tlsL, httpL := SpiltTLSListener(ln)

	httpsServer := NewHTTPServer(handler)
	httpServer := NewHTTPServer(handler)

	// 为 TLS 连接包装 TLS listener
	if tlsConfig != nil {
		tlsL = tls.NewListener(tlsL, tlsConfig)
	}

	// 对于 HTTPS，使用包装后的 TLS listener
	core.SafeGo(func() { httpsServer.Serve(tlsL) })

	// 对于 HTTP，使用普通 listener
	core.SafeGo(func() { httpServer.Serve(httpL) })

	core.SafeGo(func() { m.Serve() })

	return ln, nil
}
