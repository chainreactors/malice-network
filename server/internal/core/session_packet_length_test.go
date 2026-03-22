package core

import (
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/gookit/config/v2"
)

func TestGetPacketLengthWithPipelineConfig(t *testing.T) {
	withIsolatedListeners(t)
	withIsolatedBroker(t)

	config.Set(consts.ConfigMaxPacketLength, 10485760)
	t.Cleanup(func() { config.Set(consts.ConfigMaxPacketLength, 0) })

	listener := NewListener("test-lis", "10.0.0.1")
	Listeners.Add(listener)
	listener.AddPipeline(&clientpb.Pipeline{
		Name:         "pipe-custom",
		ListenerId:   "test-lis",
		PacketLength: 2048,
	})

	sess := newTestSession("pkt-test")
	sess.PipelineID = "pipe-custom"

	got := sess.GetPacketLength()
	if got != 2048 {
		t.Fatalf("GetPacketLength() = %d, want 2048", got)
	}
}

func TestGetPacketLengthFallsBackToGlobal(t *testing.T) {
	withIsolatedListeners(t)
	withIsolatedBroker(t)

	config.Set(consts.ConfigMaxPacketLength, 10485760)
	t.Cleanup(func() { config.Set(consts.ConfigMaxPacketLength, 0) })

	listener := NewListener("test-lis2", "10.0.0.1")
	Listeners.Add(listener)
	listener.AddPipeline(&clientpb.Pipeline{
		Name:       "pipe-default",
		ListenerId: "test-lis2",
		// PacketLength is 0 (unset)
	})

	sess := newTestSession("pkt-fallback")
	sess.PipelineID = "pipe-default"

	got := sess.GetPacketLength()
	if got != 10485760 {
		t.Fatalf("GetPacketLength() = %d, want 10485760 (global default)", got)
	}
}

func TestGetPacketLengthNoPipeline(t *testing.T) {
	withIsolatedListeners(t)
	withIsolatedBroker(t)

	config.Set(consts.ConfigMaxPacketLength, 10485760)
	t.Cleanup(func() { config.Set(consts.ConfigMaxPacketLength, 0) })

	sess := newTestSession("pkt-nopipe")
	sess.PipelineID = "nonexistent"

	got := sess.GetPacketLength()
	if got != 10485760 {
		t.Fatalf("GetPacketLength() = %d, want 10485760 (global default)", got)
	}
}
