package main

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"sync"
	"time"

	proxysuo5 "github.com/chainreactors/proxyclient/suo5"
	suo5core "github.com/zema1/suo5/suo5"
)

// Transport manages the suo5 tunnel connection to the target webshell.
type Transport struct {
	rawURL *url.URL
	mu     sync.Mutex
	client *proxysuo5.Suo5Client
}

// NewTransport creates a transport adapter for the given suo5 URL.
// Supported schemes: suo5:// (HTTP), suo5s:// (HTTPS).
func NewTransport(rawURL string) (*Transport, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse suo5 URL: %w", err)
	}
	if u.Scheme != "suo5" && u.Scheme != "suo5s" {
		return nil, fmt.Errorf("unsupported suo5 scheme: %s", u.Scheme)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("missing suo5 host")
	}

	return &Transport{
		rawURL: u,
	}, nil
}

// Dial establishes a TCP connection through the suo5 tunnel to the given address.
// The returned net.Conn transparently tunnels through the webshell's HTTP channel.
func (t *Transport) Dial(network, address string) (net.Conn, error) {
	return t.DialContext(context.Background(), network, address)
}

// DialContext establishes a TCP connection through the suo5 tunnel and binds
// the initial HTTP request to ctx so cancellation interrupts the dial.
func (t *Transport) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	if err := t.initClient(); err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	switch network {
	case "", "tcp", "tcp4", "tcp6":
	default:
		return nil, fmt.Errorf("unsupported network: %s", network)
	}

	conn := &suo5NetConn{
		Suo5Conn: suo5core.NewSuo5Conn(ctx, t.client.Conf.Suo5Client),
	}
	if err := conn.Connect(address); err != nil {
		return nil, err
	}
	return conn, nil
}

func (t *Transport) initClient() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.client != nil {
		return nil
	}
	if t.rawURL == nil {
		return fmt.Errorf("missing suo5 URL")
	}

	conf, err := proxysuo5.NewConfFromURL(t.rawURL)
	if err != nil {
		return fmt.Errorf("init suo5 config: %w", err)
	}
	t.client = &proxysuo5.Suo5Client{
		Proxy: t.rawURL,
		Conf:  conf,
	}
	return nil
}

type suo5NetConn struct {
	*suo5core.Suo5Conn
	remoteAddr string
}

// Write normalizes the return value from the underlying suo5 chunked writer.
// In half-duplex mode the underlying Write wraps data in a frame and returns
// the frame length rather than the original data length. Callers (e.g.
// cio.WriteMsg) expect n == len(p) on success, so we fix it here.
func (conn *suo5NetConn) Write(p []byte) (int, error) {
	n, err := conn.Suo5Conn.Write(p)
	if err != nil {
		return n, err
	}
	if n > len(p) {
		n = len(p)
	}
	return n, nil
}

func (conn *suo5NetConn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)}
}

func (conn *suo5NetConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0}
}

func (conn *suo5NetConn) SetDeadline(_ time.Time) error {
	return nil
}

func (conn *suo5NetConn) SetReadDeadline(_ time.Time) error {
	return nil
}

func (conn *suo5NetConn) SetWriteDeadline(_ time.Time) error {
	return nil
}
