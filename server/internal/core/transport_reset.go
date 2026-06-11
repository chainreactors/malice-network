package core

import "sync"

// ResetTransientTransportState clears the in-memory listener transport state.
// It is primarily used by tests that start and stop isolated control planes in
// the same process.
func ResetTransientTransportState() {
	if Forwarders != nil && Forwarders.forwarders != nil {
		ids := make([]string, 0)
		Forwarders.forwarders.Range(func(key, value any) bool {
			if id, ok := key.(string); ok {
				ids = append(ids, id)
			}
			return true
		})
		for _, id := range ids {
			_ = Forwarders.Remove(id)
		}
	}

	if Connections != nil && Connections.connections != nil {
		ids := make([]string, 0)
		Connections.connections.Range(func(key, value any) bool {
			if id, ok := key.(string); ok {
				ids = append(ids, id)
			}
			return true
		})
		for _, id := range ids {
			Connections.remove(id, ErrConnectionRemoved)
		}
	}

	Forwarders = &forwarders{forwarders: &sync.Map{}}
	Connections = &connections{connections: &sync.Map{}}
	ListenerSessions = &listenerSessions{sessions: &sync.Map{}}
}
