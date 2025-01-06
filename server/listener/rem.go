package listener

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/helper/proxy"
	rem "github.com/chainreactors/rem/runner"
	"github.com/chainreactors/rem/utils"
)

func NewRem(rpc listenerrpc.ListenerRPCClient, pipeline *clientpb.Pipeline) (*REM, error) {
	remConfig := pipeline.GetRem()

	remCon, err := proxy.NewRemServer(remConfig.Console)
	if err != nil {
		return nil, err
	}
	pp := &REM{
		remCon:     remCon,
		rpc:        rpc,
		Name:       pipeline.Name,
		Enable:     true,
		ListenerID: pipeline.ListenerId,
	}

	return pp, nil
}

type REM struct {
	remCon     *rem.Console
	rpc        listenerrpc.ListenerRPCClient
	ListenerID string
	Name       string
	ConsoleURL *utils.URL
	Enable     bool
}

func (rem *REM) ID() string {
	return rem.Name
}

func (rem *REM) Start() error {
	if !rem.Enable {
		return nil
	}

	err := rem.remCon.Listen(rem.remCon.ConsoleURL.Hostname())
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
				Console: rem.remCon.ConsoleURL.String(),
			},
		},
	}
}

func (rem *REM) Close() error {
	return rem.remCon.Close()
}
