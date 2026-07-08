//go:build race

package parser

import (
	"sync"
	"testing"

	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
)

func TestSpitesCacheConcurrentAppendAndBuildRace(t *testing.T) {
	sc := NewSpitesBuf()
	start := make(chan struct{})

	var wg sync.WaitGroup
	for worker := 0; worker < 4; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			<-start
			for i := 0; i < 2000; i++ {
				sc.Append(&implantpb.Spite{
					TaskId: uint32(worker*2000 + i),
					Body:   &implantpb.Spite_Empty{},
				})
			}
		}(worker)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 8000; i++ {
			if sc.Len() > 0 {
				_ = sc.Build()
			}
		}
	}()

	close(start)
	wg.Wait()
}
