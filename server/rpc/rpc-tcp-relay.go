package rpc

import (
	"context"
	"sync"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/internal/core"
)

// sessionID -> active native tcp_relay stream manager
var tcpRelayManagers sync.Map

type tcpRelaySession struct {
	writer *core.SpiteStreamWriter
	greq   *GenericRequest
	out    <-chan *implantpb.Spite
}

type TcpRelayManager struct {
	mu   sync.Mutex
	sess *tcpRelaySession
}

func getTcpRelayManager(implantID string) *TcpRelayManager {
	if v, ok := tcpRelayManagers.Load(implantID); ok {
		return v.(*TcpRelayManager)
	}
	mgr := &TcpRelayManager{}
	actual, _ := tcpRelayManagers.LoadOrStore(implantID, mgr)
	return actual.(*TcpRelayManager)
}

func (m *TcpRelayManager) Register(writer *core.SpiteStreamWriter, greq *GenericRequest, out <-chan *implantpb.Spite) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Close previous stream if any.
	if m.sess != nil && m.sess.writer != nil {
		m.sess.writer.Close()
	}
	m.sess = &tcpRelaySession{writer: writer, greq: greq, out: out}
}

func (m *TcpRelayManager) Remove() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sess != nil && m.sess.writer != nil {
		m.sess.writer.Close()
	}
	m.sess = nil
}

// alive reports whether the manager has a task and a usable stream writer.
// GetTaskProto/idempotent START must not treat a nil or closed writer as live —
// that creates zombie tasks that Open/Data can never ride.
func (m *TcpRelayManager) aliveLocked() bool {
	if m.sess == nil || m.sess.greq == nil || m.sess.greq.Task == nil || m.sess.writer == nil {
		return false
	}
	return true
}

func (m *TcpRelayManager) GetTaskProto() (*clientpb.Task, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.aliveLocked() {
		return nil, false
	}
	return m.sess.greq.Task.ToProtobuf(), true
}

func (m *TcpRelayManager) Send(spite *implantpb.Spite) bool {
	m.mu.Lock()
	s := m.sess
	m.mu.Unlock()
	if s == nil || s.writer == nil || s.greq == nil || s.greq.Task == nil {
		return false
	}
	spite.TaskId = s.greq.Task.Id
	spite.Name = consts.ModuleTcpRelay
	spite.Async = true
	if err := s.writer.Send(spite); err != nil {
		// Writer is dead/closed — drop registration so the next START recreates.
		m.Remove()
		return false
	}
	return true
}

// TcpRelay starts (or ensures) a long-lived tcp_relay module task on the implant.
// Subsequent TunnelOpen/Data/Close reuse the same task stream (like PTY).
func (rpc *Server) TcpRelay(ctx context.Context, req *implantpb.TunnelCtrl) (*clientpb.Task, error) {
	if req == nil {
		return nil, types.ErrMissingRequestField
	}

	// STOP: try send on existing stream, else one-shot task.
	if req.Action == implantpb.TunnelCtrl_STOP {
		session, err := getSession(ctx)
		if err != nil {
			return nil, err
		}
		mgr := getTcpRelayManager(session.ID)
		if taskPb, ok := mgr.GetTaskProto(); ok {
			spite := &implantpb.Spite{
				Body: &implantpb.Spite_TunnelCtrl{TunnelCtrl: req},
			}
			if mgr.Send(spite) {
				mgr.Remove()
				return taskPb, nil
			}
		}
		greq, err := newGenericRequest(ctx, req)
		if err != nil {
			return nil, err
		}
		ch, err := rpc.GenericHandler(ctx, greq)
		if err != nil {
			return nil, err
		}
		greq.HandlerResponse(ch, types.MsgTunnelList)
		return greq.Task.ToProtobuf(), nil
	}

	// LIST: only on an existing live stream. Never fall through to START.
	if req.Action == implantpb.TunnelCtrl_LIST {
		session, err := getSession(ctx)
		if err != nil {
			return nil, err
		}
		mgr := getTcpRelayManager(session.ID)
		if taskPb, ok := mgr.GetTaskProto(); ok {
			spite := &implantpb.Spite{
				Body: &implantpb.Spite_TunnelCtrl{TunnelCtrl: req},
			}
			if mgr.Send(spite) {
				return taskPb, nil
			}
		}
		return nil, types.ErrNotFoundTask
	}

	// START (or default): open streaming task.
	// Idempotent only when a *sendable* stream is registered (writer non-nil).
	// Zombie entries (task without writer) fall through and recreate.
	if req.Action == implantpb.TunnelCtrl_START || req.Action == implantpb.TunnelCtrl_Action(0) {
		session, err := getSession(ctx)
		if err != nil {
			return nil, err
		}
		mgr := getTcpRelayManager(session.ID)
		if taskPb, ok := mgr.GetTaskProto(); ok {
			return taskPb, nil
		}
		// Drop any half-registered zombie before opening a new stream.
		mgr.Remove()
	}

	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	greq.Count = -1
	in, out, err := rpc.StreamGenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	mgr := getTcpRelayManager(greq.Session.ID)
	mgr.Register(in, greq, out)

	runTaskHandler(greq.Task, func() error {
		for {
			resp, ok := recvSpite(greq.Task.Ctx, out)
			if !ok {
				return ErrTaskContextCancelled
			}
			if resp == nil {
				return nil
			}
			// Persist incremental tunnel events as task progress.
			_ = greq.HandlerSpite(resp)
		}
	}, in.Close, func() {
		greq.Task.Close()
		mgr.Remove()
		logs.Log.Debugf("[tcp_relay] cleaned up session %s task %d", greq.Session.ID, greq.Task.Id)
	})

	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) TunnelOpen(ctx context.Context, req *implantpb.TunnelOpen) (*clientpb.Task, error) {
	return rpc.sendTunnelMsg(ctx, &implantpb.Spite{
		Body: &implantpb.Spite_TunnelOpen{TunnelOpen: req},
	}, req)
}

func (rpc *Server) TunnelData(ctx context.Context, req *implantpb.TunnelData) (*clientpb.Task, error) {
	return rpc.sendTunnelMsg(ctx, &implantpb.Spite{
		Body: &implantpb.Spite_TunnelData{TunnelData: req},
	}, req)
}

func (rpc *Server) TunnelClose(ctx context.Context, req *implantpb.TunnelClose) (*clientpb.Task, error) {
	return rpc.sendTunnelMsg(ctx, &implantpb.Spite{
		Body: &implantpb.Spite_TunnelClose{TunnelClose: req},
	}, req)
}

func (rpc *Server) sendTunnelMsg(ctx context.Context, spite *implantpb.Spite, fallbackMsg interface{}) (*clientpb.Task, error) {
	session, err := getSession(ctx)
	if err != nil {
		return nil, err
	}
	mgr := getTcpRelayManager(session.ID)
	if taskPb, ok := mgr.GetTaskProto(); ok && mgr.Send(spite) {
		return taskPb, nil
	}

	// Fallback: no active stream — reject (client should call TcpRelay START first).
	_ = fallbackMsg
	return nil, types.ErrNotFoundTask
}
