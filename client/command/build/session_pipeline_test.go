package build

import (
	"testing"

	iomclient "github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
)

func TestScopedSessionPipelineIDUsesListener(t *testing.T) {
	sess := &iomclient.Session{Session: &clientpb.Session{PipelineId: "pipe-a", ListenerId: "listener-a"}}
	if got := scopedSessionPipelineID(sess); got != "listener-a:pipe-a" {
		t.Fatalf("scopedSessionPipelineID = %q, want listener-a:pipe-a", got)
	}
}

func TestScopedSessionPipelineIDAllowsLegacySession(t *testing.T) {
	sess := &iomclient.Session{Session: &clientpb.Session{PipelineId: "pipe-a"}}
	if got := scopedSessionPipelineID(sess); got != "pipe-a" {
		t.Fatalf("scopedSessionPipelineID = %q, want pipe-a", got)
	}
}
