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

var remConsoleAccept = func(con *rem.RemConsole) (*agent.Agent, error) {
	return con.Accept()
}

var remConsoleHandler = func(con *rem.RemConsole, ag *agent.Agent) {
	con.Handler(ag)
}

const (
	remHealthCheckTimeout = 10 * time.Second
	remAcceptRetryMin     = 10 * time.Millisecond
	remAcceptRetryMax     = time.Second
)

type remStartState struct {
	done       chan struct{}
	cleanupErr error
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
	startState     *remStartState
	runCtx         context.Context
	runCancel      context.CancelFunc
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
		rem.abortStart(nil)
		return err
	}
	if !rem.enabled() {
		cleanupErr := remConsoleClose(rem.con)
		rem.abortStart(cleanupErr)
		return cleanupErr
	}
	logs.Log.Important(rem.con.Link())
	if !rem.commitStart() {
		cleanupErr := remConsoleClose(rem.con)
		rem.abortStart(cleanupErr)
		return cleanupErr
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
	cancel := rem.runCancel
	rem.runCancel = nil
	rem.runCtx = nil
	if rem.starting {
		startState := rem.startState
		rem.stateMu.Unlock()
		if cancel != nil {
			cancel()
		}
		<-startState.done
		return startState.cleanupErr
	}
	rem.stateMu.Unlock()
	if cancel != nil {
		cancel()
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
			done := rem.startState.done
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
		rem.startState = &remStartState{done: make(chan struct{})}
		rem.runCtx, rem.runCancel = context.WithCancel(context.Background())
		rem.stateMu.Unlock()
		return true
	}
}

func (rem *REM) abortStart(cleanupErr error) {
	rem.stateMu.Lock()
	rem.starting = false
	rem.Enable = false
	startState := rem.startState
	rem.startState = nil
	cancel := rem.runCancel
	rem.runCancel = nil
	rem.runCtx = nil
	if startState != nil {
		startState.cleanupErr = cleanupErr
		close(startState.done)
	}
	rem.stateMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (rem *REM) commitStart() bool {
	rem.stateMu.Lock()
	defer rem.stateMu.Unlock()
	if !rem.Enable {
		return false
	}
	runCtx := rem.runCtx
	core.GoGuarded("rem-accept:"+rem.Name, func() error {
		return rem.acceptLoopContext(runCtx)
	}, rem.runtimeErrorHandler("accept loop"))
	core.GoGuarded("rem-health:"+rem.Name, func() error {
		return rem.healthLoopContext(runCtx)
	}, rem.runtimeErrorHandler("health loop"))
	rem.starting = false
	startState := rem.startState
	rem.startState = nil
	close(startState.done)
	return true
}

func (rem *REM) acceptLoopContext(runCtx context.Context) error {
	return rem.acceptLoopContextWithBackoff(runCtx, remAcceptRetryMin, remAcceptRetryMax)
}

func (rem *REM) acceptLoopContextWithBackoff(runCtx context.Context, retryDelay, retryMax time.Duration) error {
	if runCtx == nil {
		runCtx = context.Background()
	}
	retryMinimum := retryDelay
	for rem.enabled() {
		ag, err := remConsoleAccept(rem.con)
		if err != nil {
			if !rem.enabled() || runCtx.Err() != nil {
				return nil
			}
			// Accept errors are typically transient (timeout, client disconnect).
			// Log and continue rather than killing the entire pipeline — the next
			// client reconnect should succeed once the simplex channel is healthy.
			logs.Log.Errorf("rem %s accept error (will retry): %v", rem.Name, err)
			if !waitREMRetry(runCtx, retryDelay) {
				return nil
			}
			retryDelay = nextREMRetryDelay(retryDelay, retryMax)
			continue
		}
		if !rem.enabled() || runCtx.Err() != nil {
			return nil
		}
		retryDelay = retryMinimum

		rem.ownAgents.Store(ag.ID, struct{}{})

		// Trigger an immediate health check so the new agent's PivotingContext
		// is created in DB right away instead of waiting for the periodic loop.
		if err := rem.healthCheck(runCtx); err != nil {
			logs.Log.Warnf("rem %s post-accept health check failed: %v", rem.Name, err)
		}
		if !rem.enabled() || runCtx.Err() != nil {
			rem.ownAgents.Delete(ag.ID)
			return nil
		}

		core.GoGuarded("rem-agent:"+rem.Name, func() error {
			rem.handleAgent(ag)
			return nil
		}, core.LogGuardedError("rem-agent:"+rem.Name))
	}
	return nil
}

func (rem *REM) healthLoopContext(runCtx context.Context) error {
	const (
		healthFailureThreshold = 3
		opHealthDegraded       = "health-check-failed"
		opHealthRecovered      = "health-check-recovered"
	)
	if runCtx == nil {
		runCtx = context.Background()
	}

	consecutiveFailures := 0
	unhealthy := false
	for rem.enabled() {
		if err := rem.healthCheck(runCtx); err != nil {
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
		case <-runCtx.Done():
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

func (rem *REM) healthCheck(runCtx context.Context) error {
	if runCtx == nil {
		runCtx = context.Background()
	}
	ctx, cancel := context.WithTimeout(runCtx, remHealthCheckTimeout)
	defer cancel()
	return remHealthCheck(rem.rpc, ctx, rem.ToProtobuf())
}

func nextREMRetryDelay(current, maximum time.Duration) time.Duration {
	if current >= maximum || current > maximum/2 {
		return maximum
	}
	return current * 2
}

func waitREMRetry(runCtx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-runCtx.Done():
		return false
	}
}

func (rem *REM) handleAgent(ag *agent.Agent) {
	defer rem.ownAgents.Delete(ag.ID)
	remConsoleHandler(rem.con, ag)
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
