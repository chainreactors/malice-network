package pipeline

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/core"
)

type Pipelines struct {
	TCPPipelines []*TCPPipeline `config:"tcp"`
}

func (lns Pipelines) Start() {
	for _, l := range lns.TCPPipelines {
		job, err := l.Start()
		if err != nil {
			logs.Log.Error(err.Error())
			continue
		}
		core.Jobs.Add(job)
	}
}
