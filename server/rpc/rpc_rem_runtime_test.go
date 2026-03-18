package rpc

import (
	"context"
	"strings"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/malice-network/server/internal/core"
)

func TestRemAgentCtrlPropagatesListenerFailure(t *testing.T) {
	newRPCTestEnv(t)
	listener, pipeline := seedRemRuntime(t, "rem-runtime-failure")

	go func() {
		ctrl := <-listener.Ctrl
		listener.CtrlJob.Store(ctrl.Id, &clientpb.JobStatus{
			CtrlId: ctrl.Id,
			Status: consts.CtrlStatusFailed,
			Error:  "listener ctrl failed",
		})
	}()

	_, err := (&Server{}).RemAgentCtrl(context.Background(), &clientpb.REMAgent{
		PipelineId: pipeline.Name,
		Id:         "agent-1",
	})
	if err == nil || !strings.Contains(err.Error(), "listener ctrl failed") {
		t.Fatalf("RemAgentCtrl error = %v, want listener failure", err)
	}
}

func TestRemAgentLogRejectsMissingLogPayload(t *testing.T) {
	newRPCTestEnv(t)
	listener, pipeline := seedRemRuntime(t, "rem-runtime-log")

	go func() {
		ctrl := <-listener.Ctrl
		listener.CtrlJob.Store(ctrl.Id, &clientpb.JobStatus{
			CtrlId: ctrl.Id,
			Status: consts.CtrlStatusSuccess,
		})
	}()

	_, err := (&Server{}).RemAgentLog(context.Background(), &clientpb.REMAgent{
		PipelineId: pipeline.Name,
		Id:         "agent-2",
	})
	if err == nil || !strings.Contains(err.Error(), "missing log") {
		t.Fatalf("RemAgentLog error = %v, want missing log error", err)
	}
}

func TestRemAgentHandlersRejectNilRequest(t *testing.T) {
	if _, err := (&Server{}).RemAgentCtrl(context.Background(), nil); err == nil || !strings.Contains(err.Error(), types.ErrMissingRequestField.Error()) {
		t.Fatalf("RemAgentCtrl(nil) error = %v, want %v", err, types.ErrMissingRequestField)
	}
	if _, err := (&Server{}).RemAgentLog(context.Background(), nil); err == nil || !strings.Contains(err.Error(), types.ErrMissingRequestField.Error()) {
		t.Fatalf("RemAgentLog(nil) error = %v, want %v", err, types.ErrMissingRequestField)
	}
}

func seedRemRuntime(t testing.TB, name string) (*core.Listener, *clientpb.Pipeline) {
	t.Helper()

	listener := core.NewListener("listener-"+name, "127.0.0.1")
	core.Listeners.Add(listener)
	pipeline := &clientpb.Pipeline{
		Name:       name,
		ListenerId: listener.Name,
		Type:       consts.RemPipeline,
		Body: &clientpb.Pipeline_Rem{
			Rem: &clientpb.REM{
				Agents: map[string]*clientpb.REMAgent{},
			},
		},
	}
	listener.AddPipeline(pipeline)
	return listener, pipeline
}
