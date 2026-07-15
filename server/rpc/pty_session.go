package rpc

import (
	"strings"
	"sync"
	"time"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/utils/pty"
)

type implantPTYSession struct {
	info   pty.Info
	writer *core.SpiteStreamWriter
	greq   *GenericRequest
	output *pty.OutputBuffer
}

type ImplantPTYManager struct {
	mu       sync.Mutex
	sessions map[string]*implantPTYSession
}

func NewImplantPTYManager() *ImplantPTYManager {
	return &ImplantPTYManager{sessions: make(map[string]*implantPTYSession)}
}

func (m *ImplantPTYManager) Register(sessionID string, writer *core.SpiteStreamWriter, greq *GenericRequest) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[sessionID] = &implantPTYSession{
		info: pty.Info{
			ID:        sessionID,
			Kind:      "shell",
			State:     pty.StateRunning,
			StartedAt: time.Now(),
		},
		writer: writer,
		greq:   greq,
		output: pty.NewOutputBuffer(pty.DefaultBufferCap),
	}
}

func (m *ImplantPTYManager) Remove(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.sessions[sessionID]; ok {
		s.info.State = pty.StateCompleted
		s.info.EndedAt = time.Now()
		delete(m.sessions, sessionID)
	}
}

func (m *ImplantPTYManager) Get(sessionID string) (pty.Info, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[sessionID]
	if !ok {
		return pty.Info{}, false
	}
	return s.info, true
}

func (m *ImplantPTYManager) List() []pty.Info {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]pty.Info, 0, len(m.sessions))
	for _, s := range m.sessions {
		out = append(out, s.info)
	}
	return out
}

func (m *ImplantPTYManager) GetTaskProto(sessionID string) (*clientpb.Task, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[sessionID]
	if !ok || s == nil || s.greq == nil || s.greq.Task == nil {
		return nil, false
	}
	return s.greq.Task.ToProtobuf(), true
}

func (m *ImplantPTYManager) SendCommand(sessionID string, req *implantpb.PtyRequest) bool {
	m.mu.Lock()
	s, ok := m.sessions[sessionID]
	if !ok || s == nil || s.writer == nil || s.greq == nil || s.greq.Task == nil {
		m.mu.Unlock()
		return false
	}
	writer := s.writer
	taskID := s.greq.Task.Id
	m.mu.Unlock()
	spite := &implantpb.Spite{
		Name:   types.MsgPty.String(),
		Body:   &implantpb.Spite_PtyRequest{PtyRequest: req},
		TaskId: taskID,
	}
	return writer.Send(spite) == nil
}

func (m *ImplantPTYManager) PumpOutput(sessionID string, greq *GenericRequest, respCh <-chan *implantpb.Spite) {
	for {
		resp, ok := recvSpite(greq.Task.Ctx, respCh)
		if !ok || resp == nil {
			return
		}

		ptyResp := resp.GetPtyResponse()
		_ = greq.HandlerSpite(resp)

		if ptyResp != nil {
			if len(ptyResp.OutputData) > 0 {
				m.bufferOutput(sessionID, ptyResp.OutputData)
			}
			if ptyResp.OutputText != "" {
				m.bufferOutput(sessionID, []byte(ptyResp.OutputText))
			}
			if !ptyResp.SessionActive {
				greq.Task.Finish(resp, "Shell session ended")
				return
			}
		}

		moduleResp := resp.GetResponse()
		if moduleResp != nil && moduleResp.GetError() != "" &&
			strings.Contains(moduleResp.GetError(), "session") &&
			strings.Contains(moduleResp.GetError(), "closed") {
			greq.Task.Finish(resp, "Shell session ended")
			return
		}
	}
}

func (m *ImplantPTYManager) bufferOutput(sessionID string, data []byte) {
	m.mu.Lock()
	s := m.sessions[sessionID]
	m.mu.Unlock()
	if s != nil {
		s.output.Write(data)
	}
}

func (m *ImplantPTYManager) SnapshotOutput(sessionID string, n int) ([]byte, int64) {
	m.mu.Lock()
	s := m.sessions[sessionID]
	m.mu.Unlock()
	if s == nil {
		return nil, 0
	}
	return s.output.TailRawBytesWithOffset(n)
}
