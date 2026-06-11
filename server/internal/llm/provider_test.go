package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/gookit/config/v2"
)

// TestCallCompletionLive tests a real API call to verify HTTP layer works.
// Skipped in short mode. Requires bridge_agent_proto build tag.
func TestCallCompletionLive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live API test in short mode")
	}

	cfg := loadLiveTestConfig(t)
	resolved, err := resolve(ProviderOpts{
		Provider: "openai",
		APIKey:   cfg.APIKey,
		Endpoint: cfg.BaseURL,
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	reqBody := map[string]any{
		"model": cfg.Model,
		"messages": []map[string]string{
			{"role": "user", "content": "Say 'hello' and nothing else."},
		},
		"max_tokens": 16,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	endpoint := strings.TrimSuffix(resolved.BaseURL, "/") + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(
		context.Background(), "POST", endpoint, bytes.NewReader(bodyBytes),
	)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+resolved.APIKey)

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

func TestCallProviderLiveFromServerConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live API test in short mode")
	}

	cfg := loadLiveTestConfig(t)
	initLiveConfigRuntime(t)
	config.Set("server.llm.default_provider", "openai")
	config.Set("server.llm.providers.openai.endpoint", cfg.BaseURL)
	config.Set("server.llm.providers.openai.api_key", cfg.APIKey)

	reqBody := map[string]any{
		"model": cfg.Model,
		"messages": []map[string]string{
			{"role": "user", "content": "Reply with the single word ok."},
		},
		"max_tokens": 8,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	resp := CallProvider(context.Background(), ProviderOpts{Provider: "openai"}, &implantpb.BridgeLlmRequest{Data: bodyBytes})
	if resp.GetError() != "" {
		t.Fatalf("CallProvider returned error: %s", resp.GetError())
	}

	var wrapped map[string]json.RawMessage
	if err := json.Unmarshal(resp.GetData(), &wrapped); err != nil {
		t.Fatalf("unmarshal wrapped response: %v", err)
	}
	if len(wrapped["payload"]) == 0 {
		t.Fatalf("wrapped response missing payload: %s", string(resp.GetData()))
	}
}

func TestNewHTTPClientUsesProxyConfig(t *testing.T) {
	resolved := ResolvedProvider{
		ProxyURL: "http://127.0.0.1:8080",
		Timeout:  15 * time.Second,
	}

	client, err := newHTTPClient(resolved)
	if err != nil {
		t.Fatalf("newHTTPClient returned error: %v", err)
	}
	if client.Timeout != 15*time.Second {
		t.Fatalf("client.Timeout = %s, want %s", client.Timeout, 15*time.Second)
	}

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("client.Transport = %T, want *http.Transport", client.Transport)
	}
	req := &http.Request{URL: mustParseURL(t, "https://example.com")}
	proxyURL, err := transport.Proxy(req)
	if err != nil {
		t.Fatalf("transport.Proxy returned error: %v", err)
	}
	if proxyURL == nil || proxyURL.String() != "http://127.0.0.1:8080" {
		t.Fatalf("proxyURL = %v, want %q", proxyURL, "http://127.0.0.1:8080")
	}
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()

	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("url.Parse(%q): %v", raw, err)
	}
	return u
}
