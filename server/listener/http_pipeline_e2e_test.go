package listener

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/implanttypes"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/parser/malefic"
	"google.golang.org/grpc"
)

func TestHTTPPipelineHandlerDoesNotReuseConnectionAcrossPipelines(t *testing.T) {
	oldConnections := core.Connections
	oldForwarders := core.Forwarders
	oldListenerSessions := core.ListenerSessions
	core.ResetTransientTransportState()
	t.Cleanup(func() {
		core.ResetTransientTransportState()
		core.Connections = oldConnections
		core.Forwarders = oldForwarders
		core.ListenerSessions = oldListenerSessions
	})

	rawID := uint32(0x01020304)
	firstPipeline := newHTTPProbePipeline("listener-a", "http-a")
	serveMaleficHTTPProbe(t, firstPipeline, rawID)

	firstConnections := core.Connections.All()
	if len(firstConnections) != 1 {
		t.Fatalf("connections after first HTTP request = %d, want 1", len(firstConnections))
	}
	firstConn := firstConnections[0]
	if firstConn.PipelineID != "listener-a:http-a" {
		t.Fatalf("first connection pipeline = %q, want %q", firstConn.PipelineID, "listener-a:http-a")
	}

	secondPipeline := newHTTPProbePipeline("listener-b", "http-b")
	serveMaleficHTTPProbe(t, secondPipeline, rawID)

	secondConnections := core.Connections.All()
	if len(secondConnections) != 1 {
		t.Fatalf("connections after second HTTP request = %d, want 1", len(secondConnections))
	}
	secondConn := secondConnections[0]
	if secondConn == firstConn {
		t.Fatalf("HTTP pipeline reused raw session %d across pipelines: got %q, want %q",
			rawID, secondConn.PipelineID, "listener-b:http-b")
	}
	if secondConn.PipelineID != "listener-b:http-b" {
		t.Fatalf("second connection pipeline = %q, want %q", secondConn.PipelineID, "listener-b:http-b")
	}
}

func TestHTTPPipelineHandlerDeliversLargeScreenshotSizedResponse(t *testing.T) {
	oldConnections := core.Connections
	oldForwarders := core.Forwarders
	oldListenerSessions := core.ListenerSessions
	core.ResetTransientTransportState()
	t.Cleanup(func() {
		core.ResetTransientTransportState()
		core.Connections = oldConnections
		core.Forwarders = oldForwarders
		core.ListenerSessions = oldListenerSessions
	})

	pipeline := newHTTPProbePipeline("listener-a", "http-a")
	stream := newCaptureForwardStream()
	forward, err := core.NewForward(&captureForwardClient{stream: stream}, pipeline)
	if err != nil {
		t.Fatalf("NewForward failed: %v", err)
	}
	core.Forwarders.Add(forward)

	rawID := uint32(0x01020304)
	const payloadSize = 512 * 1024
	payload := deterministicPayload(payloadSize)
	serveMaleficHTTPPacket(t, pipeline, maleficBinaryResponsePacket(t, rawID, payload))

	select {
	case resp := <-stream.responses:
		if resp.GetSpite().GetBinaryResponse() == nil {
			t.Fatalf("response body = %T, want BinaryResponse", resp.GetSpite().Body)
		}
		if got := len(resp.GetSpite().GetBinaryResponse().Data); got != payloadSize {
			t.Fatalf("binary response size = %d, want %d", got, payloadSize)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("large HTTP response was not delivered to forward stream")
	}
}

func TestHTTPReadWriterReadDeadlineInterruptsSlowBody(t *testing.T) {
	readResult := make(chan error, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &httpReadWriter{body: r.Body, writer: w}
		if err := rw.SetReadDeadline(time.Now().Add(100 * time.Millisecond)); err != nil {
			readResult <- err
			return
		}
		_, err := rw.Read(make([]byte, 1))
		readResult <- err
	}))
	defer server.Close()

	conn, err := net.Dial("tcp", server.Listener.Addr().String())
	if err != nil {
		t.Fatalf("dial HTTP server: %v", err)
	}
	defer conn.Close()
	if _, err := io.WriteString(conn, "POST / HTTP/1.1\r\nHost: example.test\r\nContent-Length: 1\r\nConnection: close\r\n\r\n"); err != nil {
		t.Fatalf("write partial HTTP request: %v", err)
	}

	select {
	case err := <-readResult:
		if err == nil {
			t.Fatal("slow request body read returned without a deadline error")
		}
		var netErr net.Error
		if !errors.As(err, &netErr) || !netErr.Timeout() {
			t.Fatalf("slow request body read error = %v, want timeout", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("slow request body read was not interrupted by its deadline")
	}
}

func newHTTPProbePipeline(listenerID, name string) *HTTPPipeline {
	return &HTTPPipeline{
		Name: name,
		PipelineConfig: &core.PipelineConfig{
			ListenerID: listenerID,
			Parser:     consts.ImplantMalefic,
			Encryption: implanttypes.EncryptionsConfig{
				&implanttypes.EncryptionConfig{Type: consts.CryptorRAW},
			},
		},
	}
}

func serveMaleficHTTPProbe(t testing.TB, pipeline *HTTPPipeline, rawID uint32) {
	t.Helper()

	serveMaleficHTTPPacket(t, pipeline, maleficEmptyPacket(rawID))
}

func serveMaleficHTTPPacket(t testing.TB, pipeline *HTTPPipeline, packet []byte) {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "http://example.test/", bytes.NewReader(packet))
	resp := &deadlineResponseRecorder{ResponseRecorder: httptest.NewRecorder()}

	pipeline.handler(resp, req)
	if resp.Code >= 400 {
		t.Fatalf("HTTP handler status = %d, body = %q", resp.Code, resp.Body.String())
	}
}

type deadlineResponseRecorder struct {
	*httptest.ResponseRecorder
}

func (r *deadlineResponseRecorder) SetReadDeadline(time.Time) error {
	return nil
}

func maleficBinaryResponsePacket(t testing.TB, rawID uint32, payload []byte) []byte {
	t.Helper()

	parser := malefic.NewMaleficParser()
	packet, err := parser.Marshal(&implantpb.Spites{Spites: []*implantpb.Spite{
		{
			Name:   consts.ModuleExecute,
			TaskId: 1,
			Async:  true,
			Body: &implantpb.Spite_BinaryResponse{
				BinaryResponse: &implantpb.BinaryResponse{Data: payload},
			},
		},
	}}, rawID)
	if err != nil {
		t.Fatalf("marshal malefic response: %v", err)
	}
	return packet
}

func deterministicPayload(size int) []byte {
	payload := make([]byte, 0, size)
	var seed [32]byte
	for len(payload) < size {
		seed = sha256.Sum256(seed[:])
		payload = append(payload, seed[:]...)
	}
	return payload[:size]
}

type captureForwardStream struct {
	responses chan *clientpb.SpiteResponse
}

func newCaptureForwardStream() *captureForwardStream {
	return &captureForwardStream{responses: make(chan *clientpb.SpiteResponse, 4)}
}

func (s *captureForwardStream) Send(resp *clientpb.SpiteResponse) error {
	s.responses <- resp
	return nil
}

func (s *captureForwardStream) Recv() (*clientpb.SpiteRequest, error) {
	return nil, io.EOF
}

type captureForwardClient struct {
	stream *captureForwardStream
}

func (c *captureForwardClient) OpenForwardStream(context.Context, core.Pipeline) (core.ForwardStream, error) {
	return c.stream, nil
}

func (c *captureForwardClient) Checkin(context.Context, *implantpb.Ping, ...grpc.CallOption) (*clientpb.Empty, error) {
	return &clientpb.Empty{}, nil
}

func (c *captureForwardClient) Register(context.Context, *clientpb.RegisterSession, ...grpc.CallOption) (*clientpb.Empty, error) {
	return &clientpb.Empty{}, nil
}

func maleficEmptyPacket(rawID uint32) []byte {
	packet := make([]byte, malefic.HeaderLength+1)
	packet[0] = malefic.DefaultStartDelimiter
	binary.LittleEndian.PutUint32(packet[1:5], rawID)
	binary.LittleEndian.PutUint32(packet[5:9], 0)
	packet[malefic.HeaderLength] = malefic.DefaultEndDelimiter
	return packet
}
