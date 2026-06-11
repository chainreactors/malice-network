package core

import (
	"sync"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/types"
)

// Bug #1: jobID++ is non-atomic. Concurrent NextJobID calls race.
// Run with: go test -race ./server/internal/core/... -run TestJobIDRaceUnderConcurrentAllocation
// Bug #1 fix verification: atomic counters must produce unique IDs under concurrency.
func TestJobIDRaceUnderConcurrentAllocation(t *testing.T) {
	withIsolatedJobs(t)

	const goroutines = 20
	ids := make(chan uint32, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			ids <- NextJobID()
		}()
	}
	wg.Wait()
	close(ids)

	seen := map[uint32]bool{}
	for id := range ids {
		if seen[id] {
			t.Fatalf("duplicate job ID %d — atomic counter fix is broken", id)
		}
		seen[id] = true
	}
}

// Bug #1: ctrlID++ is non-atomic.
// Bug #1 fix verification: atomic ctrl IDs under concurrency.
func TestCtrlIDRaceUnderConcurrentAllocation(t *testing.T) {
	withIsolatedJobs(t)

	const goroutines = 20
	ids := make(chan uint32, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			ids <- NextCtrlID()
		}()
	}
	wg.Wait()
	close(ids)

	seen := map[uint32]bool{}
	for id := range ids {
		if seen[id] {
			t.Fatalf("duplicate ctrl ID %d — atomic counter fix is broken", id)
		}
		seen[id] = true
	}
}

func TestCurrentJobIDReflectsLastAllocation(t *testing.T) {
	withIsolatedJobs(t)

	id1 := NextJobID()
	if CurrentJobID() != id1 {
		t.Fatalf("CurrentJobID = %d, want %d", CurrentJobID(), id1)
	}
	id2 := NextJobID()
	if CurrentJobID() != id2 {
		t.Fatalf("CurrentJobID = %d, want %d", CurrentJobID(), id2)
	}
}

// Bug #5 fix: AddPipeline with invalid input returns unstored Job and logs warning.
func TestJobsAddPipelineNilReturnsUnstored(t *testing.T) {
	withIsolatedJobs(t)
	withIsolatedListeners(t)
	withIsolatedBroker(t)

	job := Jobs.AddPipeline(nil)
	if job == nil {
		t.Fatal("AddPipeline(nil) should return a sentinel Job (not nil) for caller safety")
	}
	if len(Jobs.All()) != 0 {
		t.Fatal("AddPipeline(nil) should NOT store the job in the map")
	}
}

// Bug #5 fix: Empty ListenerId/Name returns unstored Job and logs warning.
func TestJobsAddPipelineEmptyFieldsNotStored(t *testing.T) {
	withIsolatedJobs(t)
	withIsolatedListeners(t)
	withIsolatedBroker(t)

	job := Jobs.AddPipeline(&clientpb.Pipeline{Name: "pipe", ListenerId: ""})
	if job == nil {
		t.Fatal("expected non-nil sentinel Job")
	}
	if len(Jobs.All()) != 0 {
		t.Fatal("Job with empty ListenerId should NOT be stored in the map")
	}

	job2 := Jobs.AddPipeline(&clientpb.Pipeline{Name: "", ListenerId: "listener"})
	if job2 == nil {
		t.Fatal("expected non-nil sentinel Job")
	}
	if len(Jobs.All()) != 0 {
		t.Fatal("Job with empty Name should NOT be stored in the map")
	}
}

func TestJobsAddPipelineNewAndUpsert(t *testing.T) {
	withIsolatedJobs(t)
	withIsolatedListeners(t)
	withIsolatedBroker(t)

	listener := NewListener("test-listener", "10.0.0.1")
	Listeners.Add(listener)

	pipe := &clientpb.Pipeline{
		Name:       "tcp-pipe",
		ListenerId: "test-listener",
		Type:       consts.TCPPipeline,
	}
	job := Jobs.AddPipeline(pipe)
	if job == nil || job.ID == 0 {
		t.Fatal("AddPipeline should return a stored job with valid ID")
	}
	if len(Jobs.All()) != 1 {
		t.Fatalf("job count = %d, want 1", len(Jobs.All()))
	}

	// Upsert: same name+listener → same Job pointer, updated pipeline
	updatedPipe := &clientpb.Pipeline{
		Name:       "tcp-pipe",
		ListenerId: "test-listener",
		Type:       consts.HTTPPipeline,
	}
	sameJob := Jobs.AddPipeline(updatedPipe)
	if sameJob != job {
		t.Fatal("upsert should return the same Job pointer")
	}
	if len(Jobs.All()) != 1 {
		t.Fatalf("job count after upsert = %d, want 1", len(Jobs.All()))
	}
	if sameJob.Pipeline.Type != consts.HTTPPipeline {
		t.Fatal("upsert should update the pipeline")
	}
}

func TestJobsRemoveAndGetConsistency(t *testing.T) {
	withIsolatedJobs(t)
	withIsolatedListeners(t)
	withIsolatedBroker(t)

	listener := NewListener("rm-listener", "10.0.0.1")
	Listeners.Add(listener)

	Jobs.AddPipeline(&clientpb.Pipeline{
		Name:       "rm-pipe",
		ListenerId: "rm-listener",
	})
	if len(Jobs.All()) != 1 {
		t.Fatal("expected 1 job")
	}

	Jobs.Remove("rm-listener", "rm-pipe")
	if len(Jobs.All()) != 0 {
		t.Fatal("expected 0 jobs after Remove")
	}

	_, err := Jobs.Get("rm-pipe")
	if err != types.ErrNotFoundPipeline {
		t.Fatalf("Get after Remove should return ErrNotFoundPipeline, got %v", err)
	}
}

func TestJobsRemoveEmptyArgsNoPanic(t *testing.T) {
	withIsolatedJobs(t)

	// Should not panic.
	Jobs.Remove("", "")
	Jobs.Remove("a", "")
	Jobs.Remove("", "b")
}

func TestJobsGetMultipleMatchesError(t *testing.T) {
	withIsolatedJobs(t)
	withIsolatedListeners(t)
	withIsolatedBroker(t)

	l1 := NewListener("listener-1", "10.0.0.1")
	l2 := NewListener("listener-2", "10.0.0.2")
	Listeners.Add(l1)
	Listeners.Add(l2)

	Jobs.AddPipeline(&clientpb.Pipeline{Name: "shared-name", ListenerId: "listener-1"})
	Jobs.AddPipeline(&clientpb.Pipeline{Name: "shared-name", ListenerId: "listener-2"})

	_, err := Jobs.Get("shared-name")
	if err == nil {
		t.Fatal("Get should return error when multiple jobs share the same name")
	}

	// GetByListener should resolve the ambiguity.
	job, err := Jobs.GetByListener("shared-name", "listener-1")
	if err != nil || job == nil {
		t.Fatalf("GetByListener should resolve ambiguity, got err=%v", err)
	}
}

func TestJobsGetEmptyName(t *testing.T) {
	withIsolatedJobs(t)

	_, err := Jobs.Get("")
	if err != types.ErrNotFoundPipeline {
		t.Fatalf("Get('') = %v, want ErrNotFoundPipeline", err)
	}
}

func TestJobsGetByListenerEmptyListenerID(t *testing.T) {
	withIsolatedJobs(t)

	_, err := Jobs.GetByListener("pipe", "")
	if err == nil {
		t.Fatal("GetByListener with empty listenerID should return error")
	}
}

func TestJobToProtobuf(t *testing.T) {
	job := &Job{
		ID:   42,
		Name: "test-pipe",
		Pipeline: &clientpb.Pipeline{
			Name: "test-pipe",
			Type: consts.TCPPipeline,
		},
	}
	pb := job.ToProtobuf()
	if pb.Id != 42 || pb.Name != "test-pipe" || pb.Pipeline == nil {
		t.Fatalf("ToProtobuf = %#v", pb)
	}
}
