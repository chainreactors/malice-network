package main

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/utils/compress"
	malefic "github.com/chainreactors/malice-network/server/internal/parser/malefic"
	"github.com/gookit/config/v2"
	"google.golang.org/protobuf/proto"
)

func init() {
	// Initialize config for the malefic parser's packet length check.
	config.Set(consts.ConfigMaxPacketLength, 10*1024*1024)
}

// testWriteMaleficFrame writes a malefic-framed message to conn for test use.
func testWriteMaleficFrame(conn net.Conn, spites *implantpb.Spites, sid uint32) error {
	data, err := proto.Marshal(spites)
	if err != nil {
		return err
	}
	data, err = compress.Compress(data)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	buf.WriteByte(malefic.DefaultStartDelimiter)
	binary.Write(&buf, binary.LittleEndian, sid)
	binary.Write(&buf, binary.LittleEndian, int32(len(data)))
	buf.Write(data)
	buf.WriteByte(malefic.DefaultEndDelimiter)
	_, err = conn.Write(buf.Bytes())
	return err
}

// testReadMaleficFrame reads a malefic-framed message from conn for test use.
func testReadMaleficFrame(conn net.Conn) (uint32, *implantpb.Spites, error) {
	header := make([]byte, malefic.HeaderLength)
	if _, err := io.ReadFull(conn, header); err != nil {
		return 0, nil, err
	}
	if header[0] != malefic.DefaultStartDelimiter {
		return 0, nil, io.ErrUnexpectedEOF
	}
	sid := binary.LittleEndian.Uint32(header[1:5])
	length := binary.LittleEndian.Uint32(header[5:9])
	buf := make([]byte, length+1)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return 0, nil, err
	}
	payload := buf[:length]
	decompressed, err := compress.Decompress(payload)
	if err != nil {
		decompressed = payload
	}
	spites := &implantpb.Spites{}
	if err := proto.Unmarshal(decompressed, spites); err != nil {
		return 0, nil, err
	}
	return sid, spites, nil
}

// mockMaleficDLL simulates a malefic bind DLL.
// It accepts one connection, sends a Register handshake frame,
// then echoes Spite requests back with a modified Name field.
type mockMaleficDLL struct {
	ln        net.Listener
	register  *implantpb.Register
	sessionID uint32
}

func newMockMaleficDLL(t *testing.T) *mockMaleficDLL {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("mock DLL listen: %v", err)
	}
	return &mockMaleficDLL{
		ln:        ln,
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

func (m *mockMaleficDLL) addr() string {
	return m.ln.Addr().String()
}

func (m *mockMaleficDLL) close() {
	m.ln.Close()
}

// serve handles one client connection through the full malefic protocol.
func (m *mockMaleficDLL) serve(t *testing.T, handleN int) {
	t.Helper()
	conn, err := m.ln.Accept()
	if err != nil {
		t.Errorf("mock DLL accept: %v", err)
		return
	}
	defer conn.Close()

	// Send Register handshake as malefic frame
	regSpite := &implantpb.Spites{
		Spites: []*implantpb.Spite{
			{
				Body: &implantpb.Spite_Register{Register: m.register},
			},
		},
	}
	if err := testWriteMaleficFrame(conn, regSpite, m.sessionID); err != nil {
		t.Errorf("mock DLL send handshake: %v", err)
		return
	}

	// Echo Spite requests back with modified Name
	for i := 0; i < handleN; i++ {
		sid, spites, err := testReadMaleficFrame(conn)
		if err != nil {
			t.Errorf("mock DLL read spite: %v", err)
			return
		}

		respSpites := &implantpb.Spites{}
		for _, spite := range spites.GetSpites() {
			respSpites.Spites = append(respSpites.Spites, &implantpb.Spite{
				Name:   "resp:" + spite.Name,
				TaskId: spite.TaskId,
			})
		}
		if err := testWriteMaleficFrame(conn, respSpites, sid); err != nil {
			t.Errorf("mock DLL send response: %v", err)
			return
		}
	}
}

// dialMockDLL connects to the mock DLL and returns a Channel ready for Handshake/Forward.
func dialMockDLL(t *testing.T, addr string) *Channel {
	t.Helper()
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		t.Fatalf("dial mock DLL: %v", err)
	}
	return &Channel{
		conn:     conn,
		dllAddr:  addr,
		pending:  make(map[uint32]chan *implantpb.Spite),
		recvDone: make(chan struct{}),
		parser:   malefic.NewMaleficParser(),
	}
}

func TestChannelConnect(t *testing.T) {
	mock := newMockMaleficDLL(t)
	defer mock.close()

	// Accept the connection in background
	accepted := make(chan struct{})
	go func() {
		conn, err := mock.ln.Accept()
		if err != nil {
			return
		}
		conn.Close()
		close(accepted)
	}()

	conn, err := net.DialTimeout("tcp", mock.addr(), 5*time.Second)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	conn.Close()

	select {
	case <-accepted:
	case <-time.After(time.Second):
		t.Fatal("connection not accepted")
	}
}

func TestChannelHandshake(t *testing.T) {
	mock := newMockMaleficDLL(t)
	defer mock.close()

	go mock.serve(t, 0) // Handshake only

	ch := dialMockDLL(t, mock.addr())
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
	if ch.sessionID != 42 {
		t.Errorf("expected sessionID 42, got %d", ch.sessionID)
	}
}

func TestChannelForward(t *testing.T) {
	mock := newMockMaleficDLL(t)
	defer mock.close()

	go mock.serve(t, 2) // Handshake + 2 Spite roundtrips

	ch := dialMockDLL(t, mock.addr())
	defer ch.Close()

	if _, err := ch.Handshake(); err != nil {
		t.Fatalf("handshake: %v", err)
	}

	ch.StartRecvLoop()

	// Forward Spite #1
	resp1, err := ch.Forward(1, &implantpb.Spite{Name: "exec"})
	if err != nil {
		t.Fatalf("forward #1: %v", err)
	}
	if resp1.Name != "resp:exec" {
		t.Errorf("expected 'resp:exec', got %q", resp1.Name)
	}

	// Forward Spite #2
	resp2, err := ch.Forward(2, &implantpb.Spite{Name: "upload"})
	if err != nil {
		t.Fatalf("forward #2: %v", err)
	}
	if resp2.Name != "resp:upload" {
		t.Errorf("expected 'resp:upload', got %q", resp2.Name)
	}
}

func TestChannelForwardBatch(t *testing.T) {
	// Test that recvLoop correctly dispatches a batch response
	// (one malefic frame containing multiple Spites).
	mock := newMockMaleficDLL(t)
	defer mock.close()

	go func() {
		conn, err := mock.ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		// Send handshake
		regSpite := &implantpb.Spites{
			Spites: []*implantpb.Spite{
				{Body: &implantpb.Spite_Register{Register: mock.register}},
			},
		}
		testWriteMaleficFrame(conn, regSpite, mock.sessionID)

		// Read two individual requests
		var requests []*implantpb.Spite
		for i := 0; i < 2; i++ {
			_, spites, err := testReadMaleficFrame(conn)
			if err != nil {
				return
			}
			requests = append(requests, spites.GetSpites()...)
		}

		// Respond with a single batch frame containing both responses
		batchResp := &implantpb.Spites{}
		for _, req := range requests {
			batchResp.Spites = append(batchResp.Spites, &implantpb.Spite{
				Name:   "resp:" + req.Name,
				TaskId: req.TaskId,
			})
		}
		testWriteMaleficFrame(conn, batchResp, mock.sessionID)
	}()

	ch := dialMockDLL(t, mock.addr())
	defer ch.Close()

	if _, err := ch.Handshake(); err != nil {
		t.Fatalf("handshake: %v", err)
	}
	ch.StartRecvLoop()

	// Send two requests concurrently
	var wg sync.WaitGroup
	results := make(map[uint32]string)
	var mu sync.Mutex

	for _, tc := range []struct {
		id   uint32
		name string
	}{
		{10, "exec"},
		{20, "download"},
	} {
		wg.Add(1)
		go func(id uint32, name string) {
			defer wg.Done()
			resp, err := ch.Forward(id, &implantpb.Spite{Name: name})
			if err != nil {
				t.Errorf("forward %d: %v", id, err)
				return
			}
			mu.Lock()
			results[id] = resp.Name
			mu.Unlock()
		}(tc.id, tc.name)
	}

	wg.Wait()

	if results[10] != "resp:exec" {
		t.Errorf("task 10: expected 'resp:exec', got %q", results[10])
	}
	if results[20] != "resp:download" {
		t.Errorf("task 20: expected 'resp:download', got %q", results[20])
	}
}

func TestChannelStreamMultipleResponses(t *testing.T) {
	// Test that OpenStream receives multiple responses for the same taskID
	// without the channel being removed after the first one.
	mock := newMockMaleficDLL(t)
	defer mock.close()

	const taskID uint32 = 100
	const numResponses = 3

	go func() {
		conn, err := mock.ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		// Send handshake
		regSpite := &implantpb.Spites{
			Spites: []*implantpb.Spite{
				{Body: &implantpb.Spite_Register{Register: mock.register}},
			},
		}
		testWriteMaleficFrame(conn, regSpite, mock.sessionID)

		// Read the initial request
		testReadMaleficFrame(conn)

		// Send multiple responses for the same taskID in separate frames
		for i := 0; i < numResponses; i++ {
			resp := &implantpb.Spites{
				Spites: []*implantpb.Spite{
					{
						Name:   "chunk:" + string(rune('A'+i)),
						TaskId: taskID,
					},
				},
			}
			if err := testWriteMaleficFrame(conn, resp, mock.sessionID); err != nil {
				return
			}
		}
	}()

	ch := dialMockDLL(t, mock.addr())
	defer ch.Close()

	if _, err := ch.Handshake(); err != nil {
		t.Fatalf("handshake: %v", err)
	}

	// Open a persistent stream for this task
	respCh := ch.OpenStream(taskID)
	ch.StartRecvLoop()

	// Send the initial request
	if err := ch.SendSpite(taskID, &implantpb.Spite{Name: "start-stream"}); err != nil {
		t.Fatalf("send spite: %v", err)
	}

	// Collect all responses
	var received []string
	for i := 0; i < numResponses; i++ {
		select {
		case spite, ok := <-respCh:
			if !ok {
				t.Fatalf("channel closed after %d responses", i)
			}
			received = append(received, spite.Name)
		case <-time.After(2 * time.Second):
			t.Fatalf("timeout waiting for response %d", i)
		}
	}

	if len(received) != numResponses {
		t.Fatalf("expected %d responses, got %d", numResponses, len(received))
	}
	for i, name := range received {
		expected := "chunk:" + string(rune('A'+i))
		if name != expected {
			t.Errorf("response %d: expected %q, got %q", i, expected, name)
		}
	}

	ch.CloseStream(taskID)
}

func TestChannelCloseStream(t *testing.T) {
	// Verify that CloseStream removes the channel so subsequent dispatches
	// are dropped (logged as "no waiter").
	mock := newMockMaleficDLL(t)
	defer mock.close()

	const taskID uint32 = 200

	go func() {
		conn, err := mock.ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		// Handshake
		regSpite := &implantpb.Spites{
			Spites: []*implantpb.Spite{
				{Body: &implantpb.Spite_Register{Register: mock.register}},
			},
		}
		testWriteMaleficFrame(conn, regSpite, mock.sessionID)

		// Read initial request
		testReadMaleficFrame(conn)

		// Send first response
		testWriteMaleficFrame(conn, &implantpb.Spites{
			Spites: []*implantpb.Spite{{Name: "first", TaskId: taskID}},
		}, mock.sessionID)

		// Small delay for CloseStream to execute
		time.Sleep(100 * time.Millisecond)

		// Send second response (should be dropped after CloseStream)
		testWriteMaleficFrame(conn, &implantpb.Spites{
			Spites: []*implantpb.Spite{{Name: "second", TaskId: taskID}},
		}, mock.sessionID)
	}()

	ch := dialMockDLL(t, mock.addr())
	defer ch.Close()

	if _, err := ch.Handshake(); err != nil {
		t.Fatalf("handshake: %v", err)
	}

	respCh := ch.OpenStream(taskID)
	ch.StartRecvLoop()

	if err := ch.SendSpite(taskID, &implantpb.Spite{Name: "req"}); err != nil {
		t.Fatalf("send: %v", err)
	}

	// Receive first response
	select {
	case spite := <-respCh:
		if spite.Name != "first" {
			t.Errorf("expected 'first', got %q", spite.Name)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for first response")
	}

	// Close the stream
	ch.CloseStream(taskID)

	// Second response should be dropped — the channel should not receive it.
	// Wait briefly to let the mock DLL send it.
	time.Sleep(200 * time.Millisecond)

	select {
	case _, ok := <-respCh:
		if ok {
			t.Error("received unexpected response after CloseStream")
		}
	default:
		// Expected: nothing in channel
	}
}

func TestChannelCloseIdempotent(t *testing.T) {
	ch := &Channel{
		pending:  make(map[uint32]chan *implantpb.Spite),
		recvDone: make(chan struct{}),
		parser:   malefic.NewMaleficParser(),
	}

	if err := ch.Close(); err != nil {
		t.Fatalf("close without conn: %v", err)
	}
	if err := ch.Close(); err != nil {
		t.Fatalf("double close: %v", err)
	}
}

func TestChannelForwardAfterClose(t *testing.T) {
	ch := &Channel{
		closed:   true,
		pending:  make(map[uint32]chan *implantpb.Spite),
		recvDone: make(chan struct{}),
		parser:   malefic.NewMaleficParser(),
	}

	_, err := ch.Forward(1, &implantpb.Spite{Name: "exec"})
	if err == nil {
		t.Fatal("expected error forwarding on closed channel")
	}
}
