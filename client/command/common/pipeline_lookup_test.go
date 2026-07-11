package common

import (
	"sync"
	"testing"

	iomclient "github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/core"
)

func TestFindCachedPipelineConcurrentWithEventReconciliation(t *testing.T) {
	state := &iomclient.ServerState{Pipelines: make(map[string]*clientpb.Pipeline)}
	con := &core.Console{Server: &core.Server{ServerState: state}}

	const iterations = 2000
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < iterations; i++ {
			pipeline := &clientpb.Pipeline{
				Name:       "shared-pipeline",
				ListenerId: "listener-main",
				Type:       consts.RemPipeline,
				Body: &clientpb.Pipeline_Rem{Rem: &clientpb.REM{
					Name:       "shared-pipeline",
					ListenerId: "listener-main",
					Link:       "tcp://127.0.0.1:19966",
				}},
			}
			state.ReconcileEvent(&clientpb.Event{
				Type: consts.EventJob,
				Op:   consts.CtrlPipelineStart,
				Job:  &clientpb.Job{Pipeline: pipeline},
			})
			state.ReconcileEvent(&clientpb.Event{
				Type: consts.EventJob,
				Op:   consts.CtrlPipelineStop,
				Job:  &clientpb.Job{Pipeline: pipeline},
			})
		}
	}()

	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < iterations; i++ {
			_, _ = FindCachedPipeline(con, "shared-pipeline", nil)
		}
	}()

	close(start)
	wg.Wait()
}
