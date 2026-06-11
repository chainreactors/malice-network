package rpc

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// TestClientDefaultSendMessageSize verifies that a gRPC client whose
// DialOptions only set MaxCallRecvMsgSize (the pattern used by
// external/IoM-go/mtls/mtls.go) is NOT capped at 4 MiB on the send side.
// grpc-go's defaultClientMaxSendMessageSize is math.MaxInt32, so artifact
// uploads up to the server-side MaxRecvMsgSize should succeed without any
// extra MaxCallSendMsgSize option.
//
// This test exists as a regression guard against the (incorrect) belief that
// client send is capped at 4 MiB by default — only client receive is.
func TestClientDefaultSendMessageSize(t *testing.T) {
	const payloadSize = 8 << 20 // 8 MiB — larger than the 4 MiB recv default

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer lis.Close()

	// Server accepts up to 16 MiB so the test focuses purely on the client
	// side behavior, not the server-side recv cap.
	srv := grpc.NewServer(grpc.MaxRecvMsgSize(16 << 20))
	healthpb.RegisterHealthServer(srv, health.NewServer())
	go func() { _ = srv.Serve(lis) }()
	defer srv.Stop()

	conn, err := grpc.NewClient(lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		// Mirror mtls.GetGrpcOptions: only recv size set, no send size.
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(4<<30)),
	)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// HealthCheckRequest carries an arbitrary string field; pad it to 8 MiB
	// so the marshaled message is well past the alleged 4 MiB send cap.
	bigPayload := make([]byte, payloadSize)
	for i := range bigPayload {
		bigPayload[i] = 'A'
	}
	req := &healthpb.HealthCheckRequest{Service: string(bigPayload)}

	client := healthpb.NewHealthClient(conn)
	_, err = client.Check(context.Background(), req)
	// We don't care whether the health service knows the empty service name —
	// only whether the request marshals and ships. A "ResourceExhausted: max
	// message size" failure on send would prove the 4 MiB cap claim.
	if err != nil {
		// Status NotFound from health server is expected for unknown service;
		// the only failure we want to flag is the size cap.
		if got := err.Error(); contains(got, "max") && contains(got, "size") {
			t.Fatalf("client default send was capped: %v", err)
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
