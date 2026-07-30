package listener

import (
	"context"
	"sync"
	"time"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/internal/core"
)

const (
	forwardReconnectMinDelay = 100 * time.Millisecond
	forwardReconnectMaxDelay = 5 * time.Second
)

type forwardSupervisor struct {
	ctx      context.Context
	cancel   context.CancelFunc
	rpc      core.ForwardClient
	pipeline core.Pipeline
	kind     string
	dispatch func(*clientpb.SpiteRequest)

	mu      sync.Mutex
	current *core.Forward
	done    chan struct{}
	start   sync.Once
}

func newForwardSupervisor(
	rpc core.ForwardClient,
	pipeline core.Pipeline,
	kind string,
	initial *core.Forward,
	dispatch func(*clientpb.SpiteRequest),
) *forwardSupervisor {
	ctx, cancel := context.WithCancel(context.Background())
	return &forwardSupervisor{
		ctx:      ctx,
		cancel:   cancel,
		rpc:      rpc,
		pipeline: pipeline,
		kind:     kind,
		dispatch: dispatch,
		current:  initial,
		done:     make(chan struct{}),
	}
}

func (s *forwardSupervisor) Start() {
	if s == nil {
		return
	}
	s.start.Do(func() {
		go func() {
			defer close(s.done)
			s.run()
		}()
	})
}

func (s *forwardSupervisor) Stop() error {
	if s == nil {
		return nil
	}
	s.cancel()
	s.mu.Lock()
	current := s.current
	s.current = nil
	s.mu.Unlock()

	var abortErr error
	if current != nil {
		abortErr = core.Forwarders.RemoveIfSame(current.RuntimeKey(), current)
	}
	return abortErr
}

func (s *forwardSupervisor) run() {
	delay := forwardReconnectMinDelay
	for {
		current := s.currentForward()
		if current == nil || s.ctx.Err() != nil {
			return
		}

		msg, err := current.Stream.Recv()
		if err == nil {
			delay = forwardReconnectMinDelay
			if msg != nil && s.dispatch != nil {
				s.dispatch(msg)
			}
			continue
		}
		if s.ctx.Err() != nil {
			return
		}

		logs.Log.Warnf("%s pipeline %s forward disconnected: %v", s.kind, s.pipeline.ID(), err)
		s.detach(current)
		for s.ctx.Err() == nil {
			if !waitForwardReconnect(s.ctx, delay) {
				return
			}
			next, openErr := core.NewForwardContext(s.ctx, s.rpc, s.pipeline)
			if openErr != nil {
				logs.Log.Warnf("%s pipeline %s forward reconnect failed: %v", s.kind, s.pipeline.ID(), openErr)
				delay = nextForwardReconnectDelay(delay)
				continue
			}
			if !s.install(next) {
				_ = next.Abort()
				return
			}
			logs.Log.Infof("%s pipeline %s forward reconnected", s.kind, s.pipeline.ID())
			delay = forwardReconnectMinDelay
			break
		}
	}
}

func (s *forwardSupervisor) currentForward() *core.Forward {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.current
}

func (s *forwardSupervisor) detach(forward *core.Forward) {
	s.mu.Lock()
	if s.current == forward {
		s.current = nil
	}
	s.mu.Unlock()
	_ = core.Forwarders.RemoveIfSame(forward.RuntimeKey(), forward)
}

func (s *forwardSupervisor) install(forward *core.Forward) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ctx.Err() != nil || s.current != nil {
		return false
	}
	if !core.Forwarders.AddIfAbsent(forward) {
		return false
	}
	s.current = forward
	return true
}

func waitForwardReconnect(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func nextForwardReconnectDelay(current time.Duration) time.Duration {
	next := current * 2
	if next > forwardReconnectMaxDelay {
		return forwardReconnectMaxDelay
	}
	return next
}
