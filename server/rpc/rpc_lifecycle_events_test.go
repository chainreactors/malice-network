package rpc

import (
	"context"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
)

func TestLifecycleEventsPublishStructuredPayloads(t *testing.T) {
	oldBroker := core.EventBroker
	oldTicker := core.GlobalTicker
	defer func() {
		core.EventBroker = oldBroker
		core.GlobalTicker = oldTicker
	}()

	testTicker := core.NewTicker()
	defer testTicker.RemoveAll()
	core.GlobalTicker = testTicker

	broker := core.NewBroker()
	defer broker.Stop()
	waitEventBrokerReady(t, broker)
	sub := subscribeEventBrokerReady(t, broker)
	defer broker.Unsubscribe(sub)

	publishListenerIdentityEvent(consts.CtrlListenerRegister, "listener-a", "10.0.0.1:5004")
	listenerEvent := waitForLifecycleEvent(t, sub, consts.CtrlListenerRegister)
	if listenerEvent.EventType != consts.EventListener {
		t.Fatalf("listener event type = %q, want %q", listenerEvent.EventType, consts.EventListener)
	}
	if listenerEvent.Listener == nil || listenerEvent.Listener.GetId() != "listener-a" || listenerEvent.Listener.GetIp() != "10.0.0.1:5004" {
		t.Fatalf("unexpected listener event payload: %#v", listenerEvent.Listener)
	}

	publishWebsiteLifecycleEvent(consts.CtrlWebsiteDelete, "website-a", "listener-a", nil)
	websiteEvent := waitForLifecycleEvent(t, sub, consts.CtrlWebsiteDelete)
	if websiteEvent.EventType != consts.EventWebsite {
		t.Fatalf("website event type = %q, want %q", websiteEvent.EventType, consts.EventWebsite)
	}
	if websiteEvent.Job == nil || websiteEvent.Job.GetPipeline() == nil {
		t.Fatalf("website event missing pipeline payload: %#v", websiteEvent.Job)
	}
	pipeline := websiteEvent.Job.GetPipeline()
	if pipeline.GetName() != "website-a" || pipeline.GetListenerId() != "listener-a" || pipeline.GetType() != consts.WebsitePipeline {
		t.Fatalf("unexpected website event payload: %#v", pipeline)
	}
}

func TestCertificateMutationsPublishLifecycleEvents(t *testing.T) {
	newRPCTestEnv(t)
	if err := db.SaveCertificate(&models.Certificate{
		Name:    "certificate-event",
		Type:    "imported",
		CertPEM: "old-cert",
		KeyPEM:  "old-key",
	}); err != nil {
		t.Fatalf("SaveCertificate: %v", err)
	}
	events := subscribeEventBrokerReady(t, core.EventBroker)
	defer core.EventBroker.Unsubscribe(events)

	server := &Server{}
	if _, err := server.UpdateCertificate(context.Background(), &clientpb.TLS{Cert: &clientpb.Cert{
		Name:    "certificate-event",
		Cert:    "new-cert",
		Key:     "new-key",
		Comment: "updated",
	}}); err != nil {
		t.Fatalf("UpdateCertificate: %v", err)
	}
	updateEvent := waitForLifecycleEvent(t, events, consts.CtrlCertUpdate)
	if updateEvent.EventType != consts.EventCert {
		t.Fatalf("certificate update event type = %q, want %q", updateEvent.EventType, consts.EventCert)
	}

	if _, err := server.DeleteCertificate(context.Background(), &clientpb.Cert{Name: "certificate-event"}); err != nil {
		t.Fatalf("DeleteCertificate: %v", err)
	}
	deleteEvent := waitForLifecycleEvent(t, events, consts.CtrlCertDelete)
	if deleteEvent.EventType != consts.EventCert {
		t.Fatalf("certificate delete event type = %q, want %q", deleteEvent.EventType, consts.EventCert)
	}
}

func TestNewProfilePublishesLifecycleEvent(t *testing.T) {
	newRPCTestEnv(t)
	events := subscribeEventBrokerReady(t, core.EventBroker)
	defer core.EventBroker.Unsubscribe(events)

	if _, err := (&Server{}).NewProfile(context.Background(), &clientpb.Profile{Name: "profile-event"}); err != nil {
		t.Fatalf("NewProfile: %v", err)
	}
	event := waitForLifecycleEvent(t, events, consts.CtrlProfileCreate)
	if event.EventType != consts.EventProfile {
		t.Fatalf("profile create event type = %q, want %q", event.EventType, consts.EventProfile)
	}
}

func TestHandleJobStatusDefersPersistedLifecycleEvents(t *testing.T) {
	newRPCTestEnv(t)
	listener := core.NewListener("lifecycle-status-listener", "127.0.0.1")
	events := subscribeEventBrokerReady(t, core.EventBroker)
	defer core.EventBroker.Unsubscribe(events)

	for _, operation := range []string{
		consts.CtrlPipelineStart,
		consts.CtrlPipelineStop,
		consts.CtrlRemStart,
		consts.CtrlRemStop,
		consts.CtrlWebsiteStart,
		consts.CtrlWebsiteStop,
		consts.CtrlWebContentAdd,
		consts.CtrlWebContentUpdate,
		consts.CtrlWebContentRemove,
	} {
		ctrlID := listener.PushCtrlDeferredEvent(context.Background(), &clientpb.JobCtrl{Ctrl: operation})
		if ctrlID == 0 {
			t.Fatalf("failed to queue deferred %s control", operation)
		}
		<-listener.Ctrl
		handleJobStatus(listener, &clientpb.JobStatus{
			CtrlId: ctrlID,
			Ctrl:   operation,
			Status: consts.CtrlStatusSuccess,
			Job: &clientpb.Job{Pipeline: &clientpb.Pipeline{
				Name:       "lifecycle-status-pipeline",
				ListenerId: listener.Name,
			}},
		})
		assertNoLifecycleEvent(t, events, operation, 20*time.Millisecond)
	}
}

func TestHandleJobStatusPublishesUnmanagedLifecycleEvents(t *testing.T) {
	newRPCTestEnv(t)
	listener := core.NewListener("recovery-status-listener", "127.0.0.1")
	events := subscribeEventBrokerReady(t, core.EventBroker)
	defer core.EventBroker.Unsubscribe(events)

	for _, operation := range []string{
		consts.CtrlPipelineStart,
		consts.CtrlPipelineStop,
		consts.CtrlRemStart,
		consts.CtrlRemStop,
		consts.CtrlWebsiteStart,
		consts.CtrlWebsiteStop,
		consts.CtrlWebContentAdd,
		consts.CtrlWebContentUpdate,
		consts.CtrlWebContentRemove,
	} {
		handleJobStatus(listener, &clientpb.JobStatus{
			Ctrl:   operation,
			Status: consts.CtrlStatusSuccess,
			Job: &clientpb.Job{Pipeline: &clientpb.Pipeline{
				Name:       "recovery-status-pipeline",
				ListenerId: listener.Name,
			}},
		})
		event := waitForLifecycleEvent(t, events, operation)
		if event.EventType != consts.EventJob {
			t.Fatalf("unmanaged %s event type = %q, want %q", operation, event.EventType, consts.EventJob)
		}
	}
}

func TestPersistedLifecycleControlsPublishCommittedState(t *testing.T) {
	tests := []struct {
		name         string
		operation    string
		pipelineType string
		start        bool
		eventType    string
	}{
		{name: "pipeline start", operation: consts.CtrlPipelineStart, pipelineType: consts.TCPPipeline, start: true, eventType: consts.EventJob},
		{name: "pipeline stop", operation: consts.CtrlPipelineStop, pipelineType: consts.TCPPipeline, eventType: consts.EventJob},
		{name: "rem start", operation: consts.CtrlRemStart, pipelineType: consts.RemPipeline, start: true, eventType: consts.EventJob},
		{name: "rem stop", operation: consts.CtrlRemStop, pipelineType: consts.RemPipeline, eventType: consts.EventJob},
		{name: "website start", operation: consts.CtrlWebsiteStart, pipelineType: consts.WebsitePipeline, start: true, eventType: consts.EventWebsite},
		{name: "website stop", operation: consts.CtrlWebsiteStop, pipelineType: consts.WebsitePipeline, eventType: consts.EventWebsite},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newRPCTestEnv(t)
			listenerID := "listener-" + tt.operation
			pipelineName := "pipeline-" + tt.operation
			listener := core.NewListener(listenerID, "127.0.0.1")
			core.Listeners.Add(listener)

			pipeline := lifecycleTestPipeline(pipelineName, listenerID, tt.pipelineType, !tt.start)
			if _, err := db.SavePipeline(models.FromPipelinePb(pipeline)); err != nil {
				t.Fatalf("SavePipeline failed: %v", err)
			}
			if !tt.start {
				listener.AddPipeline(pipeline)
				if tt.pipelineType == consts.WebsitePipeline {
					core.Jobs.AddPipeline(pipeline)
				}
			}

			events := subscribeEventBrokerReady(t, core.EventBroker)
			defer core.EventBroker.Unsubscribe(events)
			result := make(chan error, 1)
			go func() {
				req := &clientpb.CtrlPipeline{Name: pipelineName, ListenerId: listenerID}
				var err error
				switch tt.operation {
				case consts.CtrlPipelineStart:
					_, err = (&Server{}).StartPipeline(context.Background(), req)
				case consts.CtrlPipelineStop:
					_, err = (&Server{}).StopPipeline(context.Background(), req)
				case consts.CtrlRemStart:
					_, err = (&Server{}).StartRem(context.Background(), req)
				case consts.CtrlRemStop:
					_, err = (&Server{}).StopRem(context.Background(), req)
				case consts.CtrlWebsiteStart:
					_, err = (&Server{}).StartWebsite(context.Background(), req)
				case consts.CtrlWebsiteStop:
					_, err = (&Server{}).StopWebsite(context.Background(), req)
				}
				result <- err
			}()

			var ctrl *clientpb.JobCtrl
			select {
			case ctrl = <-listener.Ctrl:
			case <-time.After(2 * time.Second):
				t.Fatal("timed out waiting for lifecycle control")
			}
			listener.CtrlJob.Store(ctrl.Id, nil)
			handleJobStatus(listener, &clientpb.JobStatus{
				CtrlId: ctrl.Id,
				Ctrl:   ctrl.Ctrl,
				Status: consts.CtrlStatusSuccess,
				Job:    ctrl.Job,
			})
			if err := <-result; err != nil {
				t.Fatalf("lifecycle RPC failed: %v", err)
			}

			event := waitForLifecycleEvent(t, events, tt.operation)
			if event.EventType != tt.eventType {
				t.Fatalf("event type = %q, want %q", event.EventType, tt.eventType)
			}
			if event.Job.GetPipeline().GetEnable() != tt.start {
				t.Fatalf("event enable = %v, want %v", event.Job.GetPipeline().GetEnable(), tt.start)
			}
			stored, err := db.FindPipelineByListener(pipelineName, listenerID)
			if err != nil {
				t.Fatalf("FindPipelineByListener failed: %v", err)
			}
			if stored.Enable != tt.start {
				t.Fatalf("persisted enable = %v, want %v", stored.Enable, tt.start)
			}
		})
	}
}

func lifecycleTestPipeline(name, listenerID, pipelineType string, enabled bool) *clientpb.Pipeline {
	pipeline := &clientpb.Pipeline{
		Name:       name,
		ListenerId: listenerID,
		Type:       pipelineType,
		Enable:     enabled,
	}
	switch pipelineType {
	case consts.RemPipeline:
		pipeline.Body = &clientpb.Pipeline_Rem{Rem: &clientpb.REM{
			Name:       name,
			ListenerId: listenerID,
			Console:    "tcp://127.0.0.1:12345",
		}}
	case consts.WebsitePipeline:
		pipeline.Body = &clientpb.Pipeline_Web{Web: &clientpb.Website{
			Name:       name,
			ListenerId: listenerID,
			Root:       "/",
			Port:       8080,
			Contents:   map[string]*clientpb.WebContent{},
		}}
	default:
		pipeline.Body = &clientpb.Pipeline_Tcp{Tcp: &clientpb.TCPPipeline{
			Name:       name,
			ListenerId: listenerID,
			Host:       "127.0.0.1",
			Port:       4444,
		}}
	}
	return pipeline
}

func waitForLifecycleEvent(t *testing.T, events <-chan core.Event, operation string) core.Event {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case event := <-events:
			if event.Op == operation {
				return event
			}
		case <-deadline:
			t.Fatalf("timed out waiting for %s event", operation)
		}
	}
}

func assertNoLifecycleEvent(t *testing.T, events <-chan core.Event, operation string, duration time.Duration) {
	t.Helper()
	timer := time.NewTimer(duration)
	defer timer.Stop()
	for {
		select {
		case event := <-events:
			if event.Op == operation {
				t.Fatalf("received lifecycle event %q before persisted state commit: %#v", operation, event)
			}
		case <-timer.C:
			return
		}
	}
}
