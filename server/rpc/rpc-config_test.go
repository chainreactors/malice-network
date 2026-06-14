package rpc

import (
	"context"
	"testing"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/configs"
)

func TestSaasConfigRPCUpdatesRuntimeConfig(t *testing.T) {
	configs.InitTestConfigRuntime(t)

	server := NewServer()
	req := &clientpb.SaasConfig{
		Enable: true,
		Url:    "https://saas.example/api",
		Token:  "saas-token",
	}

	if _, err := server.UpdateSaasConfig(context.Background(), req); err != nil {
		t.Fatalf("UpdateSaasConfig failed: %v", err)
	}

	got, err := server.GetSaasConfig(context.Background(), &clientpb.Empty{})
	if err != nil {
		t.Fatalf("GetSaasConfig failed: %v", err)
	}
	if got.GetEnable() != req.GetEnable() || got.GetUrl() != req.GetUrl() || got.GetToken() != req.GetToken() {
		t.Fatalf("GetSaasConfig = %#v, want %#v", got, req)
	}
}

func TestGetSaasConfigReturnsEmptyWhenMissing(t *testing.T) {
	configs.InitTestConfigRuntime(t)

	got, err := NewServer().GetSaasConfig(context.Background(), &clientpb.Empty{})
	if err != nil {
		t.Fatalf("GetSaasConfig failed: %v", err)
	}
	if got == nil || got.GetEnable() || got.GetUrl() != "" || got.GetToken() != "" {
		t.Fatalf("GetSaasConfig = %#v, want empty config", got)
	}
}
