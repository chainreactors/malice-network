package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"google.golang.org/protobuf/proto"
)

// mockWebshell simulates the webshell's X-Stage endpoints for testing.
type mockWebshell struct {
	register  *implantpb.Register
	sessionID uint32

	mu       sync.Mutex
	handler  func(stage string, body []byte) ([]byte, int) // custom handler

	streamMu     sync.Mutex
	streamFrames [][]byte // length-prefixed frames to send for X-Stage: stream
	streamDelay  time.Duration
}

func newMockWebshell() *mockWebshell {
	return &mockWebshell{
		sessionID: 42,
		register: &implantpb.Register{
			Name:   "test-dll",
			Module: []string{"exec", "upload", "download"},
			Sysinfo: &implantpb.SysInfo{
				Os: &implantpb.Os{
					Name: "Windows",
				},
			},
		},
	}
}

func (m *mockWebshell) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	stage := r.Header.Get("X-Stage")
	body, _ := io.ReadAll(r.Body)

	m.mu.Lock()
	handler := m.handler
	m.mu.Unlock()

	if handler != nil {
		respBody, status := handler(stage, body)
		if status != 0 {
			w.WriteHeader(status)
		}
		if respBody != nil {
			w.Write(respBody)
		}
		return
	}

	switch stage {
	case "status":
		w.Write([]byte(`{"ready":true,"method":"jni","deps_present":false,"bridge_version":"1.0"}`))
	case "init":
		regData, _ := proto.Marshal(m.register)
		sid := make([]byte, 4)
		binary.LittleEndian.PutUint32(sid, m.sessionID)
		w.Write(sid)
		w.Write(regData)
	case "spite":
		// Echo: parse input Spites, modify Name, return
		inSpites := &implantpb.Spites{}
		if len(body) > 0 {
			proto.Unmarshal(body, inSpites)
		}
		outSpites := &implantpb.Spites{}
		for _, s := range inSpites.GetSpites() {
			outSpites.Spites = append(outSpites.Spites, &implantpb.Spite{
				Name:   "resp:" + s.Name,
				TaskId: s.TaskId,
			})
		}
		data, _ := proto.Marshal(outSpites)
		w.Write(data)
	case "stream":
		m.handleStream(w)
	default:
		w.WriteHeader(404)
	}
}

func (m *mockWebshell) setHandler(h func(string, []byte) ([]byte, int)) {
	m.mu.Lock()
	m.handler = h
	m.mu.Unlock()
}

// setStreamFrames configures length-prefixed frames returned by X-Stage: stream.
func (m *mockWebshell) setStreamFrames(frames [][]byte, delay time.Duration) {
	m.streamMu.Lock()
	m.streamFrames = frames
	m.streamDelay = delay
	m.streamMu.Unlock()
}

// handleStream writes pre-configured length-prefixed frames to the response.
func (m *mockWebshell) handleStream(w http.ResponseWriter) {
	m.streamMu.Lock()
	frames := m.streamFrames
	delay := m.streamDelay
	m.streamMu.Unlock()

	if frames == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	flusher, _ := w.(http.Flusher)
	for _, frame := range frames {
		lenBuf := make([]byte, 4)
		binary.BigEndian.PutUint32(lenBuf, uint32(len(frame)))
		w.Write(lenBuf)
		w.Write(frame)
		if flusher != nil {
			flusher.Flush()
		}
		if delay > 0 {
			time.Sleep(delay)
		}
	}
}

func startMockWebshell(t *testing.T) (*httptest.Server, *mockWebshell) {
	t.Helper()
	mock := newMockWebshell()
	srv := httptest.NewServer(mock)
	t.Cleanup(srv.Close)
	return srv, mock
}

func TestChannelConnect(t *testing.T) {
	srv, _ := startMockWebshell(t)
	ch := NewChannel(srv.URL, "")
	defer ch.Close()

	if err := ch.Connect(t.Context()); err != nil {
		t.Fatalf("connect: %v", err)
	}
	// Verify structured status was parsed
	if ch.lastStatus == nil {
		t.Fatal("expected lastStatus to be populated")
	}
	if ch.lastStatus.Method != "jni" {
		t.Errorf("expected method 'jni', got %q", ch.lastStatus.Method)
	}
	if ch.lastStatus.BridgeVersion != "1.0" {
		t.Errorf("expected bridge_version '1.0', got %q", ch.lastStatus.BridgeVersion)
	}
}

func TestChannelConnectLegacy(t *testing.T) {
	srv, mock := startMockWebshell(t)
	// Simulate old webshell returning plain "LOADED"
	mock.setHandler(func(stage string, body []byte) ([]byte, int) {
		if stage == "status" {
			return []byte("LOADED"), 200
		}
		return nil, 404
	})
	ch := NewChannel(srv.URL, "")
	defer ch.Close()

	if err := ch.Connect(t.Context()); err != nil {
		t.Fatalf("legacy connect: %v", err)
	}
	// lastStatus should be nil for legacy responses
	if ch.lastStatus != nil {
		t.Error("expected nil lastStatus for legacy response")
	}
}

func TestChannelConnectNotLoaded(t *testing.T) {
	srv, mock := startMockWebshell(t)
	mock.setHandler(func(stage string, body []byte) ([]byte, int) {
		if stage == "status" {
			return []byte(`{"ready":false,"method":"none","deps_present":false,"bridge_version":"1.0"}`), 200
		}
		return nil, 404
	})

	ch := NewChannel(srv.URL, "")
	defer ch.Close()

	err := ch.Connect(t.Context())
	if err == nil {
		t.Fatal("expected error for not-ready status")
	}
}

func TestChannelConnectNotLoadedLegacy(t *testing.T) {
	srv, mock := startMockWebshell(t)
	mock.setHandler(func(stage string, body []byte) ([]byte, int) {
		if stage == "status" {
			return []byte("NOT_LOADED"), 200
		}
		return nil, 404
	})

	ch := NewChannel(srv.URL, "")
	defer ch.Close()

	err := ch.Connect(t.Context())
	if err == nil {
		t.Fatal("expected error for NOT_LOADED")
	}
}

func TestChannelHandshake(t *testing.T) {
	srv, _ := startMockWebshell(t)
	ch := NewChannel(srv.URL, "")
	defer ch.Close()

	reg, err := ch.Handshake()
	if err != nil {
		t.Fatalf("handshake: %v", err)
	}

	if reg.Name != "test-dll" {
		t.Errorf("expected name 'test-dll', got %q", reg.Name)
	}
	if len(reg.Module) != 3 {
		t.Errorf("expected 3 modules, got %d", len(reg.Module))
	}
	if reg.Sysinfo == nil || reg.Sysinfo.Os == nil || reg.Sysinfo.Os.Name != "Windows" {
		t.Errorf("expected Windows sysinfo, got %+v", reg.Sysinfo)
	}
	if ch.SessionID() != 42 {
		t.Errorf("expected sessionID 42, got %d", ch.SessionID())
	}
}

func TestChannelForward(t *testing.T) {
	srv, _ := startMockWebshell(t)
	ch := NewChannel(srv.URL, "")
	defer ch.Close()

	if _, err := ch.Handshake(); err != nil {
		t.Fatalf("handshake: %v", err)
	}

	resp, err := ch.Forward(1, &implantpb.Spite{Name: "exec"})
	if err != nil {
		t.Fatalf("forward: %v", err)
	}
	if resp.Name != "resp:exec" {
		t.Errorf("expected 'resp:exec', got %q", resp.Name)
	}
}

func TestChannelForwardMultiple(t *testing.T) {
	srv, _ := startMockWebshell(t)
	ch := NewChannel(srv.URL, "")
	defer ch.Close()

	if _, err := ch.Handshake(); err != nil {
		t.Fatalf("handshake: %v", err)
	}

	for i, name := range []string{"exec", "upload", "download"} {
		resp, err := ch.Forward(uint32(i+1), &implantpb.Spite{Name: name})
		if err != nil {
			t.Fatalf("forward %d: %v", i, err)
		}
		expected := "resp:" + name
		if resp.Name != expected {
			t.Errorf("task %d: expected %q, got %q", i+1, expected, resp.Name)
		}
	}
}

func TestChannelCloseIdempotent(t *testing.T) {
	ch := NewChannel("http://localhost:1", "")
	if err := ch.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}
	if err := ch.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}
}

func TestChannelForwardAfterClose(t *testing.T) {
	ch := NewChannel("http://localhost:1", "")
	ch.Close()

	_, err := ch.Forward(1, &implantpb.Spite{Name: "exec"})
	if err == nil {
		t.Fatal("expected error forwarding on closed channel")
	}
}

func TestChannelStreamDispatch(t *testing.T) {
	srv, mock := startMockWebshell(t)

	var callCount int
	var mu sync.Mutex
	mock.setHandler(func(stage string, body []byte) ([]byte, int) {
		if stage == "status" {
			return []byte(`{"ready":true,"method":"jni","deps_present":false,"bridge_version":"1.0"}`), 200
		}
		if stage == "init" {
			regData, _ := proto.Marshal(mock.register)
			sid := make([]byte, 4)
			binary.LittleEndian.PutUint32(sid, mock.sessionID)
			return append(sid, regData...), 200
		}
		if stage == "spite" {
			mu.Lock()
			callCount++
			n := callCount
			mu.Unlock()

			// First call: return a streaming response
			if n <= 3 {
				resp := &implantpb.Spites{
					Spites: []*implantpb.Spite{{
						Name:   "stream-chunk",
						TaskId: 100,
					}},
				}
				data, _ := proto.Marshal(resp)
				return data, 200
			}
			// After that: empty
			empty, _ := proto.Marshal(&implantpb.Spites{})
			return empty, 200
		}
		return nil, 404
	})

	ch := NewChannel(srv.URL, "")
	defer ch.Close()

	respCh := ch.OpenStream(100)
	ch.StartRecvLoop()

	// Wait for poll to deliver a response
	select {
	case spite := <-respCh:
		if spite.Name != "stream-chunk" {
			t.Errorf("expected 'stream-chunk', got %q", spite.Name)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for streamed response")
	}

	ch.CloseStream(100)
}

func TestComputeHMAC(t *testing.T) {
	secret := "test-secret-token-longer-than-32chars"
	token := computeToken(secret)

	// Token should be a 64-char hex string (SHA-256)
	if len(token) != 64 {
		t.Fatalf("expected 64 hex chars, got %d", len(token))
	}
	if _, err := hex.DecodeString(token); err != nil {
		t.Fatalf("token is not valid hex: %v", err)
	}

	// Same call within the same 30s window should produce the same token
	token2 := computeToken(secret)
	if token != token2 {
		t.Error("same-window HMAC should be identical")
	}

	// Different secret should produce different token
	token3 := computeToken("different-secret-also-longer-than32")
	if token == token3 {
		t.Error("different secrets should produce different tokens")
	}
}

func TestHMACWindowTolerance(t *testing.T) {
	secret := "test-secret-token-longer-than-32chars"
	now := time.Now().Unix() / 30

	// Verify that the token matches one of the valid windows (current ±1)
	token := computeToken(secret)

	matched := false
	for w := now - 1; w <= now+1; w++ {
		mac := hmac.New(sha256.New, []byte(secret))
		_ = binary.Write(mac, binary.BigEndian, w)
		expected := hex.EncodeToString(mac.Sum(nil))
		if expected == token {
			matched = true
			break
		}
	}
	if !matched {
		t.Error("HMAC token did not match any valid time window")
	}
}

func TestChannelDeliverDep(t *testing.T) {
	var receivedName string
	var receivedLen int
	srv, mock := startMockWebshell(t)
	mock.setHandler(func(stage string, body []byte) ([]byte, int) {
		if stage == "deps" {
			receivedLen = len(body)
			return []byte("OK:/dev/shm/.jna.jar"), 200
		}
		return nil, 404
	})
	// Also capture the X-Dep-Name header
	origHandler := mock.handler
	mock.setHandler(func(stage string, body []byte) ([]byte, int) {
		return origHandler(stage, body)
	})

	ch := NewChannel(srv.URL, "test-token")
	defer ch.Close()

	fakeJar := []byte("PK\x03\x04fake-jar-content")
	err := ch.DeliverDep(t.Context(), ".jna.jar", fakeJar)
	if err != nil {
		t.Fatalf("deliver dep: %v", err)
	}
	if receivedLen != len(fakeJar) {
		t.Errorf("expected %d bytes delivered, got %d", len(fakeJar), receivedLen)
	}
	_ = receivedName
}

func TestChannelStatusDepsPresent(t *testing.T) {
	srv, mock := startMockWebshell(t)
	mock.setHandler(func(stage string, body []byte) ([]byte, int) {
		if stage == "status" {
			return []byte(`{"ready":true,"method":"jna","deps_present":true,"bridge_version":"1.0"}`), 200
		}
		return nil, 404
	})
	ch := NewChannel(srv.URL, "")
	defer ch.Close()

	if err := ch.Connect(t.Context()); err != nil {
		t.Fatalf("connect: %v", err)
	}
	if ch.lastStatus == nil {
		t.Fatal("expected lastStatus")
	}
	if !ch.lastStatus.DepsPresent {
		t.Error("expected deps_present=true")
	}
	if ch.lastStatus.Method != "jna" {
		t.Errorf("expected method 'jna', got %q", ch.lastStatus.Method)
	}
}

// TestHandshakeRejectsSpiteWrapped verifies that the Go bridge rejects the
// old (buggy) wire format where Register was wrapped inside a Spite message.
// This catches the regression where Rust encoded Spite(Register) instead of
// raw Register protobuf.
func TestHandshakeRejectsSpiteWrapped(t *testing.T) {
	srv, mock := startMockWebshell(t)

	// Override init handler to return Spite-wrapped Register (the buggy format).
	mock.setHandler(func(stage string, body []byte) ([]byte, int) {
		if stage == "status" {
			return []byte(`{"ready":true,"method":"jni","deps_present":false,"bridge_version":"1.0"}`), 200
		}
		if stage == "init" {
			reg := &implantpb.Register{
				Name:   "test-dll",
				Module: []string{"exec"},
			}
			// Wrap in a Spite like the buggy Rust code did.
			spite := &implantpb.Spite{
				TaskId: 0,
				Name:   "register",
				Body:   &implantpb.Spite_Register{Register: reg},
			}
			spiteBytes, _ := proto.Marshal(spite)
			sid := make([]byte, 4)
			binary.LittleEndian.PutUint32(sid, 42)
			return append(sid, spiteBytes...), 200
		}
		return nil, 404
	})

	ch := NewChannel(srv.URL, "")
	defer ch.Close()

	reg, err := ch.Handshake()
	if err == nil && reg.Name == "test-dll" {
		t.Fatal("Spite-wrapped Register should NOT parse correctly as raw Register")
	}
	// Either err != nil or reg.Name != "test-dll" — both indicate the
	// Spite-wrapped format is rejected, which is the correct behavior.
}

// TestHandshakeMultiChunkResponse verifies that if the bridge returns multiple
// response spites (streaming), the Go side can parse them all.
func TestHandshakeMultiChunkResponse(t *testing.T) {
	srv, mock := startMockWebshell(t)

	mock.setHandler(func(stage string, body []byte) ([]byte, int) {
		if stage == "status" {
			return []byte(`{"ready":true,"method":"jni","deps_present":false,"bridge_version":"1.0"}`), 200
		}
		if stage == "init" {
			regData, _ := proto.Marshal(mock.register)
			sid := make([]byte, 4)
			binary.LittleEndian.PutUint32(sid, mock.sessionID)
			return append(sid, regData...), 200
		}
		if stage == "spite" {
			// Simulate a module that returns multiple chunks in one response.
			outSpites := &implantpb.Spites{
				Spites: []*implantpb.Spite{
					{Name: "chunk-1", TaskId: 10},
					{Name: "chunk-2", TaskId: 10},
					{Name: "chunk-3", TaskId: 10},
				},
			}
			data, _ := proto.Marshal(outSpites)
			return data, 200
		}
		return nil, 404
	})

	ch := NewChannel(srv.URL, "")
	defer ch.Close()

	if _, err := ch.Handshake(); err != nil {
		t.Fatalf("handshake: %v", err)
	}

	// Open stream, send request, verify all chunks arrive.
	respCh := ch.OpenStream(10)
	ch.StartRecvLoop()

	received := 0
	timeout := time.After(3 * time.Second)
	for received < 3 {
		select {
		case <-respCh:
			received++
		case <-timeout:
			t.Fatalf("timeout: got %d/3 chunks", received)
		}
	}
}

func TestJitterRange(t *testing.T) {
	base := 1 * time.Second
	minExpected := time.Duration(float64(base) * (1 - jitterFactor))
	maxExpected := time.Duration(float64(base) * (1 + jitterFactor))

	for i := 0; i < 100; i++ {
		j := jitter(base)
		if j < minExpected || j > maxExpected {
			t.Fatalf("jitter out of range: got %v, expected [%v, %v]", j, minExpected, maxExpected)
		}
	}
}

func TestStreamDispatchViaStreamMode(t *testing.T) {
	srv, mock := startMockWebshell(t)

	// Build stream frames: 2 spites for task 200, then connection closes.
	var frames [][]byte
	for i := 0; i < 2; i++ {
		spites := &implantpb.Spites{
			Spites: []*implantpb.Spite{{
				Name:   fmt.Sprintf("stream-data-%d", i),
				TaskId: 200,
			}},
		}
		data, _ := proto.Marshal(spites)
		frames = append(frames, data)
	}
	mock.setStreamFrames(frames, 10*time.Millisecond)

	ch := NewChannel(srv.URL, "")
	defer ch.Close()

	if err := ch.Connect(t.Context()); err != nil {
		t.Fatalf("connect: %v", err)
	}

	respCh := ch.OpenStream(200)
	ch.StartRecvLoop()

	for i := 0; i < 2; i++ {
		select {
		case spite := <-respCh:
			expected := fmt.Sprintf("stream-data-%d", i)
			if spite.Name != expected {
				t.Errorf("frame %d: expected %q, got %q", i, expected, spite.Name)
			}
		case <-time.After(3 * time.Second):
			t.Fatalf("timeout waiting for stream frame %d", i)
		}
	}
	ch.CloseStream(200)
}

func TestStreamFallbackToPoll(t *testing.T) {
	srv, mock := startMockWebshell(t)

	// Stream returns 404 (not supported) — should fall back to poll mode.
	// Leave streamFrames nil so handleStream returns 404.

	var callCount int
	var mu sync.Mutex
	mock.setHandler(func(stage string, body []byte) ([]byte, int) {
		switch stage {
		case "status":
			return []byte(`{"ready":true,"method":"jni","deps_present":false,"bridge_version":"1.0"}`), 200
		case "init":
			regData, _ := proto.Marshal(mock.register)
			sid := make([]byte, 4)
			binary.LittleEndian.PutUint32(sid, mock.sessionID)
			return append(sid, regData...), 200
		case "stream":
			return nil, 404
		case "spite":
			mu.Lock()
			callCount++
			n := callCount
			mu.Unlock()
			if n <= 2 {
				resp := &implantpb.Spites{
					Spites: []*implantpb.Spite{{
						Name:   "poll-data",
						TaskId: 300,
					}},
				}
				data, _ := proto.Marshal(resp)
				return data, 200
			}
			empty, _ := proto.Marshal(&implantpb.Spites{})
			return empty, 200
		}
		return nil, 404
	})

	ch := NewChannel(srv.URL, "")
	defer ch.Close()

	if err := ch.Connect(t.Context()); err != nil {
		t.Fatalf("connect: %v", err)
	}

	respCh := ch.OpenStream(300)
	ch.StartRecvLoop()

	select {
	case spite := <-respCh:
		if spite.Name != "poll-data" {
			t.Errorf("expected 'poll-data', got %q", spite.Name)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for poll fallback data")
	}
	ch.CloseStream(300)
}

func TestStreamHeartbeatFrame(t *testing.T) {
	srv, mock := startMockWebshell(t)

	// Send: heartbeat (0-len frame), real data, heartbeat.
	spites := &implantpb.Spites{
		Spites: []*implantpb.Spite{{Name: "after-heartbeat", TaskId: 400}},
	}
	data, _ := proto.Marshal(spites)
	mock.setStreamFrames([][]byte{
		{},   // zero-length heartbeat
		data, // real frame
		{},   // another heartbeat
	}, 10*time.Millisecond)

	ch := NewChannel(srv.URL, "")
	defer ch.Close()

	if err := ch.Connect(t.Context()); err != nil {
		t.Fatalf("connect: %v", err)
	}

	respCh := ch.OpenStream(400)
	ch.StartRecvLoop()

	select {
	case spite := <-respCh:
		if spite.Name != "after-heartbeat" {
			t.Errorf("expected 'after-heartbeat', got %q", spite.Name)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for data after heartbeat")
	}
	ch.CloseStream(400)
}

func TestReadFrameOversized(t *testing.T) {
	// Frame length exceeding streamFrameMaxSize should error.
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, streamFrameMaxSize+1)
	r := bytes.NewReader(lenBuf)
	_, err := readFrame(r)
	if err == nil {
		t.Fatal("expected error for oversized frame")
	}
}
