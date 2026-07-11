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
