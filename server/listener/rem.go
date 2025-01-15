package listener

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/services/listenerrpc"
	rem "github.com/chainreactors/rem/core"
	remrunner "github.com/chainreactors/rem/runner"
)

func NewRem(rpc listenerrpc.ListenerRPCClient, pipeline *clientpb.Pipeline) (*REM, error) {
	remConfig := pipeline.GetRem()

	u, err := rem.NewConsoleURL(remConfig.Console)
	if err != nil {
		return nil, err
	}
	var option remrunner.Options
	err = option.ParseArgs([]string{"-c", remConfig.Console})
	if err != nil {
		return nil, err
	}
	remRunner, err := option.Prepare()
	if err != nil {
		return nil, err
	}
	remRunner.URLs.ConsoleURL = u
	console, err := remrunner.NewConsole(remRunner, remRunner.URLs)
	if err != nil {
		return nil, err
	}
	pp := &REM{
		remCon:     console,
		rpc:        rpc,
		Name:       pipeline.Name,
		Enable:     true,
		ListenerID: pipeline.ListenerId,
	}

	return pp, nil
}

type REM struct {
	remCon     *remrunner.Console
	rpc        listenerrpc.ListenerRPCClient
	ListenerID string
	Name       string
	ConsoleURL *rem.URL
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
				Console: rem.remCon.ConsoleURL.String(),
			},
		},
	}
}

func (rem *REM) Close() error {
	return rem.remCon.Close()
}
