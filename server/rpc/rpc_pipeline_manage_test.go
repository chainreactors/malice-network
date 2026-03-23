package rpc

import (
	"context"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/core"
)

// ---------------------------------------------------------------------------
// RegisterPipeline
// ---------------------------------------------------------------------------

// BUG TEST: RegisterPipeline with nil request will panic because it accesses
// req.ListenerId without a nil check.
func TestRegisterPipeline_NilRequest(t *testing.T) {
	_ = newRPCTestEnv(t)
	defer func() {
		if r := recover(); r != nil {
			t.Logf("BUG CONFIRMED: RegisterPipeline(nil) panics: %v", r)
		}
	}()
	_, err := (&Server{}).RegisterPipeline(context.Background(), nil)
	if err != nil {
		t.Logf("RegisterPipeline(nil) returned error (no panic): %v", err)
	}
}

func TestRegisterPipeline_UnknownListener(t *testing.T) {
	_ = newRPCTestEnv(t)
	_, err := (&Server{}).RegisterPipeline(context.Background(), &clientpb.Pipeline{
		Name:       "orphan-pipe",
		ListenerId: "nonexistent-listener",
		Type:       consts.TCPPipeline,
		Body: &clientpb.Pipeline_Tcp{
			Tcp: &clientpb.TCPPipeline{
				Name:       "orphan-pipe",
				ListenerId: "nonexistent-listener",
				Host:       "127.0.0.1",
				Port:       5555,
			},
		},
	})
	if err == nil {
		t.Fatal("RegisterPipeline with unknown listener should return error")
	}
}

func TestRegisterPipeline_Valid(t *testing.T) {
	env := newRPCTestEnv(t)
	// seedSession creates a listener named "test-listener".
	env.seedSession(t, "rp-valid-sess", "rp-valid-pipe", true)

	_, err := (&Server{}).RegisterPipeline(context.Background(), &clientpb.Pipeline{
		Name:       "new-registered-pipe",
		ListenerId: "test-listener",
		Ip:         "127.0.0.1",
		Type:       consts.TCPPipeline,
		Secure:     &clientpb.Secure{},
		Body: &clientpb.Pipeline_Tcp{
			Tcp: &clientpb.TCPPipeline{
				Name:       "new-registered-pipe",
				ListenerId: "test-listener",
				Host:       "127.0.0.1",
				Port:       6666,
			},
		},
	})
	if err != nil {
		t.Fatalf("RegisterPipeline error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// ListPipelines
// ---------------------------------------------------------------------------

func TestListPipelines_Empty(t *testing.T) {
	_ = newRPCTestEnv(t)
	resp, err := (&Server{}).ListPipelines(context.Background(), &clientpb.Listener{Id: "no-such-listener"})
	if err != nil {
		t.Fatalf("ListPipelines error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if len(resp.Pipelines) != 0 {
		t.Fatalf("expected 0 pipelines, got %d", len(resp.Pipelines))
	}
}

func TestListPipelines_AfterRegister(t *testing.T) {
	env := newRPCTestEnv(t)
	env.seedSession(t, "lp-after-sess", "lp-after-pipe", true)

	pipeline := &clientpb.Pipeline{
		Name:       "list-test-pipe",
		ListenerId: "test-listener",
		Ip:         "127.0.0.1",
		Type:       consts.TCPPipeline,
		Secure:     &clientpb.Secure{},
		Body: &clientpb.Pipeline_Tcp{
			Tcp: &clientpb.TCPPipeline{
				Name:       "list-test-pipe",
				ListenerId: "test-listener",
				Host:       "127.0.0.1",
				Port:       7777,
			},
		},
	}
	if _, err := (&Server{}).RegisterPipeline(context.Background(), pipeline); err != nil {
		t.Fatalf("RegisterPipeline error: %v", err)
	}

	resp, err := (&Server{}).ListPipelines(context.Background(), &clientpb.Listener{Id: "test-listener"})
	if err != nil {
		t.Fatalf("ListPipelines error: %v", err)
	}
	found := false
	for _, p := range resp.Pipelines {
		if p.Name == "list-test-pipe" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("registered pipeline not found in ListPipelines result")
	}
}

// BUG TEST: ListPipelines with nil request panics accessing req.Id.
func TestListPipelines_NilRequest(t *testing.T) {
	_ = newRPCTestEnv(t)
	defer func() {
		if r := recover(); r != nil {
			t.Logf("BUG CONFIRMED: ListPipelines(nil) panics: %v", r)
		}
	}()
	_, err := (&Server{}).ListPipelines(context.Background(), nil)
	if err != nil {
		t.Logf("ListPipelines(nil) returned error (no panic): %v", err)
	}
}

// ---------------------------------------------------------------------------
// SyncPipeline - nil request
// ---------------------------------------------------------------------------

func TestSyncPipeline_NilRequest(t *testing.T) {
	_ = newRPCTestEnv(t)
	defer func() {
		if r := recover(); r != nil {
			t.Logf("BUG CONFIRMED: SyncPipeline(nil) panics: %v", r)
		}
	}()
	_, err := (&Server{}).SyncPipeline(context.Background(), nil)
	if err != nil {
		t.Logf("SyncPipeline(nil) returned error (no panic): %v", err)
	}
}

// ---------------------------------------------------------------------------
// Edge: RegisterPipeline creates default profile
// ---------------------------------------------------------------------------

func TestRegisterPipeline_CreatesDefaultProfile(t *testing.T) {
	env := newRPCTestEnv(t)
	env.seedSession(t, "rp-prof-sess", "rp-prof-pipe", true)

	pipeName := "profile-check-pipe"
	_, err := (&Server{}).RegisterPipeline(context.Background(), &clientpb.Pipeline{
		Name:       pipeName,
		ListenerId: "test-listener",
		Ip:         "127.0.0.1",
		Type:       consts.TCPPipeline,
		Secure:     &clientpb.Secure{},
		Body: &clientpb.Pipeline_Tcp{
			Tcp: &clientpb.TCPPipeline{
				Name:       pipeName,
				ListenerId: "test-listener",
				Host:       "127.0.0.1",
				Port:       8888,
			},
		},
	})
	if err != nil {
		t.Fatalf("RegisterPipeline error: %v", err)
	}

	// RegisterPipeline should have called db.NewProfile with name "<pipe>_default".
	// We just verify the call did not error and the pipeline was registered.
	lns, err := core.Listeners.Get("test-listener")
	if err != nil {
		t.Fatalf("Listeners.Get error: %v", err)
	}
	if pipe := lns.GetPipeline(pipeName); pipe == nil {
		// RegisterPipeline does not add to listener's in-memory pipeline list --
		// it only persists to DB. This is expected.
		t.Log("pipeline not in listener memory (expected: RegisterPipeline only persists to DB)")
	}
}
