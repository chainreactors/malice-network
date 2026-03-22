package listener

import (
	"encoding/binary"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"google.golang.org/protobuf/proto"
)

func TestWriteReadFrameTLV(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	want := &implantpb.Spites{
		Spites: []*implantpb.Spite{
			{Name: "test_cmd", TaskId: 42},
		},
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- writeFrame(server, want, 1234)
	}()

	got, err := readFrame(client)
	if err != nil {
		t.Fatalf("readFrame: %v", err)
	}
	if writeErr := <-errCh; writeErr != nil {
		t.Fatalf("writeFrame: %v", writeErr)
	}

	if len(got.Spites) != 1 {
		t.Fatalf("spite count = %d, want 1", len(got.Spites))
	}
	if got.Spites[0].Name != "test_cmd" {
		t.Fatalf("spite name = %q, want %q", got.Spites[0].Name, "test_cmd")
	}
	if got.Spites[0].TaskId != 42 {
		t.Fatalf("task_id = %d, want 42", got.Spites[0].TaskId)
	}
}

func TestWriteFrameTLVWireFormat(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	spites := &implantpb.Spites{
		Spites: []*implantpb.Spite{{Name: "ping"}},
	}
	var sid uint32 = 0xDEAD

	go writeFrame(server, spites, sid)

	// Read raw bytes to verify TLV wire format.
	var hdr [tlvHeaderLen]byte
	if _, err := io.ReadFull(client, hdr[:]); err != nil {
		t.Fatalf("read header: %v", err)
	}

	if hdr[0] != tlvStart {
		t.Fatalf("start delimiter = 0x%02x, want 0x%02x", hdr[0], tlvStart)
	}
	gotSid := binary.LittleEndian.Uint32(hdr[1:5])
	if gotSid != sid {
		t.Fatalf("sid = %d, want %d", gotSid, sid)
	}

	dataLen := binary.LittleEndian.Uint32(hdr[5:9])
	data, _ := proto.Marshal(spites)
	if dataLen != uint32(len(data)) {
		t.Fatalf("frame length = %d, want %d", dataLen, len(data))
	}

	// Read payload + end delimiter.
	payload := make([]byte, dataLen+1)
	if _, err := io.ReadFull(client, payload); err != nil {
		t.Fatalf("read payload: %v", err)
	}
	if payload[dataLen] != tlvEnd {
		t.Fatalf("end delimiter = 0x%02x, want 0x%02x", payload[dataLen], tlvEnd)
	}
}

func TestReadFrameInvalidStart(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	go func() {
		// Write garbage header.
		buf := make([]byte, tlvHeaderLen)
		buf[0] = 0xFF
		server.Write(buf)
	}()

	_, err := readFrame(client)
	if err == nil {
		t.Fatal("expected error for invalid start delimiter")
	}
}

func TestSuo5ToHTTPURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"suo5://target/bridge.php", "http://target/bridge.php"},
		{"suo5s://target/bridge.php", "https://target/bridge.php"},
		{"suo5://10.0.0.1:8080/shell.jsp", "http://10.0.0.1:8080/shell.jsp"},
	}
	for _, tt := range tests {
		got := suo5ToHTTPURL(tt.input)
		if got != tt.want {
			t.Errorf("suo5ToHTTPURL(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestComputeBootstrapToken(t *testing.T) {
	if got := computeBootstrapToken(""); got != "" {
		t.Fatalf("empty secret = %q, want empty", got)
	}
	if got := computeBootstrapToken("short"); got != "short" {
		t.Fatalf("short secret = %q, want %q", got, "short")
	}
	got := computeBootstrapToken("this-is-a-very-long-secret-that-exceeds-32-characters")
	if len(got) != 64 {
		t.Fatalf("HMAC token length = %d, want 64", len(got))
	}
}

func TestNewWebShellPipelineMissingParams(t *testing.T) {
	_, err := NewWebShellPipeline(nil, nil)
	if err == nil {
		t.Fatal("expected error for nil pipeline")
	}
}

func TestHTTPTransportOPSECHeaders(t *testing.T) {
	var gotUA, gotCT string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		gotCT = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer ts.Close()

	transport := &httpTransport{
		client: ts.Client(),
		url:    ts.URL,
		token:  "",
	}

	_, err := transport.do(wsStageStatus, nil, 0)
	if err != nil {
		t.Fatalf("transport.do: %v", err)
	}

	if gotUA != "" {
		t.Errorf("User-Agent = %q, want empty", gotUA)
	}
	if gotCT != "application/x-www-form-urlencoded" {
		t.Errorf("Content-Type = %q, want application/x-www-form-urlencoded", gotCT)
	}
}

func TestHTTPTransportPlaintextEnvelope(t *testing.T) {
	token := "my-secret-token"
	var receivedBody []byte

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer ts.Close()

	transport := newHTTPTransport("suo5://unused", token, 5*time.Second)
	transport.client = ts.Client()
	transport.url = ts.URL

	_, err := transport.do(wsStageStatus, []byte("test"), 0)
	if err != nil {
		t.Fatalf("transport.do: %v", err)
	}

	// Envelope is plaintext: first byte must be the raw stage code.
	if len(receivedBody) == 0 || receivedBody[0] != wsStageStatus {
		t.Errorf("first byte = 0x%02x, want 0x%02x (plaintext stage)", receivedBody[0], wsStageStatus)
	}
}
