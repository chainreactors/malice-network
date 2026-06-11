package agent

import (
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
)

// BridgeAgentChat sends a BridgeAgentRequest carrying the selected provider/model from config ai.
func BridgeAgentChat(rpc clientrpc.MaliceRPCClient, sess *client.Session,
	text, model, provider string, maxTurns uint32) (*clientpb.Task, error) {
	task, err := rpc.BridgeAgentChat(sess.Context(), &implantpb.BridgeAgentRequest{
		Session:  sess.SessionId,
		Text:     text,
		Model:    model,
		Provider: provider,
		MaxTurns: maxTurns,
	})
	if err != nil {
		return nil, err
	}
	return task, nil
}

func BridgeAgentAvailable() bool {
	return true
}
