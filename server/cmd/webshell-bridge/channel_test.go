package main

import (
	"encoding/binary"
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
		w.Write([]byte("LOADED"))
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
	default:
		w.WriteHeader(404)
	}
}

func (m *mockWebshell) setHandler(h func(string, []byte) ([]byte, int)) {
	m.mu.Lock()
	m.handler = h
	m.mu.Unlock()
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
}

func TestChannelConnectNotLoaded(t *testing.T) {
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
			return []byte("LOADED"), 200
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
