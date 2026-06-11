//go:build integration

package main

import (
	"context"
	"testing"
	"time"

	iomclient "github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	clientcore "github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/server/testsupport"
)

func TestClientServerControlPlaneIntegration(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := h.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
	})

	server, err := clientcore.NewServer(conn, h.Admin)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	if server.Client == nil || server.Client.Name != "admin" {
		t.Fatalf("unexpected client identity: %#v", server.Client)
	}
	if server.Info == nil {
		t.Fatal("expected basic server info to be loaded")
	}
	if _, ok := server.Listeners["fixture-listener"]; !ok {
		t.Fatalf("expected fixture listener in initial state, got %#v", server.Listeners)
	}

	startEvents := make(chan *clientpb.Event, 1)
	stopEvents := make(chan *clientpb.Event, 1)
	server.AddEventHook(iomclient.EventCondition{
		Type:       consts.EventJob,
		Op:         consts.CtrlPipelineStart,
		PipelineId: "tcp-integration",
	}, func(event *clientpb.Event) (bool, error) {
		select {
		case startEvents <- event:
		default:
		}
		return true, nil
	})
	server.AddEventHook(iomclient.EventCondition{
		Type:       consts.EventJob,
		Op:         consts.CtrlPipelineStop,
		PipelineId: "tcp-integration",
	}, func(event *clientpb.Event) (bool, error) {
		select {
		case stopEvents <- event:
		default:
		}
		return true, nil
	})

	go server.EventHandler()
	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		return server.EventStatus
	}, "event stream to become active")

	pipeline := h.NewTCPPipeline(t, "tcp-integration")
	if _, err := server.Rpc.RegisterPipeline(context.Background(), pipeline); err != nil {
		t.Fatalf("RegisterPipeline failed: %v", err)
	}
	if _, err := server.Rpc.StartPipeline(context.Background(), &clientpb.CtrlPipeline{
		Name:       pipeline.Name,
		ListenerId: pipeline.ListenerId,
	}); err != nil {
		t.Fatalf("StartPipeline failed: %v", err)
	}
	testsupport.WaitForEvent(t, startEvents, "start pipeline event")

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		_, ok := server.Pipelines[pipeline.Name]
		return ok
	}, "client state to include started pipeline")

	if _, err := server.Rpc.StopPipeline(context.Background(), &clientpb.CtrlPipeline{
		Name:       pipeline.Name,
		ListenerId: pipeline.ListenerId,
		Pipeline:   pipeline,
	}); err != nil {
		t.Fatalf("StopPipeline failed: %v", err)
	}
	testsupport.WaitForEvent(t, stopEvents, "stop pipeline event")

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		_, ok := server.Pipelines[pipeline.Name]
		return !ok
	}, "event reconciliation to remove stopped pipeline")
}
