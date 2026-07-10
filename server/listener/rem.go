package listener

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/services/listenerrpc"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/third/rem"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/rem/agent"
	"github.com/chainreactors/rem/protocol/message"
)

var remHealthCheck = func(client listenerrpc.ListenerRPCClient, ctx context.Context, pipeline *clientpb.Pipeline) error {
	if client == nil {
		return errors.New("rem rpc client is nil")
	}
	_, err := client.HealthCheckRem(ctx, pipeline)
	return err
}

var remConsoleListen = func(con *rem.RemConsole) error {
	return con.Listen(con.ConsoleURL)
}

var remConsoleClose = func(con *rem.RemConsole) error {
	return con.Close()
}

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
		con:            console,
		rpc:            rpc,
		remConfig:      remConfig,
		Name:           pipeline.Name,
		ListenerID:     pipeline.ListenerId,
		CertName:       pipeline.CertName,
		PipelineConfig: core.FromPipeline(pipeline),
	}
	return pp, nil
}

type REM struct {
	stateMu        sync.RWMutex
	starting       bool
	startDone      chan struct{}
	stopCh         chan struct{}
	healthInterval time.Duration
	con            *rem.RemConsole
	rpc            listenerrpc.ListenerRPCClient
	remConfig      *clientpb.REM
	ListenerID     string
	Name           string
	Enable         bool
	CertName       string
	*core.PipelineConfig
	ownAgents sync.Map // agent.ID → struct{}: tracks agents belonging to this pipeline
}

func (rem *REM) ID() string {
	return rem.Name
}

func (rem *REM) Start() error {
	if !rem.beginStart() {
		return nil
	}

	err := remConsoleListen(rem.con)
	if err != nil {
		rem.abortStart()
		return err
	}
	if !rem.enabled() {
		_ = remConsoleClose(rem.con)
		rem.abortStart()
		return nil
	}
	logs.Log.Important(rem.con.Link())
	if !rem.commitStart() {
		_ = remConsoleClose(rem.con)
		rem.abortStart()
		return nil
	}
	return nil
}

func (rem *REM) ToProtobuf() *clientpb.Pipeline {
	enabled := rem.enabled()
	link := rem.getLink()
	subscribe := rem.getSubscribe()
	host := ""
	var port uint32
	agents := map[string]*clientpb.REMAgent{}
	if rem.con != nil && rem.con.ConsoleURL != nil {
		host = rem.con.ConsoleURL.Hostname()
		port = uint32(rem.con.ConsoleURL.IntPort())
		allAgents := rem.con.ToProtobuf()
		rem.ownAgents.Range(func(key, value interface{}) bool {
			id := key.(string)
			if a, ok := allAgents[id]; ok {
				agents[id] = a
			}
			return true
		})
	}

	var tlsConfig *clientpb.TLS
	var encryption []*clientpb.Encryption
	var secure *clientpb.Secure
	parserName := ""
	if rem.PipelineConfig != nil {
		parserName = rem.Parser
		if rem.TLSConfig != nil {
			tlsConfig = rem.TLSConfig.ToProtobuf()
		}
		encryption = rem.Encryption.ToProtobuf()
		if rem.SecureConfig != nil {
			secure = rem.SecureConfig.ToProtobuf()
		}
	}

	return &clientpb.Pipeline{
		Name:       rem.Name,
		Enable:     enabled,
		ListenerId: rem.ListenerID,
		Parser:     parserName,
		Type:       consts.RemPipeline,
		CertName:   rem.CertName,
		Body: &clientpb.Pipeline_Rem{
			Rem: &clientpb.REM{
				Name:       rem.Name,
				ListenerId: rem.ListenerID,
				Host:       host,
				Console:    rem.remConfig.Console,
				Port:       port,
				Link:       link,
				Subscribe:  subscribe,
				Agents:     agents,
			},
		},
		Tls:        tlsConfig,
		Encryption: encryption,
		Secure:     secure,
	}
}

func (rem *REM) getLink() (link string) {
	if rem.remConfig != nil && rem.remConfig.Link != "" {
		link = rem.remConfig.Link
	}
	if rem.con == nil {
		return link
	}
	core.RunGuarded("rem-link:"+rem.Name, func() error {
		if runtimeLink := rem.con.Link(); runtimeLink != "" {
			link = runtimeLink
		}
		return nil
	}, func(err error) {
		logs.Log.Debugf("rem runtime link unavailable: %s", core.ErrorText(err))
	})
	return link
}

func (rem *REM) getSubscribe() (subscribe string) {
	if rem.remConfig != nil && rem.remConfig.Subscribe != "" {
		subscribe = rem.remConfig.Subscribe
	}
	if rem.con == nil {
		return subscribe
	}
	core.RunGuarded("rem-subscribe:"+rem.Name, func() error {
		if runtimeSubscribe := rem.con.Subscribe(); runtimeSubscribe != "" {
			subscribe = runtimeSubscribe
		}
		return nil
	}, func(err error) {
		logs.Log.Debugf("rem runtime subscribe unavailable: %s", core.ErrorText(err))
	})
	return subscribe
}

func (rem *REM) Close() error {
	rem.stateMu.Lock()
	wasActive := rem.Enable
	rem.Enable = false
	stopCh := rem.stopCh
	rem.stopCh = nil
	if rem.starting {
		done := rem.startDone
		rem.stateMu.Unlock()
		if stopCh != nil {
			close(stopCh)
		}
		<-done
		return nil
	}
	rem.stateMu.Unlock()
	if stopCh != nil {
		close(stopCh)
	}
	if !wasActive || rem.con == nil {
		return nil
	}
	return remConsoleClose(rem.con)
}

func (rem *REM) enabled() bool {
	rem.stateMu.RLock()
	defer rem.stateMu.RUnlock()
	return rem.Enable
}

func (rem *REM) beginStart() bool {
	for {
		rem.stateMu.Lock()
		if rem.starting {
			done := rem.startDone
			rem.stateMu.Unlock()
			<-done
			continue
		}
		if rem.Enable {
			rem.stateMu.Unlock()
			return false
		}
		rem.Enable = true
		rem.starting = true
		rem.startDone = make(chan struct{})
		rem.stopCh = make(chan struct{})
		rem.stateMu.Unlock()
		return true
	}
}

func (rem *REM) abortStart() {
	rem.stateMu.Lock()
	rem.starting = false
	rem.Enable = false
	done := rem.startDone
	rem.startDone = nil
	stopCh := rem.stopCh
	rem.stopCh = nil
	if done != nil {
		close(done)
	}
	rem.stateMu.Unlock()
	if stopCh != nil {
		close(stopCh)
	}
}

func (rem *REM) commitStart() bool {
	rem.stateMu.Lock()
	defer rem.stateMu.Unlock()
	if !rem.Enable {
		return false
	}
	core.GoGuarded("rem-accept:"+rem.Name, rem.acceptLoop, rem.runtimeErrorHandler("accept loop"))
	core.GoGuarded("rem-health:"+rem.Name, rem.healthLoop, rem.runtimeErrorHandler("health loop"))
	rem.starting = false
	done := rem.startDone
	rem.startDone = nil
	close(done)
	return true
}

func (rem *REM) acceptLoop() error {
	for rem.enabled() {
		ag, err := rem.con.Accept()
		if err != nil {
			if !rem.enabled() {
				return nil
			}
			// Accept errors are typically transient (timeout, client disconnect).
			// Log and continue rather than killing the entire pipeline — the next
			// client reconnect should succeed once the simplex channel is healthy.
			logs.Log.Errorf("rem %s accept error (will retry): %v", rem.Name, err)
			continue
		}

		rem.ownAgents.Store(ag.ID, struct{}{})

		// Trigger an immediate health check so the new agent's PivotingContext
		// is created in DB right away instead of waiting for the periodic loop.
		if err := remHealthCheck(rem.rpc, context.Background(), rem.ToProtobuf()); err != nil {
			logs.Log.Warnf("rem %s post-accept health check failed: %v", rem.Name, err)
		}

		core.GoGuarded("rem-agent:"+rem.Name, func() error {
			rem.con.Handler(ag)
			rem.ownAgents.Delete(ag.ID)
			return nil
		}, core.LogGuardedError("rem-agent:"+rem.Name))
	}
	return nil
}

func (rem *REM) healthLoop() error {
	const (
		healthFailureThreshold = 3
		opHealthDegraded       = "health-check-failed"
		opHealthRecovered      = "health-check-recovered"
	)

	consecutiveFailures := 0
	unhealthy := false
	stopCh := rem.stopSignal()
	for rem.enabled() {
		if err := remHealthCheck(rem.rpc, context.Background(), rem.ToProtobuf()); err != nil {
			consecutiveFailures++
			logs.Log.Errorf("rem %s health check failed (%d/%d): %v", rem.Name, consecutiveFailures, healthFailureThreshold, err)
			if consecutiveFailures >= healthFailureThreshold && !unhealthy {
				unhealthy = true
				if core.EventBroker != nil {
					core.EventBroker.Publish(core.Event{
						EventType: consts.EventListener,
						Op:        opHealthDegraded,
						Listener:  &clientpb.Listener{Id: rem.ListenerID},
						Message:   fmt.Sprintf("rem pipeline %s health degraded", rem.Name),
						Err:       err.Error(),
						Important: true,
					})
				}
			}
		} else {
			if unhealthy && core.EventBroker != nil {
				core.EventBroker.Publish(core.Event{
					EventType: consts.EventListener,
					Op:        opHealthRecovered,
					Listener:  &clientpb.Listener{Id: rem.ListenerID},
					Message:   fmt.Sprintf("rem pipeline %s health recovered", rem.Name),
					Important: true,
				})
			}
			consecutiveFailures = 0
			unhealthy = false
		}
		interval := rem.healthInterval
		if interval <= 0 {
			interval = 30 * time.Second
		}
		timer := time.NewTimer(interval)
		select {
		case <-timer.C:
		case <-stopCh:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return nil
		}
	}
	return nil
}

func (rem *REM) stopSignal() <-chan struct{} {
	rem.stateMu.Lock()
	defer rem.stateMu.Unlock()
	if rem.stopCh == nil {
		rem.stopCh = make(chan struct{})
	}
	return rem.stopCh
}

func (rem *REM) runtimeErrorHandler(scope string) core.GoErrorHandler {
	label := fmt.Sprintf("rem pipeline %s %s", rem.Name, scope)
	return core.CombineErrorHandlers(
		core.LogGuardedError(label),
		func(err error) {
			_ = rem.Close()
			if core.EventBroker != nil {
				core.EventBroker.Publish(core.Event{
					EventType: consts.EventListener,
					Op:        consts.CtrlRemStop,
					Listener:  &clientpb.Listener{Id: rem.ListenerID},
					Message:   label,
					Err:       core.ErrorText(err),
					Important: true,
				})
			}
		},
	)
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
	rem.(*REM).ownAgents.Store(a.ID, struct{}{})
	job.Body = &clientpb.Job_RemAgent{
		RemAgent: &clientpb.REMAgent{
			Id:          a.Name(),
			InboundSide: a.InboundSide,
			Local:       a.LocalURL.String(),
			Remote:      a.RemoteURL.String(),
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

func (lns *listener) handlerRemAgentReconfigure(job *clientpb.Job) error {
	body := job.GetRemAgent()
	if body == nil {
		return errors.New("agent not found")
	}
	if len(body.Args) < 2 {
		return errors.New("missing interval argument (args: reconfigure <interval_ms>)")
	}
	a, ok := agent.Agents.Get(body.Id)
	if !ok {
		return fmt.Errorf("agent %s not found in Agents registry", body.Id)
	}
	interval, err := strconv.ParseInt(body.Args[1], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid interval: %w", err)
	}
	logs.Log.Importantf("[rem.reconfigure] sending Reconfigure{interval: %d} to agent %s",
		interval, a.Name())
	err = a.Send(&message.Reconfigure{Options: map[string]string{"interval": strconv.FormatInt(interval, 10)}})
	if err != nil {
		logs.Log.Errorf("[rem.reconfigure] send failed for agent %s: %v", a.Name(), err)
	} else {
		logs.Log.Importantf("[rem.reconfigure] send succeeded for agent %s", a.Name())
	}
	return err
}
