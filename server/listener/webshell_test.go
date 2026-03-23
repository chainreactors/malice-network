package listener

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/server/internal/parser"
)

func TestMaleficParserRoundtrip(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	p, err := parser.NewParser(consts.ImplantMalefic)
	if err != nil {
		t.Fatalf("NewParser: %v", err)
	}

	want := &implantpb.Spites{
		Spites: []*implantpb.Spite{
			{Name: "test_cmd", TaskId: 42},
		},
	}
	var sid uint32 = 1234

	errCh := make(chan error, 1)
	go func() {
		errCh <- p.WritePacket(server, want, sid)
	}()

	gotSid, got, err := p.ReadPacket(client)
	if err != nil {
		t.Fatalf("ReadPacket: %v", err)
	}
	if writeErr := <-errCh; writeErr != nil {
		t.Fatalf("WritePacket: %v", writeErr)
	}

	if gotSid != sid {
		t.Fatalf("sid = %d, want %d", gotSid, sid)
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

func TestMaleficParserInvalidStart(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	p, _ := parser.NewParser(consts.ImplantMalefic)

	go func() {
		buf := make([]byte, 9)
		buf[0] = 0xFF // invalid start delimiter
		server.Write(buf)
	}()

	_, _, err := p.ReadPacket(client)
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

func TestNewWebShellPipelineMissingParams(t *testing.T) {
	_, err := NewWebShellPipeline(nil, nil)
	if err == nil {
		t.Fatal("expected error for nil pipeline")
	}
}

func TestNewWebShellPipelineValidParams(t *testing.T) {
	pipeline := &clientpb.Pipeline{
		Name:       "ws1",
		ListenerId: "listener-a",
		Enable:     true,
		Type:       "webshell",
		Body: &clientpb.Pipeline_Custom{
			Custom: &clientpb.CustomPipeline{
				Name:   "ws1",
				Params: `{"suo5_url":"suo5://target/bridge.php","dll_path":"/tmp/bridge.dll"}`,
			},
		},
	}

	p, err := NewWebShellPipeline(nil, pipeline)
	if err != nil {
		t.Fatalf("NewWebShellPipeline: %v", err)
	}
	if p.Suo5URL != "suo5://target/bridge.php" {
		t.Fatalf("Suo5URL = %q, want %q", p.Suo5URL, "suo5://target/bridge.php")
	}
	if p.parser == nil {
		t.Fatal("parser should not be nil")
	}
}

func TestBootstrapHTTPQueryString(t *testing.T) {
	var gotStage, gotCT string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotStage = r.URL.Query().Get("s")
		gotCT = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("LOADED"))
	}))
	defer ts.Close()

	p := &WebShellPipeline{
		Suo5URL:    ts.URL, // use test server URL directly
		httpClient: ts.Client(),
	}
	// Override suo5ToHTTPURL by using an http:// URL directly
	body, err := p.bootstrapHTTP(wsStageStatus, nil)
	if err != nil {
		t.Fatalf("bootstrapHTTP: %v", err)
	}

	if gotStage != "status" {
		t.Errorf("stage query = %q, want %q", gotStage, "status")
	}
	if gotCT != "application/octet-stream" {
		t.Errorf("Content-Type = %q, want %q", gotCT, "application/octet-stream")
	}
	if string(body) != "LOADED" {
		t.Errorf("body = %q, want %q", string(body), "LOADED")
	}
}
