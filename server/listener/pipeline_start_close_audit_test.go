//go:build audit

package listener

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/implanttypes"
	"github.com/chainreactors/malice-network/server/internal/core"
	"google.golang.org/grpc"
)

var errAuditForwardUnavailable = errors.New("audit forward unavailable")

type auditUnavailablePipelineRPC struct{}

func (*auditUnavailablePipelineRPC) OpenForwardStream(context.Context, core.Pipeline) (core.ForwardStream, error) {
	return nil, errAuditForwardUnavailable
}
func (*auditUnavailablePipelineRPC) Register(context.Context, *clientpb.RegisterSession, ...grpc.CallOption) (*clientpb.Empty, error) {
	return &clientpb.Empty{}, nil
}
func (*auditUnavailablePipelineRPC) Checkin(context.Context, *implantpb.Ping, ...grpc.CallOption) (*clientpb.Empty, error) {
	return &clientpb.Empty{}, nil
}
func (*auditUnavailablePipelineRPC) GetArtifact(context.Context, *clientpb.Artifact, ...grpc.CallOption) (*clientpb.Artifact, error) {
	return nil, errAuditForwardUnavailable
}

func runAuditConcurrentStartClose(t *testing.T, startFn, closeFn func() error) {
	t.Helper()

	const iterations = 2000
	start := make(chan struct{})
	errCh := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		for range iterations {
			if err := startFn(); !errors.Is(err, errAuditForwardUnavailable) {
				errCh <- err
				return
			}
		}
	}()
	go func() {
		defer wg.Done()
		<-start
		for range iterations {
			if err := closeFn(); err != nil {
				errCh <- err
				return
			}
		}
	}()

	close(start)
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatalf("concurrent Start/Close returned an unexpected error: %v", err)
	}
}

func TestAuditTCPPipelineConcurrentStartClose(t *testing.T) {
	rpc := &auditUnavailablePipelineRPC{}
	pipeline, err := NewTcpPipeline(rpc, &clientpb.Pipeline{
		Name:       "audit-tcp-start-close",
		ListenerId: "audit-listener",
		Tls:        &clientpb.TLS{},
		Secure:     &clientpb.Secure{},
		Body: &clientpb.Pipeline_Tcp{Tcp: &clientpb.TCPPipeline{
			Host: "127.0.0.1",
			Port: 0,
		}},
	})
	if err != nil {
		t.Fatalf("NewTcpPipeline failed: %v", err)
	}
	runAuditConcurrentStartClose(t, pipeline.Start, pipeline.Close)
}

func TestAuditHTTPPipelineConcurrentStartClose(t *testing.T) {
	rpc := &auditUnavailablePipelineRPC{}
	pipeline, err := NewHttpPipeline(rpc, &clientpb.Pipeline{
		Name:       "audit-http-start-close",
		ListenerId: "audit-listener",
		Tls:        &clientpb.TLS{},
		Secure:     &clientpb.Secure{},
		Body: &clientpb.Pipeline_Http{Http: &clientpb.HTTPPipeline{
			Host:   "127.0.0.1",
			Port:   0,
			Params: (&implanttypes.PipelineParams{}).String(),
		}},
	})
	if err != nil {
		t.Fatalf("NewHttpPipeline failed: %v", err)
	}
	runAuditConcurrentStartClose(t, pipeline.Start, pipeline.Close)
}
