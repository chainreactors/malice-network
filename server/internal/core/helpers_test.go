package core

import (
	"sync"
	"testing"
)

// brokerMu serializes access to EventBroker replacement to avoid racing
// with the background goroutine started in session_test.go init().
var brokerMu sync.Mutex

// withIsolatedClients saves and restores the global Clients and clientID counter.
func withIsolatedClients(t *testing.T) {
	t.Helper()

	oldClients := Clients
	oldClientID := clientID.Load()
	Clients = &clients{mutex: &sync.Mutex{}, active: map[int]*Client{}}
	clientID.Store(0)
	t.Cleanup(func() {
		Clients = oldClients
		clientID.Store(oldClientID)
	})
}

// withIsolatedJobs saves and restores the global Jobs, jobID, and ctrlID.
func withIsolatedJobs(t *testing.T) {
	t.Helper()

	oldJobs := Jobs.Map
	oldJobID := jobID.Load()
	oldCtrlID := ctrlID.Load()
	Jobs.Map = &sync.Map{}
	jobID.Store(0)
	ctrlID.Store(0)
	t.Cleanup(func() {
		Jobs.Map = oldJobs
		jobID.Store(oldJobID)
		ctrlID.Store(oldCtrlID)
	})
}

// withIsolatedListeners saves and restores the global Listeners map.
func withIsolatedListeners(t *testing.T) {
	t.Helper()

	oldMap := Listeners.Map
	Listeners.Map = &sync.Map{}
	t.Cleanup(func() {
		Listeners.Map = oldMap
	})
}

// withIsolatedBroker saves and restores the global EventBroker with a test broker.
// Uses brokerMu to prevent racing with the background broker goroutine from init().
func withIsolatedBroker(t *testing.T) *eventBroker {
	t.Helper()

	brokerMu.Lock()
	oldBroker := EventBroker
	broker := newTestBroker()
	EventBroker = broker
	brokerMu.Unlock()
	t.Cleanup(func() {
		brokerMu.Lock()
		EventBroker = oldBroker
		brokerMu.Unlock()
	})
	return broker
}
