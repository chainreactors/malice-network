package listener

import (
	"context"
	"errors"
	"testing"

	"github.com/chainreactors/IoM-go/proto/services/listenerrpc"
)

func TestListenerHandlerReturnsJobStreamOpenError(t *testing.T) {
	want := errors.New("job stream open failed")
	oldOpen := openListenerJobStream
	openListenerJobStream = func(listenerrpc.ListenerRPCClient, context.Context) (listenerrpc.ListenerRPC_JobStreamClient, error) {
		return nil, want
	}
	defer func() { openListenerJobStream = oldOpen }()

	lns := &listener{Name: "listener-a"}
	err := lns.Handler()
	if !errors.Is(err, want) {
		t.Fatalf("listener.Handler error = %v, want %v", err, want)
	}
}
