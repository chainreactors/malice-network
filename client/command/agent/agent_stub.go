//go:build !bridge_agent_proto
// +build !bridge_agent_proto

package agent

import (
	"errors"

	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
)

const ModuleBridgeAgent = "bridge_agent"

var errBridgeAgentUnavailable = errors.New("bridge agent is unavailable in this build: current proto definitions do not include the required RPC/messages")

func BridgeAgentAvailable() bool {
	return false
}

func ChatCmd(cmd *cobra.Command, con *core.Console, args []string) error {
	return errBridgeAgentUnavailable
}

func BridgeAgentChat(rpc clientrpc.MaliceRPCClient, sess *client.Session,
	text, model, provider, apiKey, endpoint string, maxTurns uint32) (*clientpb.Task, error) {
	return nil, errBridgeAgentUnavailable
}

func RegisterBridgeAgentFunc(con *core.Console) {
}

func hasModule(sess *client.Session, name string) bool {
	if sess == nil {
		return false
	}
	for _, mod := range sess.Modules {
		if mod == name {
			return true
		}
	}
	return false
}
