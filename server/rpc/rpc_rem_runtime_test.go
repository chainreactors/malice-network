package rpc

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/malice-network/helper/utils/configutil"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
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

func TestRemAgentCtrlUsesScopedPipelineID(t *testing.T) {
	newRPCTestEnv(t)
	pipelineName := "rem-runtime-shared"
	listenerA := core.NewListener("listener-rem-runtime-a", "127.0.0.1")
	listenerB := core.NewListener("listener-rem-runtime-b", "127.0.0.1")
	core.Listeners.Add(listenerA)
	core.Listeners.Add(listenerB)
	for _, listener := range []*core.Listener{listenerA, listenerB} {
		listener.AddPipeline(&clientpb.Pipeline{
			Name:       pipelineName,
			ListenerId: listener.Name,
			Type:       consts.RemPipeline,
			Body: &clientpb.Pipeline_Rem{
				Rem: &clientpb.REM{
					Agents: map[string]*clientpb.REMAgent{},
				},
			},
		})
	}

	got := make(chan *clientpb.JobCtrl, 1)
	go func() {
		got <- <-listenerB.Ctrl
	}()

	_, err := (&Server{}).RemAgentCtrl(context.Background(), &clientpb.REMAgent{
		PipelineId: listenerB.Name + ":" + pipelineName,
		Id:         "agent-scoped",
		Args:       []string{"reconfigure", "500"},
	})
	if err != nil {
		t.Fatalf("RemAgentCtrl failed: %v", err)
	}
	select {
	case ctrl := <-got:
		if ctrl.GetJob().GetPipeline().GetListenerId() != listenerB.Name {
			t.Fatalf("ctrl listener = %q, want %q", ctrl.GetJob().GetPipeline().GetListenerId(), listenerB.Name)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for scoped listener ctrl")
	}
	select {
	case ctrl := <-listenerA.Ctrl:
		t.Fatalf("unexpected ctrl on listener A: %#v", ctrl)
	default:
	}
}

func TestRemAgentCtrlRejectsAmbiguousBarePipelineID(t *testing.T) {
	newRPCTestEnv(t)
	pipelineName := "rem-runtime-ambiguous"
	listenerA, _ := seedRemRuntimeWithListener(t, "listener-rem-ambiguous-a", pipelineName)
	listenerB, _ := seedRemRuntimeWithListener(t, "listener-rem-ambiguous-b", pipelineName)

	_, err := (&Server{}).RemAgentCtrl(context.Background(), &clientpb.REMAgent{
		PipelineId: pipelineName,
		Id:         "agent-ambiguous",
		Args:       []string{"reconfigure", "500"},
	})
	if err == nil {
		t.Fatal("RemAgentCtrl should reject ambiguous bare pipeline ID")
	}
	select {
	case ctrl := <-listenerA.Ctrl:
		t.Fatalf("unexpected ctrl on listener A: %#v", ctrl)
	default:
	}
	select {
	case ctrl := <-listenerB.Ctrl:
		t.Fatalf("unexpected ctrl on listener B: %#v", ctrl)
	default:
	}
}

func TestRemAgentCtrlRejectsDuplicateAgentWithoutPipelineID(t *testing.T) {
	newRPCTestEnv(t)
	agentID := "agent-duplicate"
	listenerA, pipelineA := seedRemRuntimeWithListener(t, "listener-rem-agent-a", "rem-agent-a")
	listenerB, pipelineB := seedRemRuntimeWithListener(t, "listener-rem-agent-b", "rem-agent-b")
	pipelineA.GetRem().Agents[agentID] = &clientpb.REMAgent{Id: agentID}
	pipelineB.GetRem().Agents[agentID] = &clientpb.REMAgent{Id: agentID}

	_, err := (&Server{}).RemAgentCtrl(context.Background(), &clientpb.REMAgent{
		Id:   agentID,
		Args: []string{"reconfigure", "500"},
	})
	if err == nil {
		t.Fatal("RemAgentCtrl should reject duplicate agent ID without pipeline ID")
	}
	select {
	case ctrl := <-listenerA.Ctrl:
		t.Fatalf("unexpected ctrl on listener A: %#v", ctrl)
	default:
	}
	select {
	case ctrl := <-listenerB.Ctrl:
		t.Fatalf("unexpected ctrl on listener B: %#v", ctrl)
	default:
	}
}

func TestRemDialUsesScopedPipelineBeforeAgentExists(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "rem-dial-session", "rem-dial-transport", true)
	pipelineName := "rem-dial-shared"
	listenerA, _ := seedRemRuntimeWithListener(t, "listener-rem-dial-a", pipelineName)
	listenerB, _ := seedRemRuntimeWithListener(t, "listener-rem-dial-b", pipelineName)
	agentID := "agent-created-by-dial"

	pipelinesCh.Store(sess.PipelineID, &testRPCServerStream{
		sendMsg: func(interface{}) error { return nil },
	})
	t.Cleanup(func() { pipelinesCh.Delete(sess.PipelineID) })

	gotSync := make(chan *clientpb.JobCtrl, 1)
	go func() {
		ctrl := <-listenerB.Ctrl
		ctrl.GetJob().GetPipeline().GetRem().Agents[agentID] = &clientpb.REMAgent{
			Id:          agentID,
			PipelineId:  pipelineName,
			InboundSide: "local",
			Local:       "127.0.0.1:9001",
			Remote:      "127.0.0.1:9002",
			Enable:      true,
		}
		listenerB.CtrlJob.Store(ctrl.Id, &clientpb.JobStatus{
			CtrlId: ctrl.Id,
			Status: consts.CtrlStatusSuccess,
			Job:    ctrl.Job,
		})
		gotSync <- ctrl
	}()

	task, err := (&Server{}).RemDial(incomingSessionContext(sess.ID), &implantpb.Request{
		Name: consts.ModuleRemDial,
		Params: map[string]string{
			"pipeline_id": listenerB.Name + ":" + pipelineName,
		},
	})
	if err != nil {
		t.Fatalf("RemDial failed: %v", err)
	}
	deliverTaskResponse(t, sess, task.TaskId, &implantpb.Spite{
		Body: &implantpb.Spite_Response{
			Response: &implantpb.Response{Output: agentID},
		},
	})

	select {
	case ctrl := <-gotSync:
		if ctrl.GetJob().GetPipeline().GetListenerId() != listenerB.Name {
			t.Fatalf("sync listener = %q, want %q", ctrl.GetJob().GetPipeline().GetListenerId(), listenerB.Name)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for scoped REM sync")
	}
	select {
	case ctrl := <-listenerA.Ctrl:
		t.Fatalf("unexpected ctrl on listener A: %#v", ctrl)
	default:
	}

	waitForCondition(t, 2*time.Second, func() bool {
		ctxs, err := db.NewContextQuery().WhereType(consts.ContextPivoting).WherePipeline(pipelineName).Find()
		if err != nil {
			return false
		}
		for _, ctxModel := range ctxs {
			pivot, ok := ctxModel.Context.(*output.PivotingContext)
			if ok && pivot.RemAgentID == agentID && pivot.Listener == listenerB.Name {
				return true
			}
		}
		return false
	}, "scoped REM dial pivot context")
}

func TestListRemsScopesPivotContextsByListener(t *testing.T) {
	newRPCTestEnv(t)
	pipelineName := "rem-list-shared"
	listenerA, _ := seedRemRuntimeWithListener(t, "listener-rem-list-a", pipelineName)
	listenerB, _ := seedRemRuntimeWithListener(t, "listener-rem-list-b", pipelineName)

	for _, tc := range []struct {
		listenerID string
		agentID    string
	}{
		{listenerID: listenerA.Name, agentID: "agent-a"},
		{listenerID: listenerB.Name, agentID: "agent-b"},
	} {
		if err := db.Session().Create(&models.Context{
			PipelineID: pipelineName,
			Type:       consts.ContextPivoting,
			Value: output.MarshalContext(&output.PivotingContext{
				Enable:      true,
				Listener:    tc.listenerID,
				Pipeline:    pipelineName,
				RemAgentID:  tc.agentID,
				InboundSide: "local",
				LocalURL:    "127.0.0.1:8080",
			}),
		}).Error; err != nil {
			t.Fatalf("Create context(%s) failed: %v", tc.agentID, err)
		}
	}

	resp, err := (&Server{}).ListRems(context.Background(), &clientpb.Listener{Id: listenerB.Name})
	if err != nil {
		t.Fatalf("ListRems failed: %v", err)
	}
	if len(resp.Pipelines) != 1 {
		t.Fatalf("pipeline count = %d, want 1", len(resp.Pipelines))
	}
	got := resp.Pipelines[0]
	if got.ListenerId != listenerB.Name {
		t.Fatalf("pipeline listener = %q, want %q", got.ListenerId, listenerB.Name)
	}
	if _, ok := got.GetRem().GetAgents()["agent-b"]; !ok {
		t.Fatalf("listener-b agent missing: %#v", got.GetRem().GetAgents())
	}
	if _, ok := got.GetRem().GetAgents()["agent-a"]; ok {
		t.Fatalf("listener-a agent leaked into listener-b result: %#v", got.GetRem().GetAgents())
	}
}

func TestHealthCheckRemDoesNotDisableOtherListenerOrAmbiguousLegacyContexts(t *testing.T) {
	newRPCTestEnv(t)
	pipelineName := "rem-health-shared"
	listenerA, pipelineA := seedRemRuntimeWithListener(t, "listener-rem-health-a", pipelineName)
	listenerB, _ := seedRemRuntimeWithListener(t, "listener-rem-health-b", pipelineName)

	legacyID := createPivotContext(t, pipelineName, "", "agent-legacy", true)
	otherID := createPivotContext(t, pipelineName, listenerB.Name, "agent-b", true)

	pipelineA.GetRem().Agents = map[string]*clientpb.REMAgent{}
	if _, err := (&Server{}).HealthCheckRem(context.Background(), pipelineA); err != nil {
		t.Fatalf("HealthCheckRem failed: %v", err)
	}

	for _, tc := range []struct {
		id          string
		description string
	}{
		{id: legacyID, description: "legacy ambiguous context"},
		{id: otherID, description: "other listener context"},
	} {
		ctxModel, err := db.FindContext(tc.id)
		if err != nil {
			t.Fatalf("FindContext(%s) failed: %v", tc.description, err)
		}
		pivot := ctxModel.Context.(*output.PivotingContext)
		if !pivot.Enable {
			t.Fatalf("%s was disabled by %s health check", tc.description, listenerA.Name)
		}
	}
}

func TestRegisterRemPreservesExistingDBAndBackfillsConfig(t *testing.T) {
	newRPCTestEnv(t)
	listener := core.NewListener("listener-rem-preserve", "127.0.0.1")
	core.Listeners.Add(listener)
	if err := configutil.SetStructByTag("listeners", &configs.ListenerConfig{
		Name: "listener-rem-preserve",
		REMs: []*configs.REMConfig{
			{Enable: true, Name: "rem-preserve", Console: "tcp://0.0.0.0:20001"},
		},
	}, "config"); err != nil {
		t.Fatalf("SetStructByTag failed: %v", err)
	}
	existing := &clientpb.Pipeline{
		Name:       "rem-preserve",
		ListenerId: listener.Name,
		Type:       consts.RemPipeline,
		Body: &clientpb.Pipeline_Rem{
			Rem: &clientpb.REM{
				Name:       "rem-preserve",
				ListenerId: listener.Name,
				Console:    "tcp://0.0.0.0:20000",
				Link:       "tcp://127.0.0.1:20000",
			},
		},
	}
	if _, err := db.SavePipeline(models.FromPipelinePb(existing)); err != nil {
		t.Fatalf("SavePipeline failed: %v", err)
	}

	_, err := (&Server{}).RegisterRem(context.Background(), &clientpb.Pipeline{
		Name:       "rem-preserve",
		ListenerId: listener.Name,
		Type:       consts.RemPipeline,
		Body: &clientpb.Pipeline_Rem{
			Rem: &clientpb.REM{
				Console: "tcp://0.0.0.0:29999",
			},
		},
	})
	if err != nil {
		t.Fatalf("RegisterRem failed: %v", err)
	}

	found, err := db.FindPipelineByListener("rem-preserve", listener.Name)
	if err != nil {
		t.Fatalf("FindPipelineByListener failed: %v", err)
	}
	if found.Console != "tcp://0.0.0.0:20000" || found.PipelineParams.Link != "tcp://127.0.0.1:20000" {
		t.Fatalf("DB REM was overwritten: console=%q link=%q", found.Console, found.PipelineParams.Link)
	}
	cfg := configs.GetListenerConfig()
	if cfg.REMs[0].Console != "tcp://0.0.0.0:20000" || cfg.REMs[0].Link != "tcp://127.0.0.1:20000" {
		t.Fatalf("config REM was not backfilled from DB: %#v", cfg.REMs[0])
	}
}

func TestRegisterRemRejectsColonName(t *testing.T) {
	newRPCTestEnv(t)
	core.Listeners.Add(core.NewListener("listener-rem-colon", "127.0.0.1"))

	_, err := (&Server{}).RegisterRem(context.Background(), &clientpb.Pipeline{
		Name:       "rem:bad",
		ListenerId: "listener-rem-colon",
		Type:       consts.RemPipeline,
		Body: &clientpb.Pipeline_Rem{
			Rem: &clientpb.REM{
				Name:       "rem:bad",
				ListenerId: "listener-rem-colon",
				Console:    "tcp://127.0.0.1:19001",
			},
		},
	})
	if err == nil {
		t.Fatal("RegisterRem should reject ':' in REM pipeline name")
	}
}

func TestSyncPipelineWritesRuntimeRemLinkToConfig(t *testing.T) {
	newRPCTestEnv(t)
	if err := configutil.SetStructByTag("listeners", &configs.ListenerConfig{
		Name: "listener-sync-rem",
		REMs: []*configs.REMConfig{
			{Enable: true, Name: "rem-sync", Console: "tcp://0.0.0.0:21000"},
		},
	}, "config"); err != nil {
		t.Fatalf("SetStructByTag failed: %v", err)
	}
	pipeline := &clientpb.Pipeline{
		Name:       "rem-sync",
		ListenerId: "listener-sync-rem",
		Type:       consts.RemPipeline,
		Body: &clientpb.Pipeline_Rem{
			Rem: &clientpb.REM{
				Name:       "rem-sync",
				ListenerId: "listener-sync-rem",
				Console:    "tcp://0.0.0.0:21000",
				Link:       "tcp://127.0.0.1:21000",
			},
		},
	}

	if _, err := (&Server{}).SyncPipeline(context.Background(), pipeline); err != nil {
		t.Fatalf("SyncPipeline failed: %v", err)
	}

	cfg := configs.GetListenerConfig()
	if cfg.REMs[0].Link != "tcp://127.0.0.1:21000" {
		t.Fatalf("runtime link was not synced to config: %#v", cfg.REMs[0])
	}
}

func seedRemRuntime(t testing.TB, name string) (*core.Listener, *clientpb.Pipeline) {
	t.Helper()

	return seedRemRuntimeWithListener(t, "listener-"+name, name)
}

func seedRemRuntimeWithListener(t testing.TB, listenerID, name string) (*core.Listener, *clientpb.Pipeline) {
	t.Helper()

	listener := core.NewListener(listenerID, "127.0.0.1")
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

func createPivotContext(t testing.TB, pipelineName, listenerID, agentID string, enabled bool) string {
	t.Helper()

	ctxModel := &models.Context{
		PipelineID: pipelineName,
		Type:       consts.ContextPivoting,
		Value: output.MarshalContext(&output.PivotingContext{
			Enable:      enabled,
			Listener:    listenerID,
			Pipeline:    pipelineName,
			RemAgentID:  agentID,
			InboundSide: "local",
			LocalURL:    "127.0.0.1:8080",
		}),
	}
	if err := db.Session().Create(ctxModel).Error; err != nil {
		t.Fatalf("Create context(%s) failed: %v", agentID, err)
	}
	return ctxModel.ID.String()
}
