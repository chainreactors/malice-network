//go:build integration

package main

import (
	"context"
	"testing"
	"time"

	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/testsupport"
	"google.golang.org/grpc/metadata"
)

func TestListenerCheckinRecoversDatabaseOnlySessionIntegration(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	h.SeedPipeline(t, h.NewTCPPipeline(t, "integration-core-recover-pipe"), true)
	sess := h.SeedSession(t, "integration-core-recover", "integration-core-recover-pipe", false)

	if _, err := core.Sessions.Get(sess.ID); err == nil {
		t.Fatal("seeded db-only session should not already be active in memory")
	}

	listenerConf := h.NewListenerClientConfig(t, h.ListenerID())
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := h.ConnectWithConfig(ctx, listenerConf)
	if err != nil {
		t.Fatalf("ConnectWithConfig failed: %v", err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
	})

	listenerClient := listenerrpc.NewListenerRPCClient(conn)
	checkinCtx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
		"session_id", sess.ID,
	))
	if _, err := listenerClient.Checkin(checkinCtx, &implantpb.Ping{Nonce: 1}); err != nil {
		t.Fatalf("listener Checkin failed: %v", err)
	}

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		_, err := core.Sessions.Get(sess.ID)
		return err == nil
	}, "db-only session recovery")

	recovered, err := core.Sessions.Get(sess.ID)
	if err != nil {
		t.Fatalf("recovered session lookup failed: %v", err)
	}
	if recovered.LastCheckinUnix() == 0 {
		t.Fatal("recovered session should have a refreshed last checkin timestamp")
	}

	stored, err := h.GetSession(sess.ID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if stored == nil || stored.SessionId != sess.ID {
		t.Fatalf("stored session = %#v, want session %s", stored, sess.ID)
	}
	if stored.LastCheckin == 0 {
		t.Fatal("database session should persist the refreshed last checkin timestamp")
	}
}
