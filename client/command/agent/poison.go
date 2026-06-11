package agent

import (
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
)

// ExecuteBridgeChat sends a natural-language chat request through the bridge-backed
// session path. The bridge injects the message into the hijacked agent session
// and streams back observe events as LLMEvent spites.
func ExecuteBridgeChat(rpc clientrpc.MaliceRPCClient, sess *client.Session, text string) (*clientpb.Task, error) {
	task, err := rpc.ExecuteModule(sess.Context(), &implantpb.ExecuteModuleRequest{
		Spite: &implantpb.Spite{
			Name: ModuleChat,
			Body: &implantpb.Spite_Request{
				Request: &implantpb.Request{
					Name:  ModuleChat,
					Input: text,
				},
			},
		},
		Expect: "llm.observe",
	})
	if err != nil {
		return nil, err
	}
	return task, nil
}
