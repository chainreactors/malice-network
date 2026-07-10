//go:build audit

package listener

import (
	"errors"
	"io"
	"testing"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
)

func TestAuditForwardSendEventRejectsAfterClose(t *testing.T) {
	accepted := 0
	for range 64 {
		stream := newAuditForwardLocalStream(1, 1)
		stream.close()
		err := stream.sendEvent(&clientpb.SpiteRequest{ListenerId: "after-close"})
		if err == nil {
			accepted++
			continue
		}
		if !errors.Is(err, io.ErrClosedPipe) {
			t.Fatalf("sendEvent error = %v, want %v", err, io.ErrClosedPipe)
		}
	}

	// Both select branches are ready for a closed stream with an empty queue.
	// Missing all invalid accepts therefore has probability 2^-64.
	if accepted > 0 {
		t.Fatalf("sendEvent accepted %d of 64 events after close", accepted)
	}
}
