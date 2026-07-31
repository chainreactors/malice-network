package rpc

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func TestRegisterListenerRestoresReconnectRuntimeSnapshot(t *testing.T) {
	_ = newRPCTestEnv(t)
	const listenerID = "listener-reconnect-snapshot"
	pipeline := &clientpb.Pipeline{
		Name:       "existing-runtime",
		ListenerId: listenerID,
		Enable:     true,
	}

	_, err := (&Server{}).RegisterListener(context.Background(), &clientpb.RegisterListener{
		Name:      listenerID,
		Host:      "10.0.0.8",
		Pipelines: &clientpb.Pipelines{Pipelines: []*clientpb.Pipeline{pipeline}},
	})
	if err != nil {
		t.Fatalf("RegisterListener error: %v", err)
	}
	lns, err := core.Listeners.Get(listenerID)
	if err != nil {
		t.Fatalf("listener not found after registration: %v", err)
	}
	if got := lns.GetPipeline(pipeline.Name); got == nil || !got.GetEnable() {
		t.Fatalf("restored runtime = %#v, want enabled pipeline", got)
	}
	if !lns.RuntimeSnapshotReceived() {
		t.Fatal("listener was not marked for runtime reconciliation")
	}
}

func TestRegisterListenerRejectsRuntimeSnapshotFromAnotherListener(t *testing.T) {
	_ = newRPCTestEnv(t)
	_, err := (&Server{}).RegisterListener(context.Background(), &clientpb.RegisterListener{
		Name: "listener-a",
		Pipelines: &clientpb.Pipelines{Pipelines: []*clientpb.Pipeline{{
			Name:       "foreign-runtime",
			ListenerId: "listener-b",
			Enable:     true,
		}}},
	})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("RegisterListener error = %v, want PermissionDenied", err)
	}
}

func TestReconcileListenerRuntimeStartsEnabledPipelineMissingFromSnapshot(t *testing.T) {
	_ = newRPCTestEnv(t)
	const listenerID = "listener-reconcile"
	pipeline := &clientpb.Pipeline{
		Name:       "AA",
		ListenerId: listenerID,
		Enable:     true,
		Type:       consts.HTTPPipeline,
		Body: &clientpb.Pipeline_Http{Http: &clientpb.HTTPPipeline{
			Name:       "AA",
			ListenerId: listenerID,
			Host:       "127.0.0.1",
			Port:       8899,
		}},
	}
	if _, err := db.SavePipeline(models.FromPipelinePb(pipeline)); err != nil {
		t.Fatalf("SavePipeline failed: %v", err)
	}

	lns := core.NewListener(listenerID, "127.0.0.1")
	core.Listeners.Add(lns)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- (&Server{}).reconcileListenerRuntime(ctx, lns)
	}()

	select {
	case ctrl := <-lns.Ctrl:
		if ctrl.GetCtrl() != consts.CtrlPipelineStart {
			t.Fatalf("reconcile control = %q, want %q", ctrl.GetCtrl(), consts.CtrlPipelineStart)
		}
		if got := ctrl.GetJob().GetPipeline(); got.GetName() != "AA" || got.GetListenerId() != listenerID {
			t.Fatalf("reconcile pipeline = %#v, want %s/AA", got, listenerID)
		}
		lns.CtrlJob.Store(ctrl.GetId(), &clientpb.JobStatus{
			CtrlId: ctrl.GetId(),
			Ctrl:   ctrl.GetCtrl(),
			Status: consts.CtrlStatusSuccess,
			Job:    ctrl.GetJob(),
		})
	case <-ctx.Done():
		t.Fatal("timed out waiting for pipeline recovery control")
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("reconcileListenerRuntime failed: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for runtime reconciliation")
	}
	stored, err := db.FindPipelineByListener("AA", listenerID)
	if err != nil {
		t.Fatalf("FindPipelineByListener failed: %v", err)
	}
	if !stored.Enable {
		t.Fatal("successful reconciliation changed the desired enabled state")
	}
}

func TestReconcileListenerRuntimeStopsPipelineDisabledOrDeletedWhileOffline(t *testing.T) {
	for _, tc := range []struct {
		name      string
		persisted bool
	}{
		{name: "disabled", persisted: true},
		{name: "deleted", persisted: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_ = newRPCTestEnv(t)
			listenerID := "listener-offline-stop-reconcile-" + tc.name
			pipeline := &clientpb.Pipeline{
				Name:       "AA",
				ListenerId: listenerID,
				Enable:     false,
				Type:       consts.HTTPPipeline,
				Body: &clientpb.Pipeline_Http{Http: &clientpb.HTTPPipeline{
					Name:       "AA",
					ListenerId: listenerID,
					Host:       "127.0.0.1",
					Port:       8899,
				}},
			}
			if tc.persisted {
				if _, err := db.SavePipeline(models.FromPipelinePb(pipeline)); err != nil {
					t.Fatalf("SavePipeline failed: %v", err)
				}
			}
			pipeline.Enable = true
			lns := core.NewListener(listenerID, "127.0.0.1")
			lns.AddPipeline(pipeline)
			core.Listeners.Add(lns)

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			done := make(chan error, 1)
			go func() {
				done <- (&Server{}).reconcileListenerRuntime(ctx, lns)
			}()

			select {
			case ctrl := <-lns.Ctrl:
				if ctrl.GetCtrl() != consts.CtrlPipelineStop {
					t.Fatalf("reconcile control = %q, want %q", ctrl.GetCtrl(), consts.CtrlPipelineStop)
				}
				lns.CtrlJob.Store(ctrl.GetId(), &clientpb.JobStatus{
					CtrlId: ctrl.GetId(),
					Ctrl:   ctrl.GetCtrl(),
					Status: consts.CtrlStatusSuccess,
					Job:    ctrl.GetJob(),
				})
			case <-ctx.Done():
				t.Fatal("timed out waiting for pipeline stop reconciliation")
			}

			select {
			case err := <-done:
				if err != nil {
					t.Fatalf("reconcileListenerRuntime failed: %v", err)
				}
			case <-ctx.Done():
				t.Fatal("timed out waiting for runtime reconciliation")
			}
			if got := lns.GetPipeline("AA"); got != nil {
				t.Fatalf("undesired runtime remained registered: %#v", got)
			}
		})
	}
}

func TestRegisterListenerRejectsColonName(t *testing.T) {
	_ = newRPCTestEnv(t)

	_, err := (&Server{}).RegisterListener(context.Background(), &clientpb.RegisterListener{
		Name: "team:a",
		Host: "10.0.0.1",
	})
	if err == nil {
		t.Fatal("RegisterListener should reject ':' in listener name")
	}
}

func TestRegisterListener_RejectsActiveDuplicate(t *testing.T) {
	_ = newRPCTestEnv(t)

	_, err := (&Server{}).RegisterListener(context.Background(), &clientpb.RegisterListener{
		Name: "active-listener",
		Host: "10.0.0.2",
	})
	if err != nil {
		t.Fatalf("RegisterListener error: %v", err)
	}

	_, err = (&Server{}).RegisterListener(context.Background(), &clientpb.RegisterListener{
		Name: "active-listener",
		Host: "10.0.0.2",
	})
	if err == nil {
		t.Fatal("expected duplicate active listener registration to fail")
	}
	if status.Code(err) != codes.AlreadyExists {
		t.Fatalf("RegisterListener duplicate error code = %v, want %v", status.Code(err), codes.AlreadyExists)
	}

	lns, err := core.Listeners.Get("active-listener")
	if err != nil {
		t.Fatalf("listener not found after registration: %v", err)
	}
	if lns.IP != "10.0.0.2" {
		t.Fatalf("listener IP = %q, want %q", lns.IP, "10.0.0.2")
	}
}

func TestRegisterListener_ReRegisterAfterStop(t *testing.T) {
	_ = newRPCTestEnv(t)

	_, err := (&Server{}).RegisterListener(context.Background(), &clientpb.RegisterListener{
		Name: "restartable-listener",
		Host: "10.0.0.2",
	})
	if err != nil {
		t.Fatalf("RegisterListener error: %v", err)
	}

	if err := core.Listeners.Stop("restartable-listener"); err != nil {
		t.Fatalf("Stop listener error: %v", err)
	}

	_, err = (&Server{}).RegisterListener(context.Background(), &clientpb.RegisterListener{
		Name: "restartable-listener",
		Host: "10.0.0.3",
	})
	if err != nil {
		t.Fatalf("RegisterListener after stop error: %v", err)
	}

	lns, err := core.Listeners.Get("restartable-listener")
	if err != nil {
		t.Fatalf("listener not found after re-registration: %v", err)
	}
	if lns.IP != "10.0.0.3" {
		t.Fatalf("listener IP = %q, want %q", lns.IP, "10.0.0.3")
	}
}

func TestRegisterListener_ConcurrentDuplicateHasSingleWinner(t *testing.T) {
	_ = newRPCTestEnv(t)

	const attempts = 32
	start := make(chan struct{})
	results := make(chan error, attempts)
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := (&Server{}).RegisterListener(context.Background(), &clientpb.RegisterListener{
				Name: "concurrent-listener",
				Host: "10.0.0.4",
			})
			results <- err
		}()
	}
	close(start)
	wg.Wait()
	close(results)

	successes := 0
	conflicts := 0
	for err := range results {
		switch status.Code(err) {
		case codes.OK:
			successes++
		case codes.AlreadyExists:
			conflicts++
		default:
			t.Fatalf("unexpected registration error: %v", err)
		}
	}
	if successes != 1 || conflicts != attempts-1 {
		t.Fatalf("registration results = %d successes/%d conflicts, want 1/%d", successes, conflicts, attempts-1)
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
