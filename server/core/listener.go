package core

import "github.com/chainreactors/malice-network/proto/services/listenerrpc"

// Listener manager listener
type Listener struct {
	Forwarders []*Forward
	Rpc        listenerrpc.ListenerRPCClient
	// Connections
	// Sessions
}
