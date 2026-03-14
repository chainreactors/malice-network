//go:build bridge_agent_proto
// +build bridge_agent_proto

package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
)

// CallProvider forwards a BridgeLlmRequest to the specified LLM provider and returns BridgeLlmResponse.
// Configuration priority: opts fields > environment variables > provider presets.
func CallProvider(ctx context.Context, opts ProviderOpts, req *implantpb.BridgeLlmRequest) *implantpb.BridgeLlmResponse {
	baseURL, apiKey, err := resolve(opts)
	if err != nil {
		return &implantpb.BridgeLlmResponse{Error: err.Error()}
	}

	endpoint := strings.TrimSuffix(baseURL, "/") + "/chat/completions"
	reqCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	httpReq, err := http.NewRequestWithContext(
		reqCtx, "POST", endpoint, bytes.NewReader(req.GetData()),
	)
	if err != nil {
		return &implantpb.BridgeLlmResponse{Error: fmt.Sprintf("create request: %s", err)}
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return &implantpb.BridgeLlmResponse{Error: fmt.Sprintf("http request: %s", err)}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return &implantpb.BridgeLlmResponse{Error: fmt.Sprintf("read response: %s", err)}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &implantpb.BridgeLlmResponse{
			Error: fmt.Sprintf("API error (%d): %s", resp.StatusCode, string(respBody)),
		}
	}

	// Wrap raw OpenAI response as {"payload": <response>} for the implant's BridgeProvider.
	bridgeResp := map[string]json.RawMessage{
		"payload": json.RawMessage(respBody),
	}
	wrappedBytes, err := json.Marshal(bridgeResp)
	if err != nil {
		return &implantpb.BridgeLlmResponse{Error: fmt.Sprintf("wrap response: %s", err)}
	}

	return &implantpb.BridgeLlmResponse{Data: wrappedBytes}
}
