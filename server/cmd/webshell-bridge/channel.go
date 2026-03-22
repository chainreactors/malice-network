package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/logs"
	malefic "github.com/chainreactors/malice-network/server/internal/parser/malefic"
)

const (
	handshakeBridgeID uint32 = 0
	connectTimeout           = 10 * time.Second
	streamChanBuffer         = 16
)

// Channel manages the malefic protocol connection to the bind DLL on the target.
// The bridge binary acts as a client, the DLL acts as a malefic bind server.
// Communication tunnels through suo5: transport.Dial -> suo5 HTTP -> target localhost.
//
// Streaming task support: OpenStream registers a per-taskID response channel
// that persists across multiple DLL responses. recvLoop dispatches incoming
// packets to the correct channel without removing it, enabling PTY, bridge-agent,
// and other streaming modules. CloseStream or CloseAllStreams handles cleanup.
type Channel struct {
	conn      net.Conn
	transport dialTransport
	dllAddr   string

	sessionID uint32 // malefic session ID from DLL's first frame
	parser    *malefic.MaleficParser

	writeMu  sync.Mutex                    // serializes writes to conn
	pending  map[uint32]chan *implantpb.Spite // taskID -> response channel
	pendMu   sync.Mutex                    // guards pending map
	closed   bool
	closeMu  sync.Mutex
	recvDone chan struct{} // closed when recvLoop exits
	recvErr  error        // terminal error from recvLoop
}

type dialTransport interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// NewChannel creates a channel that will connect through the given transport
// to the DLL's malefic bind server at dllAddr (e.g. "127.0.0.1:13338").
func NewChannel(transport dialTransport, dllAddr, pipelineName string) *Channel {
	return &Channel{
		transport: transport,
		dllAddr:   dllAddr,
		pending:   make(map[uint32]chan *implantpb.Spite),
		recvDone:  make(chan struct{}),
		parser:    malefic.NewMaleficParser(),
	}
}

// Connect dials the DLL through suo5. No handshake is performed here;
// malefic bind sends the Register frame immediately after TCP connect.
func (c *Channel) Connect(ctx context.Context) error {
	dialCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	conn, err := c.transport.DialContext(dialCtx, "tcp", c.dllAddr)
	if err != nil {
		if dialCtx.Err() != nil {
			return fmt.Errorf("connect timeout: %w", dialCtx.Err())
		}
		return fmt.Errorf("dial DLL at %s: %w", c.dllAddr, err)
	}
	c.conn = conn
	return nil
}

// Handshake reads the initial registration data from the DLL.
// The malefic bind DLL sends a frame containing Spites{[Spite{Body: Register{...}}]}
// immediately after TCP connect.
func (c *Channel) Handshake() (*implantpb.Register, error) {
	sid, length, err := c.parser.ReadHeader(c.conn)
	if err != nil {
		return nil, fmt.Errorf("read handshake header: %w", err)
	}

	buf := make([]byte, length)
	if _, err := io.ReadFull(c.conn, buf); err != nil {
		return nil, fmt.Errorf("read handshake payload: %w", err)
	}

	spites, err := c.parser.Parse(buf)
	if err != nil {
		return nil, fmt.Errorf("parse handshake: %w", err)
	}

	if len(spites.GetSpites()) == 0 {
		return nil, fmt.Errorf("empty handshake frame")
	}

	spite := spites.GetSpites()[0]
	reg := spite.GetRegister()
	if reg == nil {
		return nil, fmt.Errorf("handshake spite has no Register body")
	}

	c.sessionID = sid
	logs.Log.Debugf("handshake received: sid=%d name=%s modules=%v", sid, reg.Name, reg.Module)
	return reg, nil
}

// StartRecvLoop starts a background goroutine that reads responses from the
// DLL and dispatches them to the appropriate pending channel by taskID.
// Unlike the old single-response model, channels are NOT removed on first
// dispatch — they persist until explicitly closed via CloseStream.
// Must be called after Connect + Handshake.
func (c *Channel) StartRecvLoop() {
	go c.recvLoop()
}

func (c *Channel) recvLoop() {
	defer close(c.recvDone)
	for {
		_, length, err := c.parser.ReadHeader(c.conn)
		if err != nil {
			c.handleRecvLoopExit(err)
			return
		}

		buf := make([]byte, length)
		if _, err := io.ReadFull(c.conn, buf); err != nil {
			c.handleRecvLoopExit(fmt.Errorf("read payload: %w", err))
			return
		}

		spites, err := c.parser.Parse(buf)
		if err != nil {
			logs.Log.Debugf("recv loop parse error (skipping frame): %v", err)
			continue
		}

		// Dispatch each Spite by its TaskId — do NOT delete the channel entry.
		for _, spite := range spites.GetSpites() {
			taskID := spite.GetTaskId()
			c.pendMu.Lock()
			ch, ok := c.pending[taskID]
			c.pendMu.Unlock()

			if ok {
				select {
				case ch <- spite:
				default:
					logs.Log.Debugf("recv loop: channel full for task %d, dropping", taskID)
				}
			} else {
				logs.Log.Debugf("recv loop: no waiter for task %d", taskID)
			}
		}
	}
}

func (c *Channel) handleRecvLoopExit(err error) {
	c.closeMu.Lock()
	closed := c.closed
	c.closeMu.Unlock()
	if !closed {
		logs.Log.Debugf("recv loop error: %v", err)
	}
	// Close all pending channels to signal EOF to waiters.
	c.pendMu.Lock()
	c.recvErr = err
	for id, ch := range c.pending {
		close(ch)
		delete(c.pending, id)
	}
	c.pendMu.Unlock()
}

// OpenStream registers a buffered response channel for taskID and returns the read end.
// The channel receives all DLL responses for this taskID until CloseStream is called
// or the recvLoop exits (which closes the channel).
func (c *Channel) OpenStream(taskID uint32) <-chan *implantpb.Spite {
	ch := make(chan *implantpb.Spite, streamChanBuffer)
	c.pendMu.Lock()
	c.pending[taskID] = ch
	c.pendMu.Unlock()
	return ch
}

// SendSpite sends a single spite to the DLL for the given taskID.
// Thread-safe: multiple goroutines can call SendSpite concurrently.
func (c *Channel) SendSpite(taskID uint32, spite *implantpb.Spite) error {
	c.closeMu.Lock()
	if c.closed || c.conn == nil {
		c.closeMu.Unlock()
		return fmt.Errorf("channel closed")
	}
	c.closeMu.Unlock()

	spite.TaskId = taskID
	spites := &implantpb.Spites{Spites: []*implantpb.Spite{spite}}

	data, err := c.parser.Marshal(spites, c.sessionID)
	if err != nil {
		return fmt.Errorf("marshal spite: %w", err)
	}

	c.writeMu.Lock()
	_, err = c.conn.Write(data)
	c.writeMu.Unlock()
	return err
}

// CloseStream removes the pending channel for taskID.
// Does NOT close the channel itself to avoid send-on-closed-channel panic
// if recvLoop is concurrently dispatching.
func (c *Channel) CloseStream(taskID uint32) {
	c.pendMu.Lock()
	delete(c.pending, taskID)
	c.pendMu.Unlock()
}

// CloseAllStreams closes and removes all pending channels.
// Safe to call during teardown (holds pendMu for the entire operation).
func (c *Channel) CloseAllStreams() {
	c.pendMu.Lock()
	for id, ch := range c.pending {
		close(ch)
		delete(c.pending, id)
	}
	c.pendMu.Unlock()
}

// Forward sends a Spite request to the DLL and waits for a single response.
// Convenience wrapper over OpenStream + SendSpite + CloseStream for unary tasks.
func (c *Channel) Forward(taskID uint32, spite *implantpb.Spite) (*implantpb.Spite, error) {
	ch := c.OpenStream(taskID)

	if err := c.SendSpite(taskID, spite); err != nil {
		c.CloseStream(taskID)
		return nil, err
	}

	resp, ok := <-ch
	c.CloseStream(taskID)
	if !ok {
		return nil, fmt.Errorf("channel closed during forward")
	}
	return resp, nil
}

// WithSecure enables Age encryption/decryption on the malefic wire protocol.
func (c *Channel) WithSecure(keyPair *clientpb.KeyPair) {
	c.parser.WithSecure(keyPair)
}

// Close shuts down the malefic connection.
func (c *Channel) Close() error {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()

	if c.closed {
		return nil
	}
	c.closed = true
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
