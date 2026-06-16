package core

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	mcpserver "github.com/mark3labs/mcp-go/server"
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
			if r.URL.Path != "/mcp/sse" {
				http.NotFound(w, r)
				return
			}
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

func TestCheckMCPHealthAcceptsStreamableHTTPServer(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to allocate test listener: %v", err)
	}
	addr := ln.Addr().String()
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		ln.Close()
		t.Fatalf("failed to parse listener address: %v", err)
	}
	ln.Close()

	startPort := 0
	_, err = fmt.Sscanf(port, "%d", &startPort)
	if err != nil {
		t.Fatalf("failed to parse port: %v", err)
	}

	mcpServer := mcpserver.NewMCPServer("test-server", "1.0.0")
	streamServer := mcpserver.NewStreamableHTTPServer(
		mcpServer,
		mcpserver.WithEndpointPath("/mcp"),
	)
	go func() {
		if err := streamServer.Start(addr); err != nil && err != http.ErrServerClosed {
			t.Errorf("streamable HTTP server failed: %v", err)
		}
	}()
	defer streamServer.Shutdown(context.Background())

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if checkMCPHealth(host, startPort) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("expected Streamable HTTP MCP service on port %d to be recognized as MCP", startPort)
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
