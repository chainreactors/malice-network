package rpc

import (
	"context"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/internal/llm"
)

// BridgeAgentChat handles the bridge agent RPC: operator -> implant -> [agent loop with LLM proxy] -> result.
// The client selects provider/model; endpoint/api key/proxy are resolved server-side.
func (rpc *Server) BridgeAgentChat(ctx context.Context, req *implantpb.BridgeAgentRequest) (*clientpb.Task, error) {
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	greq.Count = -1 // streaming mode

	in, out, err := rpc.StreamGenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	// Provider selection comes from the client; server-side config resolves endpoint/api key/proxy.
	providerOpts := llm.ProviderOpts{
		Provider: req.GetProvider(),
	}

	runTaskHandler(greq.Task, func() error {
		for resp := range out {
			// BridgeLlmRequest: implant is asking for an LLM completion
			if llmReq := resp.GetBridgeLlmRequest(); llmReq != nil {
				llmResp := llm.CallProvider(greq.Task.Ctx, providerOpts, llmReq)
				reply, buildErr := greq.NewSpite(llmResp)
				if buildErr != nil {
					logs.Log.Errorf("bridge agent: build spite error: %s", buildErr)
					continue
				}
				reply.TaskId = greq.Task.Id
				if err := in.Send(reply); err != nil {
					logs.Log.Errorf("bridge agent: send llm response error: %s", err)
					return err
				}
				continue
			}

			// BridgeAgentResponse: agent loop is done
			if agentResp := resp.GetBridgeAgentResponse(); agentResp != nil {
				if len(agentResp.AvailableTools) > 0 {
					names := make([]string, len(agentResp.AvailableTools))
					for i, t := range agentResp.AvailableTools {
						names[i] = t.Name
					}
					logs.Log.Infof("bridge agent tools: %v", names)
				}

				resp.Name = consts.ModuleChat
				if err := greq.HandlerSpite(resp); err != nil {
					logs.Log.Errorf("bridge agent: handler spite error: %s", err)
					return err
				}
				greq.Task.Finish(resp, "")
				return nil
			}

			logs.Log.Warnf("bridge agent: unexpected message type from implant")
		}
		return nil
	}, in.Close, greq.Task.Close)

	return greq.Task.ToProtobuf(), nil
}
