package core

import (
	"strings"
	"sync/atomic"
	"testing"
	"time"

	iomclient "github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
)

func TestTaskMessageBufferRoundTrip(t *testing.T) {
	s := &Server{taskMessages: make(map[string]string)}
	task := &clientpb.Task{
		SessionId: "sess-1",
		TaskId:    7,
	}

	s.appendTaskMessage(task, []byte("first"))
	s.appendTaskMessage(task, []byte("second"))

	if got := s.popTaskMessage(task.SessionId, task.TaskId); got != "first\nsecond" {
		t.Fatalf("popTaskMessage = %q, want %q", got, "first\nsecond")
	}
	if got := s.popTaskMessage(task.SessionId, task.TaskId); got != "" {
		t.Fatalf("popTaskMessage should clear state, got %q", got)
	}
}

func TestAppendTaskMessageIgnoresEmptyInputs(t *testing.T) {
	s := &Server{taskMessages: make(map[string]string)}
	task := &clientpb.Task{
		SessionId: "sess-1",
		TaskId:    8,
	}

	s.appendTaskMessage(nil, []byte("ignored"))
	s.appendTaskMessage(task, nil)
	s.appendTaskMessage(task, []byte(""))

	if got := s.popTaskMessage(task.SessionId, task.TaskId); got != "" {
		t.Fatalf("unexpected buffered task message %q", got)
	}
}

func TestRenderEventAppliesColoringForSessionRegister(t *testing.T) {
	event := &clientpb.Event{
		Type:      consts.EventSession,
		Op:        consts.CtrlSessionRegister,
		Formatted: "session registered",
	}

	got := renderEvent(event)
	if !strings.Contains(got, event.Formatted) {
		t.Fatalf("renderEvent = %q, want to contain %q", got, event.Formatted)
	}
	if got == event.Formatted {
		t.Fatalf("renderEvent should decorate session register events")
	}
}

func TestRenderEventFallsBackToFormattedForUnknownTypes(t *testing.T) {
	event := &clientpb.Event{
		Type:      "custom",
		Op:        "noop",
		Formatted: "leave me alone",
	}

	if got := renderEvent(event); got != event.Formatted {
		t.Fatalf("renderEvent = %q, want %q", got, event.Formatted)
	}
}

func TestReconcileEventTracksWebsiteLifecycle(t *testing.T) {
	state := &iomclient.ServerState{
		Pipelines: make(map[string]*clientpb.Pipeline),
	}

	website := &clientpb.Pipeline{
		Name: "site-alpha",
		Type: consts.WebsitePipeline,
		Body: &clientpb.Pipeline_Web{
			Web: &clientpb.Website{
				Name:     "site-alpha",
				Root:     "/",
				Port:     8080,
				Contents: map[string]*clientpb.WebContent{},
			},
		},
	}

	state.ReconcileEvent(&clientpb.Event{
		Type: consts.EventJob,
		Op:   consts.CtrlWebsiteStart,
		Job: &clientpb.Job{
			Pipeline: website,
		},
	})

	if _, ok := state.Pipelines["site-alpha"]; !ok {
		t.Fatal("website start event should populate client pipeline cache")
	}

	state.ReconcileEvent(&clientpb.Event{
		Type: consts.EventJob,
		Op:   consts.CtrlWebsiteStop,
		Job: &clientpb.Job{
			Pipeline: website,
		},
	})

	if _, ok := state.Pipelines["site-alpha"]; ok {
		t.Fatal("website stop event should remove client pipeline cache entry")
	}
}

func TestReconcileEventTracksWebsiteContentMutations(t *testing.T) {
	state := &iomclient.ServerState{
		Pipelines: make(map[string]*clientpb.Pipeline),
	}

	base := &clientpb.Pipeline{
		Name: "site-content",
		Type: consts.WebsitePipeline,
		Body: &clientpb.Pipeline_Web{
			Web: &clientpb.Website{
				Name:     "site-content",
				Root:     "/",
				Port:     8080,
				Contents: map[string]*clientpb.WebContent{},
			},
		},
	}
	state.Pipelines[base.Name] = base

	added := &clientpb.WebContent{
		Id:        "content-1",
		WebsiteId: "site-content",
		Path:      "/index.html",
	}

	state.ReconcileEvent(&clientpb.Event{
		Type: consts.EventJob,
		Op:   consts.CtrlWebContentAdd,
		Job: &clientpb.Job{
			Pipeline: &clientpb.Pipeline{
				Name: "site-content",
				Type: consts.WebsitePipeline,
				Body: &clientpb.Pipeline_Web{
					Web: &clientpb.Website{
						Name: "site-content",
						Contents: map[string]*clientpb.WebContent{
							added.Path: added,
						},
					},
				},
			},
		},
	})

	if got := state.Pipelines["site-content"].GetWeb().Contents[added.Path]; got == nil || got.Id != added.Id {
		t.Fatalf("website content add event did not update client cache: %#v", got)
	}

	state.ReconcileEvent(&clientpb.Event{
		Type: consts.EventJob,
		Op:   consts.CtrlWebContentRemove,
		Job: &clientpb.Job{
			Pipeline: &clientpb.Pipeline{
				Name: "site-content",
				Type: consts.WebsitePipeline,
				Body: &clientpb.Pipeline_Web{
					Web: &clientpb.Website{
						Name: "site-content",
						Contents: map[string]*clientpb.WebContent{
							added.Path: {Path: added.Path},
						},
					},
				},
			},
		},
	})

	if _, ok := state.Pipelines["site-content"].GetWeb().Contents[added.Path]; ok {
		t.Fatal("website content remove event should evict content from client cache")
	}
}

func TestTriggerTaskDoneIgnoresMissingTask(t *testing.T) {
	s := &Server{taskMessages: make(map[string]string)}
	s.triggerTaskDone(&clientpb.Event{})
}

func TestTriggerTaskFinishIgnoresMissingTask(t *testing.T) {
	s := &Server{taskMessages: make(map[string]string)}
	s.triggerTaskFinish(&clientpb.Event{})
}

func TestHandlerEventIgnoresNilEvent(t *testing.T) {
	state := &iomclient.ServerState{
		EventHook:     map[iomclient.EventCondition][]iomclient.OnEventFunc{},
		EventCallback: map[string]func(*clientpb.Event){},
	}
	s := &Server{ServerState: state, taskMessages: make(map[string]string)}
	s.HandlerEvent(nil)
}

func TestHandlerSessionIgnoresMissingSession(t *testing.T) {
	s := &Server{taskMessages: make(map[string]string)}
	s.handlerSession(&clientpb.Event{})
}

func TestHandlerTaskIgnoresMissingTask(t *testing.T) {
	s := &Server{taskMessages: make(map[string]string)}
	s.handlerTask(&clientpb.Event{})
}

func TestRenderEventNilReturnsEmptyString(t *testing.T) {
	if got := renderEvent(nil); got != "" {
		t.Fatalf("renderEvent(nil) = %q, want empty string", got)
	}
}

func TestAddEventHookInitializesNilMap(t *testing.T) {
	s := &Server{ServerState: &iomclient.ServerState{}}
	cond := iomclient.EventCondition{Type: consts.EventBroadcast}

	s.AddEventHook(cond, func(*clientpb.Event) (bool, error) { return false, nil })

	if hooks := s.EventHook[cond]; len(hooks) != 1 {
		t.Fatalf("event hooks = %d, want 1", len(hooks))
	}
}

func TestDispatchEventHooksRemovesLuaVMDeadHookOnly(t *testing.T) {
	s := &Server{
		ServerState: &iomclient.ServerState{
			EventHook: map[iomclient.EventCondition][]iomclient.OnEventFunc{},
		},
	}
	cond := iomclient.EventCondition{Type: consts.EventBroadcast}

	deadCalls := atomic.Int32{}
	healthyCalls := atomic.Int32{}
	deadDone := make(chan struct{}, 1)
	healthyDone := make(chan struct{}, 2)

	deadHook := func(*clientpb.Event) (bool, error) {
		deadCalls.Add(1)
		select {
		case deadDone <- struct{}{}:
		default:
		}
		return false, ErrLuaVMDead
	}
	healthyHook := func(*clientpb.Event) (bool, error) {
		healthyCalls.Add(1)
		healthyDone <- struct{}{}
		return false, nil
	}

	s.AddEventHook(cond, deadHook)
	s.AddEventHook(cond, healthyHook)

	evt := &clientpb.Event{Type: consts.EventBroadcast, Op: "ping"}
	s.dispatchEventHooks(evt)

	select {
	case <-deadDone:
	case <-time.After(2 * time.Second):
		t.Fatal("dead hook did not execute")
	}
	select {
	case <-healthyDone:
	case <-time.After(2 * time.Second):
		t.Fatal("healthy hook did not execute")
	}

	deadline := time.After(2 * time.Second)
	for {
		s.eventHookMu.RLock()
		hooks := append([]iomclient.OnEventFunc(nil), s.EventHook[cond]...)
		s.eventHookMu.RUnlock()
		if len(hooks) == 1 {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("dead hook was not removed, hooks len = %d", len(hooks))
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	s.dispatchEventHooks(evt)
	select {
	case <-healthyDone:
	case <-time.After(2 * time.Second):
		t.Fatal("healthy hook did not execute on second dispatch")
	}

	if got := deadCalls.Load(); got != 1 {
		t.Fatalf("dead hook calls = %d, want 1", got)
	}
	if got := healthyCalls.Load(); got != 2 {
		t.Fatalf("healthy hook calls = %d, want 2", got)
	}
}
