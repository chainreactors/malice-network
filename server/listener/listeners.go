package listener

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/configs"
	"github.com/chainreactors/malice-network/server/core"
)

type Listeners struct {
	TCPListeners []*TCPListener `config:"tcp"`
}

func (lns Listeners) Start() {
	for _, l := range lns.TCPListeners {
		job, err := l.Start()
		if err != nil {
			logs.Log.Error(err.Error())
			continue
		}
		core.Jobs.Add(job)
		core.Forwarders.Add(core.NewForward(configs.GetServerConfig().String(), l))
	}
}
