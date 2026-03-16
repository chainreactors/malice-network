package core

import (
	"sync"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
)

// Bug #1: clientID is incremented with ++ (non-atomic).
// Under concurrent NewClient calls this is a data race.
// Run with: go test -race ./server/internal/core/... -run TestClientIDRaceUnderConcurrentCreation
func TestClientIDRaceUnderConcurrentCreation(t *testing.T) {
	withIsolatedClients(t)
	withIsolatedBroker(t)

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(n int) {
			defer wg.Done()
			c := NewClient("operator")
			Clients.Add(c)
			_ = c.ID
		}(i)
	}
	wg.Wait()

	clients := Clients.ActiveClients()
	if len(clients) != goroutines {
		t.Fatalf("active clients = %d, want %d", len(clients), goroutines)
	}
}

func TestClientsAddPublishesJoinEvent(t *testing.T) {
	withIsolatedClients(t)
	broker := withIsolatedBroker(t)

	c := NewClient("test-operator")
	Clients.Add(c)

	select {
	case evt := <-broker.publish:
		if evt.EventType != consts.EventJoin {
			t.Fatalf("event type = %q, want %q", evt.EventType, consts.EventJoin)
		}
		if !evt.Important {
			t.Fatal("join event should be marked important")
		}
		if evt.Client == nil || evt.Client.Name != "test-operator" {
			t.Fatalf("event client = %v, want name=test-operator", evt.Client)
		}
	default:
		t.Fatal("expected EventJoin to be published, but broker channel is empty")
	}
}

// Bug #8 fix: Clients.Remove now publishes EventLeft.
func TestClientsRemovePublishesLeftEvent(t *testing.T) {
	withIsolatedClients(t)
	broker := withIsolatedBroker(t)

	c := NewClient("leaving-operator")
	Clients.Add(c)
	<-broker.publish // drain the join event

	Clients.Remove(int(c.ID))

	select {
	case evt := <-broker.publish:
		if evt.EventType != consts.EventLeft {
			t.Fatalf("event type = %q, want %q", evt.EventType, consts.EventLeft)
		}
		if evt.Client.Name != "leaving-operator" {
			t.Fatalf("event client = %q, want leaving-operator", evt.Client.Name)
		}
	default:
		t.Fatal("Remove should publish EventLeft (bug #8 not fixed)")
	}

	if len(Clients.ActiveClients()) != 0 {
		t.Fatal("client should be removed from active map")
	}
}

func TestClientsRemoveNonExistentNoPanic(t *testing.T) {
	withIsolatedClients(t)
	withIsolatedBroker(t)

	// Should not panic on unknown IDs.
	Clients.Remove(999)
	Clients.Remove(0)
	Clients.Remove(-1)
}

func TestClientsActiveClientsExcludesOffline(t *testing.T) {
	withIsolatedClients(t)
	withIsolatedBroker(t)

	online := NewClient("online-op")
	offline := NewClient("offline-op")
	Clients.Add(online)
	Clients.Add(offline)

	offline.Online = false

	active := Clients.ActiveClients()
	if len(active) != 1 {
		t.Fatalf("active clients = %d, want 1 (only online)", len(active))
	}
	if active[0].Name != "online-op" {
		t.Fatalf("active client name = %q, want online-op", active[0].Name)
	}

	// But ActiveOperators still includes both (it doesn't check Online).
	operators := Clients.ActiveOperators()
	if len(operators) != 2 {
		t.Fatalf("active operators = %d, want 2 (ActiveOperators does not filter by Online)", len(operators))
	}
}

func TestClientsConcurrentAddRemove(t *testing.T) {
	withIsolatedClients(t)
	withIsolatedBroker(t)

	const n = 50
	var wg sync.WaitGroup
	wg.Add(n * 2)

	ids := make([]uint32, n)
	for i := 0; i < n; i++ {
		go func(idx int) {
			defer wg.Done()
			c := NewClient("op")
			ids[idx] = c.ID
			Clients.Add(c)
		}(i)
	}
	// Concurrently remove half of them (some may not exist yet — should not panic).
	for i := 0; i < n; i++ {
		go func(idx int) {
			defer wg.Done()
			Clients.Remove(idx + 1)
		}(i)
	}
	wg.Wait()
	// No assertion on count — just verifying no panic/race.
}

func TestClientToProtobuf(t *testing.T) {
	withIsolatedClients(t)

	c := NewClient("proto-op")
	pb := c.ToProtobuf()
	if pb.Name != "proto-op" || !pb.Online || pb.ID == 0 {
		t.Fatalf("ToProtobuf = %#v, want Name=proto-op Online=true ID>0", pb)
	}
}

func TestNewClientAssignsIncrementingID(t *testing.T) {
	withIsolatedClients(t)

	c1 := NewClient("op1")
	c2 := NewClient("op2")
	if c1.ID == 0 || c2.ID == 0 {
		t.Fatal("client ID should be non-zero")
	}
	if c2.ID <= c1.ID {
		t.Fatalf("client IDs should increment: c1=%d c2=%d", c1.ID, c2.ID)
	}
}
