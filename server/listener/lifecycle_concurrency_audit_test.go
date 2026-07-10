package listener

import (
	"sync"
	"testing"

	"github.com/chainreactors/malice-network/server/internal/core"
)

func runAuditConcurrentClose(t *testing.T, closeFn func() error) {
	t.Helper()

	const (
		workers    = 8
		iterations = 1000
	)

	start := make(chan struct{})
	errs := make(chan error, workers*iterations)
	var wg sync.WaitGroup
	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			<-start
			for range iterations {
				if err := closeFn(); err != nil {
					errs <- err
				}
			}
		}()
	}

	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent Close returned an error: %v", err)
	}
}

func TestAuditTCPPipelineConcurrentClose(t *testing.T) {
	pipeline := &TCPPipeline{Enable: true}
	runAuditConcurrentClose(t, pipeline.Close)
}

func TestAuditHTTPPipelineConcurrentClose(t *testing.T) {
	pipeline := &HTTPPipeline{Enable: true}
	runAuditConcurrentClose(t, pipeline.Close)
}

func TestAuditREMConcurrentClose(t *testing.T) {
	pipeline := &REM{Enable: true}
	runAuditConcurrentClose(t, pipeline.Close)
}

func TestAuditListenerConcurrentClose(t *testing.T) {
	lns := &listener{
		Name:      "audit-listener-close",
		pipelines: core.NewPipelines(),
		websites:  make(map[string]*Website),
	}

	oldListener := Listener
	Listener = lns
	t.Cleanup(func() {
		Listener = oldListener
	})

	runAuditConcurrentClose(t, lns.Close)
}
