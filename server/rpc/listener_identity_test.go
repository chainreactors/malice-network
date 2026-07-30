package rpc

import (
	"context"
	"testing"

	"github.com/chainreactors/IoM-go/mtls"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const testRegisterListenerMethod = "/listenerrpc.ListenerRPC/RegisterListener"

func seedListenerIdentity(t *testing.T, name, fingerprint string) *PeerIdentity {
	t.Helper()
	if err := db.CreateOperator(&models.Operator{
		Name:        name,
		Type:        mtls.Listener,
		Role:        models.RoleListener,
		Fingerprint: fingerprint,
	}); err != nil {
		t.Fatalf("CreateOperator failed: %v", err)
	}
	opCache.Invalidate()
	t.Cleanup(opCache.Invalidate)
	return &PeerIdentity{Fingerprint: fingerprint}
}

func listenerIncomingContext(listenerID string) context.Context {
	return metadata.NewIncomingContext(context.Background(), metadata.Pairs("listener_id", listenerID))
}

func TestAuthorizeListenerRPCIdentityAcceptsMatchingName(t *testing.T) {
	_ = newRPCTestEnv(t)
	identity := seedListenerIdentity(t, "listener-2", "listener-2-fingerprint")

	err := authorizeListenerRPCIdentity(
		listenerIncomingContext("listener-2"),
		identity,
		testRegisterListenerMethod,
		&clientpb.RegisterListener{Name: "listener-2"},
	)
	if err != nil {
		t.Fatalf("authorizeListenerRPCIdentity failed: %v", err)
	}
}

func TestAuthorizeListenerRPCIdentityRejectsMetadataMismatch(t *testing.T) {
	_ = newRPCTestEnv(t)
	identity := seedListenerIdentity(t, "listener-2", "listener-2-fingerprint")

	err := authorizeListenerRPCIdentity(
		listenerIncomingContext("listener"),
		identity,
		testRegisterListenerMethod,
		&clientpb.RegisterListener{Name: "listener"},
	)
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("authorization error = %v, want PermissionDenied", err)
	}
}

func TestAuthorizeListenerRPCIdentityRejectsRequestMismatch(t *testing.T) {
	_ = newRPCTestEnv(t)
	identity := seedListenerIdentity(t, "listener-2", "listener-2-fingerprint")

	err := authorizeListenerRPCIdentity(
		listenerIncomingContext("listener-2"),
		identity,
		"/listenerrpc.ListenerRPC/RegisterPipeline",
		&clientpb.Pipeline{Name: "tcp", ListenerId: "listener"},
	)
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("authorization error = %v, want PermissionDenied", err)
	}
}

func TestAuthorizeListenerRPCIdentityRejectsMissingMetadata(t *testing.T) {
	_ = newRPCTestEnv(t)
	identity := seedListenerIdentity(t, "listener-2", "listener-2-fingerprint")

	err := authorizeListenerRPCIdentity(
		context.Background(),
		identity,
		testRegisterListenerMethod,
		&clientpb.RegisterListener{Name: "listener-2"},
	)
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("authorization error = %v, want PermissionDenied", err)
	}
}

func TestIdentityServerStreamRejectsMismatchedListenerMessage(t *testing.T) {
	_ = newRPCTestEnv(t)
	identity := seedListenerIdentity(t, "listener-2", "listener-2-fingerprint")
	base := &testRPCServerStream{
		ctx: listenerIncomingContext("listener-2"),
		recvMsg: func(message interface{}) error {
			response := message.(*clientpb.SpiteResponse)
			response.ListenerId = "listener"
			return nil
		},
	}
	stream := &identityServerStream{
		ServerStream: base,
		identity:     identity,
		method:       "/listenerrpc.ListenerRPC/SpiteStream",
	}

	err := stream.RecvMsg(&clientpb.SpiteResponse{})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("RecvMsg error = %v, want PermissionDenied", err)
	}
}
