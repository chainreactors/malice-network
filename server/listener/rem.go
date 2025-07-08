package listener

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/helper/third/rem"
	"github.com/chainreactors/rem/agent"
)

func NewRem(rpc listenerrpc.ListenerRPCClient, pipeline *clientpb.Pipeline) (*REM, error) {
	remConfig := pipeline.GetRem()
	var conURL string
	if remConfig.Link != "" {
		conURL = remConfig.Link
	} else {
		conURL = remConfig.Console
	}

	console, err := rem.NewRemServer(conURL, pipeline.Ip)
	if err != nil {
		return nil, err
	}
	pp := &REM{
		con:        console,
		rpc:        rpc,
		remConfig:  remConfig,
		Name:       pipeline.Name,
		ListenerID: pipeline.ListenerId,
	}
	return pp, nil
}

type REM struct {
	con        *rem.RemConsole
	rpc        listenerrpc.ListenerRPCClient
	remConfig  *clientpb.REM
	ListenerID string
	Name       string
	Enable     bool
}

func (rem *REM) ID() string {
	return rem.Name
}

func (rem *REM) Start() error {
	if rem.Enable {
		return nil
	}

	err := rem.con.Listen(rem.con.ConsoleURL)
	if err != nil {
		return err
	}
	rem.Enable = true
	logs.Log.Important(rem.con.Link())
	go func() {
		for rem.Enable {
			agent, err := rem.con.Accept()
			if err != nil {
				logs.Log.Error(err)
				continue
			}

			go rem.con.Handler(agent)
		}
	}()

	go func() {
		for rem.Enable {
			_, err := rem.rpc.HealthCheckRem(context.Background(), rem.ToProtobuf())
			if err != nil {
				logs.Log.Error(err)
			}

			time.Sleep(30 * time.Second)
		}
	}()
	return nil
}

func (rem *REM) ToProtobuf() *clientpb.Pipeline {
	return &clientpb.Pipeline{
		Name:       rem.Name,
		Enable:     rem.Enable,
		ListenerId: rem.ListenerID,
		Type:       consts.RemPipeline,
		Body: &clientpb.Pipeline_Rem{
			Rem: &clientpb.REM{
				Name:       rem.Name,
				ListenerId: rem.ListenerID,
				Host:       rem.con.ConsoleURL.Hostname(),
				Console:    rem.remConfig.Console,
				Port:       uint32(rem.con.ConsoleURL.IntPort()),
				Link:       rem.con.Link(),
				Subscribe:  rem.con.Subscribe(),
				Agents:     rem.con.ToProtobuf(),
			},
		},
	}
}

func (rem *REM) Close() error {
	return rem.con.Close()
}

func (lns *listener) handlerRemAgentCtrl(job *clientpb.Job) error {
	rem := lns.pipelines.Get(job.Name)
	if rem == nil {
		return errors.New("rem not found")
	}

	body := job.GetRemAgent()
	if body == nil {
		return errors.New("agent not found")
	}
	a, err := rem.(*REM).con.Fork(body.Id, body.Args)
	if err != nil {
		return err
	}
	job.Body = &clientpb.Job_RemAgent{
		RemAgent: &clientpb.REMAgent{
			Id:     a.Name(),
			Mod:    a.Mod,
			Local:  a.LocalURL.String(),
			Remote: a.RemoteURL.String(),
		},
	}
	return nil
}

func (lns *listener) handlerRemAgentLog(job *clientpb.Job) error {
	rem := lns.pipelines.Get(job.Name)
	if rem == nil {
		return errors.New("rem not found")
	}

	body := job.GetRemAgent()
	if body == nil {
		return errors.New("agent not found")
	}
	a, ok := agent.Agents.Get(body.Id)
	if ok {
		job.Body = &clientpb.Job_RemLog{
			RemLog: &clientpb.RemLog{
				PipelineId: job.Name,
				AgentId:    body.Id,
				Log:        a.HistoryLog(),
			},
		}
		return nil
	} else {
		return errors.New("agent not found")
	}
}

func (lns *listener) handlerRemAgentStop(job *clientpb.Job) error {
	rem := lns.pipelines.Get(job.Name)
	if rem == nil {
		return errors.New("rem not found")
	}

	body := job.GetRemAgent()
	if body == nil {
		return errors.New("agent not found")
	}
	a, ok := agent.Agents.Get(body.Id)
	if ok {
		a.Close(fmt.Errorf("stop by manual"))
		return nil
	} else {
		return errors.New("agent not found")
	}
}
