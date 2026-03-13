package core

import (
	"strings"
	"testing"

	"github.com/chainreactors/malice-network/client/assets"
)

func TestAIClientValidateUsesConfigAIHint(t *testing.T) {
	client := NewAIClient(&assets.AISettings{})

	err := client.validate()
	if err == nil {
		t.Fatal("expected validate to fail for disabled AI")
	}
	if !strings.Contains(err.Error(), "config ai --enable") {
		t.Fatalf("expected config ai hint, got %q", err.Error())
	}
	if strings.Contains(err.Error(), "ai-config") {
		t.Fatalf("unexpected legacy alias in error: %q", err.Error())
	}
}

func TestAIClientBuildEndpointUsesConfigAIHint(t *testing.T) {
	client := NewAIClient(&assets.AISettings{
		Enable:   true,
		APIKey:   "sk-test",
		Endpoint: "",
	})

	_, err := client.buildEndpoint("/chat/completions")
	if err == nil {
		t.Fatal("expected buildEndpoint to fail for empty endpoint")
	}
	if !strings.Contains(err.Error(), "config ai --endpoint <url>") {
		t.Fatalf("expected config ai hint, got %q", err.Error())
	}
	if strings.Contains(err.Error(), "ai-config") {
		t.Fatalf("unexpected legacy alias in error: %q", err.Error())
	}
}
