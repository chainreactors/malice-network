package testsupport

import (
	"context"
	"io"
	"os"
	"testing"
	"time"

	iomclient "github.com/chainreactors/IoM-go/client"
	clientcore "github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
)

type ClientHarness struct {
	Server  *clientcore.Server
	Console *clientcore.Console
	Conn    io.Closer
}

func NewClientHarness(t testing.TB, h *ControlPlaneHarness) *ClientHarness {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := h.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
	})

	server, err := clientcore.NewServer(conn, h.Admin)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	go server.EventHandler()
	WaitForCondition(t, 5*time.Second, func() bool {
		return server.EventStatus
	}, "event stream to become active")

	con := &clientcore.Console{
		Server:  server,
		Log:     iomclient.Log,
		CMDs:    make(map[string]*cobra.Command),
		Helpers: make(map[string]*cobra.Command),
	}

	oldWriter := io.Writer(os.Stdout)
	iomclient.Stdout.SetWriter(io.Discard)
	t.Cleanup(func() {
		iomclient.Stdout.SetWriter(oldWriter)
	})

	return &ClientHarness{
		Server:  server,
		Console: con,
		Conn:    conn,
	}
}

func CaptureOutput(fn func()) string {
	start := time.Now()
	fn()
	return iomclient.RemoveANSI(iomclient.Stdout.Range(start, time.Now()))
}
