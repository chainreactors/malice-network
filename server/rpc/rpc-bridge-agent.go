//go:build bridge_agent_proto
// +build bridge_agent_proto

package rpc

import (
	"context"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/internal/llm"
)

// BridgeAgentChat handles the bridge agent RPC: operator -> implant -> [agent loop with LLM proxy] -> result.
// LLM provider config is passed from the client's config ai settings via BridgeAgentRequest.
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

	// LLM config from client's config ai, passed through BridgeAgentRequest
	providerOpts := llm.ProviderOpts{
		Provider: req.GetProvider(),
		APIKey:   req.GetApiKey(),
		Endpoint: req.GetEndpoint(),
	}

	runTaskHandler(greq.Task, func() error {
		for resp := range out {
			// BridgeLlmRequest: implant is asking for an LLM completion
			if llmReq := resp.GetBridgeLlmRequest(); llmReq != nil {
				llmResp := llm.CallProvider(providerOpts, llmReq)
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

				if err := types.AssertSpite(resp, types.MsgBridgeAgentResponse); err != nil {
					greq.Task.Panic(buildErrorEvent(greq.Task, err))
					return nil
				}
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
