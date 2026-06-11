package core

import (
	"fmt"
	"net"
	"net/http"
	"testing"
)

func TestCheckMCPHealthRejectsNonSSEHTTPServers(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to allocate test listener: %v", err)
	}
	defer ln.Close()

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
		}),
	}
	defer server.Close()
	go server.Serve(ln)

	host, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatalf("failed to parse listener address: %v", err)
	}

	startPort := 0
	_, err = fmt.Sscanf(port, "%d", &startPort)
	if err != nil {
		t.Fatalf("failed to parse port: %v", err)
	}

	if checkMCPHealth(host, startPort) {
		t.Fatalf("expected plain HTTP service on port %d to be rejected as MCP", startPort)
	}
}

func TestCheckMCPHealthAcceptsSSEServer(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to allocate test listener: %v", err)
	}
	defer ln.Close()

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
		}),
	}
	defer server.Close()
	go server.Serve(ln)

	host, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatalf("failed to parse listener address: %v", err)
	}

	startPort := 0
	_, err = fmt.Sscanf(port, "%d", &startPort)
	if err != nil {
		t.Fatalf("failed to parse port: %v", err)
	}

	if !checkMCPHealth(host, startPort) {
		t.Fatalf("expected SSE service on port %d to be recognized as MCP", startPort)
	}
}

func TestFindAvailableMCPPortSkipsNonMCPHTTPServers(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to allocate test listener: %v", err)
	}
	defer ln.Close()

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		}),
	}
	defer server.Close()
	go server.Serve(ln)

	host, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatalf("failed to parse listener address: %v", err)
	}

	startPort := 0
	_, err = fmt.Sscanf(port, "%d", &startPort)
	if err != nil {
		t.Fatalf("failed to parse port: %v", err)
	}

	gotPort, err := findAvailableMCPPort(host, startPort)
	if err != nil {
		t.Fatalf("findAvailableMCPPort returned error: %v", err)
	}
	if gotPort == startPort {
		t.Fatalf("expected occupied non-MCP port %d to be skipped", startPort)
	}
}
