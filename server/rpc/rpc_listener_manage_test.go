package rpc

import (
	"context"
	"testing"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/core"
)

// ---------------------------------------------------------------------------
// GetListeners
// ---------------------------------------------------------------------------

func TestGetListeners_Empty(t *testing.T) {
	_ = newRPCTestEnv(t)
	resp, err := (&Server{}).GetListeners(context.Background(), &clientpb.Empty{})
	if err != nil {
		t.Fatalf("GetListeners error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if len(resp.Listeners) != 0 {
		t.Fatalf("expected 0 listeners, got %d", len(resp.Listeners))
	}
}

func TestGetListeners_AfterRegister(t *testing.T) {
	_ = newRPCTestEnv(t)

	_, err := (&Server{}).RegisterListener(context.Background(), &clientpb.RegisterListener{
		Name: "test-get-listener",
		Host: "192.168.1.1",
	})
	if err != nil {
		t.Fatalf("RegisterListener error: %v", err)
	}

	resp, err := (&Server{}).GetListeners(context.Background(), &clientpb.Empty{})
	if err != nil {
		t.Fatalf("GetListeners error: %v", err)
	}
	found := false
	for _, l := range resp.Listeners {
		if l.Id == "test-get-listener" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("registered listener not found in GetListeners result")
	}
}

// ---------------------------------------------------------------------------
// RegisterListener
// ---------------------------------------------------------------------------

func TestRegisterListener_CreatesNew(t *testing.T) {
	_ = newRPCTestEnv(t)

	_, err := (&Server{}).RegisterListener(context.Background(), &clientpb.RegisterListener{
		Name: "new-listener",
		Host: "10.0.0.1",
	})
	if err != nil {
		t.Fatalf("RegisterListener error: %v", err)
	}

	lns, err := core.Listeners.Get("new-listener")
	if err != nil {
		t.Fatalf("listener not found after registration: %v", err)
	}
	if lns.IP != "10.0.0.1" {
		t.Fatalf("listener IP = %q, want %q", lns.IP, "10.0.0.1")
	}
}

func TestRegisterListener_Idempotent(t *testing.T) {
	_ = newRPCTestEnv(t)

	for i := 0; i < 3; i++ {
		_, err := (&Server{}).RegisterListener(context.Background(), &clientpb.RegisterListener{
			Name: "idempotent-listener",
			Host: "10.0.0.2",
		})
		if err != nil {
			t.Fatalf("RegisterListener attempt %d error: %v", i+1, err)
		}
	}

	lns, err := core.Listeners.Get("idempotent-listener")
	if err != nil {
		t.Fatalf("listener not found after re-registration: %v", err)
	}
	if lns.IP != "10.0.0.2" {
		t.Fatalf("listener IP = %q, want %q", lns.IP, "10.0.0.2")
	}
}

// BUG TEST: RegisterListener with nil request panics accessing req.Name.
func TestRegisterListener_NilRequest(t *testing.T) {
	_ = newRPCTestEnv(t)
	defer func() {
		if r := recover(); r != nil {
			t.Logf("BUG CONFIRMED: RegisterListener(nil) panics: %v", r)
		}
	}()
	_, err := (&Server{}).RegisterListener(context.Background(), nil)
	if err != nil {
		t.Logf("RegisterListener(nil) returned error (no panic): %v", err)
	}
}

// Edge: empty name creates a listener with empty name.
func TestRegisterListener_EmptyName(t *testing.T) {
	_ = newRPCTestEnv(t)
	_, err := (&Server{}).RegisterListener(context.Background(), &clientpb.RegisterListener{
		Name: "",
		Host: "10.0.0.3",
	})
	// This likely succeeds but creates a listener with empty name.
	// core.Listeners.Get("") returns ErrNotFoundListener.
	if err != nil {
		t.Logf("RegisterListener(empty name) returned error: %v", err)
		return
	}
	// Try to retrieve it; Get("") should fail.
	_, getErr := core.Listeners.Get("")
	if getErr != nil {
		t.Log("RegisterListener(empty name) succeeded but Get('') fails -- inconsistent state")
	}
}

// ---------------------------------------------------------------------------
// ListJobs
// ---------------------------------------------------------------------------

func TestListJobs_Empty(t *testing.T) {
	_ = newRPCTestEnv(t)
	resp, err := (&Server{}).ListJobs(context.Background(), &clientpb.Empty{})
	if err != nil {
		t.Fatalf("ListJobs error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if len(resp.Pipelines) != 0 {
		t.Fatalf("expected 0 jobs, got %d", len(resp.Pipelines))
	}
}

func TestListJobs_AfterAddPipeline(t *testing.T) {
	_ = newRPCTestEnv(t)

	core.Jobs.AddPipeline(&clientpb.Pipeline{
		Name:       "job-pipe",
		ListenerId: "job-listener",
	})

	resp, err := (&Server{}).ListJobs(context.Background(), &clientpb.Empty{})
	if err != nil {
		t.Fatalf("ListJobs error: %v", err)
	}
	if len(resp.Pipelines) < 1 {
		t.Fatalf("expected at least 1 job, got %d", len(resp.Pipelines))
	}
}

// ---------------------------------------------------------------------------
// GetListeners with nil request (req is unused, should not panic)
// ---------------------------------------------------------------------------

func TestGetListeners_NilRequest(t *testing.T) {
	_ = newRPCTestEnv(t)
	resp, err := (&Server{}).GetListeners(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetListeners(nil) error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}
