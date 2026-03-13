//go:build bridge_agent_proto
// +build bridge_agent_proto

package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestCallCompletionLive tests a real API call to verify HTTP layer works.
// Skipped in short mode. Requires bridge_agent_proto build tag.
func TestCallCompletionLive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live API test in short mode")
	}

	baseURL, apiKey, err := resolve(ProviderOpts{
		Provider: "openai",
		APIKey:   "sk-Kdp7jXbyICmcCh7k6",
		Endpoint: "https://wafcdn.aimeeting.store/v1",
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	reqBody := map[string]any{
		"model": "gpt-5.4",
		"messages": []map[string]string{
			{"role": "user", "content": "Say 'hello' and nothing else."},
		},
		"max_tokens": 16,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	endpoint := strings.TrimSuffix(baseURL, "/") + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(
		context.Background(), "POST", endpoint, bytes.NewReader(bodyBytes),
	)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		t.Fatalf("http request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		t.Fatalf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	choices, ok := result["choices"].([]any)
	if !ok || len(choices) == 0 {
		t.Fatalf("expected non-empty choices, got: %s", string(respBody))
	}

	choice := choices[0].(map[string]any)
	message := choice["message"].(map[string]any)
	content, ok := message["content"].(string)
	if !ok || content == "" {
		t.Fatalf("expected non-empty content, got: %v", message)
	}

	t.Logf("API response: %s", content)
}
