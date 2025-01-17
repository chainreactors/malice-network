package listener

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/helper/rem"
)

func NewRem(rpc listenerrpc.ListenerRPCClient, pipeline *clientpb.Pipeline) (*REM, error) {
	remConfig := pipeline.GetRem()

	console, err := rem.NewRemServer(remConfig.Console, pipeline.Ip)
	if err != nil {
		return nil, err
	}
	pp := &REM{
		remCon:     console,
		rpc:        rpc,
		URL:        &rem.URL{URL: console.ConsoleURL},
		Name:       pipeline.Name,
		Enable:     true,
		ListenerID: pipeline.ListenerId,
	}

	return pp, nil
}

type REM struct {
	remCon     *rem.RemConsole
	rpc        listenerrpc.ListenerRPCClient
	URL        *rem.URL
	ListenerID string
	Name       string
	Enable     bool
}

func (rem *REM) ID() string {
	return rem.Name
}

func (rem *REM) Start() error {
	if !rem.Enable {
		return nil
	}

	err := rem.remCon.Listen(rem.remCon.ConsoleURL)
	if err != nil {
		return err
	}

	logs.Log.Important(rem.remCon.Link())
	go func() {
		for {
			agent, err := rem.remCon.Accept()
			if err != nil {
				logs.Log.Error(err)
				continue
			}

			go rem.remCon.Handler(agent)
		}
	}()
	return nil
}

func (rem *REM) ToProtobuf() *clientpb.Pipeline {
	return &clientpb.Pipeline{
		Name:       rem.Name,
		Enable:     rem.Enable,
		ListenerId: rem.ListenerID,
		Body: &clientpb.Pipeline_Rem{
			Rem: &clientpb.REM{
				Host:      rem.remCon.ConsoleURL.Hostname(),
				Console:   rem.URL.String(),
				Port:      rem.URL.Port(),
				Link:      rem.remCon.Link(),
				Subscribe: rem.remCon.Subscribe(),
				Agents:    rem.remCon.ToProtobuf(),
			},
		},
	}
}

func (rem *REM) Close() error {
	return rem.remCon.Close()
}
