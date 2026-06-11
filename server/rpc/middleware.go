package rpc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"strings"
	"sync"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// caFingerprintOnce lazily computes and caches the CA certificate fingerprint.
// The RootClient (used by CLI commands like "user add") authenticates with the
// CA certificate itself, whose fingerprint is not stored in the operators table.
// This allows the localhost admin bypass to recognize it.
var (
	caFingerprintOnce sync.Once
	caFingerprint     string
)

func getCAFingerprint() string {
	caFingerprintOnce.Do(func() {
		ca, _, err := certutils.GetCertificateAuthority()
		if err != nil {
			return
		}
		hash := sha256.Sum256(ca.Raw)
		caFingerprint = hex.EncodeToString(hash[:])
	})
	return caFingerprint
}

type contextKey int

const (
	Transport contextKey = iota
)

func buildOptions(option []grpc.ServerOption,
	unaryInterceptors []grpc.UnaryServerInterceptor,
	streamInterceptors []grpc.StreamServerInterceptor,
) []grpc.ServerOption {
	option = append(option, grpc.ChainUnaryInterceptor(unaryInterceptors...))
	if len(streamInterceptors) > 0 {
		option = append(option, grpc.ChainStreamInterceptor(streamInterceptors...))
	}
	return option
}

// logInterceptor - Log middleware
func logInterceptor(log *logs.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if sid, err := getSessionID(ctx); err == nil {
			log.Infof("[implant] %s call %s with %s: %s", getClientName(ctx), info.FullMethod, sid, reflect.TypeOf(req))
			resp, err := handler(ctx, req)
			log.Infof("[implant] %s back %s: %s", sid, info.FullMethod, reflect.TypeOf(resp))
			return resp, err
		} else {
			log.Infof("[malice] %s call %s with %s", getClientName(ctx), info.FullMethod, reflect.TypeOf(req))
			resp, err := handler(ctx, req)
			log.Infof("[malice] %s back %s: %s", getClientName(ctx), info.FullMethod, reflect.TypeOf(resp))
			return resp, err
		}
	}
}

func auditInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		sess, err := getSession(ctx)
		if err == nil && sess.RpcLogger() != nil {
			sess.RpcLogger().Consolef("[request] %s %s \n", info.FullMethod, reflect.TypeOf(req))
			sess.RpcLogger().Debugf("%+v", req)
			resp, err := handler(ctx, req)
			sess.RpcLogger().Consolef("[response] %s %s \n", info.FullMethod, reflect.TypeOf(resp))
			sess.RpcLogger().Debugf("%+v", resp)
			return resp, err
		}
		return handler(ctx, req)
	}
}

// authInterceptor - Auth middleware for unary RPCs
func authInterceptor(log *logs.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		identity, err := extractPeerIdentity(ctx)
		if err != nil {
			log.Errorf("[auth] identity extraction failed: %v", err)
			return nil, err
		}

		ctx = contextWithIdentity(ctx, identity)

		log.Debugf("[auth] peer: cn=%s fp=%s remote=%s",
			identity.CommonName, identity.Fingerprint[:16], identity.RemoteAddr)

		// Localhost admin bypass: registered admin operator OR the CA certificate itself
		// (the CLI RootClient uses the CA cert directly for local admin commands)
		if identity.IsLoopback {
			if op, ok := opCache.LookupByFingerprint(identity.Fingerprint); ok && op.Role == "admin" {
				return handler(ctx, req)
			}
			if fp := getCAFingerprint(); fp != "" && identity.Fingerprint == fp {
				log.Debugf("[auth] localhost CA certificate bypass for %s", info.FullMethod)
				return handler(ctx, req)
			}
		}

		if err := authenticateByFingerprint(identity, info.FullMethod); err != nil {
			return nil, err
		}

		// Root operation localhost check
		if isRootOperation(info.FullMethod) && !identity.IsLoopback {
			return nil, status.Errorf(codes.PermissionDenied,
				"root operations only allowed from localhost, got: %s", identity.RemoteIP)
		}

		return handler(ctx, req)
	}
}

// authStreamInterceptor - Auth middleware for streaming RPCs
func authStreamInterceptor(log *logs.Logger) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()

		identity, err := extractPeerIdentity(ctx)
		if err != nil {
			log.Errorf("[auth-stream] identity extraction failed: %v", err)
			return err
		}

		log.Debugf("[auth-stream] peer: cn=%s fp=%s method=%s",
			identity.CommonName, identity.Fingerprint[:16], info.FullMethod)

		// Localhost admin bypass: registered admin operator OR the CA certificate itself
		if identity.IsLoopback {
			if op, ok := opCache.LookupByFingerprint(identity.Fingerprint); ok && op.Role == "admin" {
				wrapped := &identityServerStream{ServerStream: ss, identity: identity}
				return handler(srv, wrapped)
			}
			if fp := getCAFingerprint(); fp != "" && identity.Fingerprint == fp {
				log.Debugf("[auth-stream] localhost CA certificate bypass for %s", info.FullMethod)
				wrapped := &identityServerStream{ServerStream: ss, identity: identity}
				return handler(srv, wrapped)
			}
		}

		if err := authenticateByFingerprint(identity, info.FullMethod); err != nil {
			return err
		}

		wrapped := &identityServerStream{ServerStream: ss, identity: identity}
		return handler(srv, wrapped)
	}
}

// identityServerStream wraps grpc.ServerStream to override Context() with identity.
type identityServerStream struct {
	grpc.ServerStream
	identity *PeerIdentity
}

func (s *identityServerStream) Context() context.Context {
	ctx := s.ServerStream.Context()
	ctx = contextWithIdentity(ctx, s.identity)
	return ctx
}

// authenticateByFingerprint looks up the operator by cert fingerprint and checks permissions.
func authenticateByFingerprint(identity *PeerIdentity, method string) error {
	op, ok := opCache.LookupByFingerprint(identity.Fingerprint)
	if !ok {
		return status.Errorf(codes.Unauthenticated,
			"certificate fingerprint not registered: %s", identity.Fingerprint[:16])
	}
	if op.Revoked {
		return status.Errorf(codes.Unauthenticated,
			"operator %s has been revoked", op.Name)
	}
	return authorizeByRole(op.Role, method)
}

// authorizeByRole checks if a role is allowed to call the given method
// using the authz_rules table (via cache).
func authorizeByRole(role, method string) error {
	rules := ruleCache.GetRules(role)
	if len(rules) == 0 {
		return status.Errorf(codes.PermissionDenied,
			"no authorization rules found for role %q", role)
	}

	for _, rule := range rules {
		if matchMethod(rule.Method, method) {
			if rule.Allow {
				return nil
			}
			return status.Errorf(codes.PermissionDenied,
				"access denied by rule for role %q on %s", role, method)
		}
	}

	return status.Errorf(codes.PermissionDenied,
		"no matching rule for role %q on method %s", role, method)
}

// isRootOperation checks if the method is a root operation requiring localhost access.
func isRootOperation(method string) bool {
	return strings.Contains(method, ".Root")
}
