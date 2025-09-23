package rpc

import (
	"context"
	"errors"
	"github.com/chainreactors/logs"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"net"
	"reflect"
	"strings"
)

type contextKey int

const (
	Transport contextKey = iota
	Operator
	rootName = "Root"
)

func buildOptions(option []grpc.ServerOption, interceptors ...grpc.UnaryServerInterceptor) []grpc.ServerOption {
	option = append(option, grpc.ChainUnaryInterceptor(interceptors...))
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

// authInterceptor - Auth middleware with UPN authentication
func authInterceptor(log *logs.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		client, ok := peer.FromContext(ctx)
		if !ok {
			log.Errorf("[auth] failed to get peers information from context")
			return ctx, errors.New("failed to get peers information from context")
		}
		if client.AuthInfo == nil {
			log.Errorf("[auth] auth info not found")
			return ctx, errors.New("auth info not found")
		}

		tlsInfo, ok := client.AuthInfo.(credentials.TLSInfo)
		if !ok {
			log.Errorf("[auth] TLS info not found")
			return ctx, errors.New("TLS info not found")
		}

		// Extract UPN from client certificate
		upn := ""
		if len(tlsInfo.State.PeerCertificates) > 0 {
			cert := tlsInfo.State.PeerCertificates[0]
			// UPN is stored in EmailAddresses field
			if len(cert.EmailAddresses) > 0 {
				upn = cert.EmailAddresses[0]
			}
		}

		// If no UPN found, return error directly
		if upn == "" {
			log.Errorf("[auth] UPN not found in client certificate")
			return ctx, errors.New("UPN not found in client certificate")
		}

		log.Debugf("[auth] authenticating user with UPN: %s", upn)

		// Store UPN in context for later use
		ctx = context.WithValue(ctx, Operator, upn)

		// Validate UPN and permissions based on method
		if err := validateUPNPermissions(upn, info.FullMethod, log); err != nil {
			log.Errorf("[auth] UPN validation failed for %s: %v", upn, err)
			return ctx, err
		}

		// For root operations, also check IP restrictions
		if isRootOperation(info.FullMethod) {
			host, _, err := net.SplitHostPort(client.Addr.String())
			if err != nil {
				log.Errorf("[auth] invalid remote address format")
				return ctx, errors.New("invalid remote address format")
			}

			parsed := net.ParseIP(host)
			if !(parsed != nil && parsed.IsLoopback()) {
				log.Errorf("[auth] root operations only allowed from localhost, got: %s", host)
				return ctx, errors.New("root operations only allowed from localhost")
			}
		}

		return handler(ctx, req)
	}
}

// validateUPNPermissions validates if the UPN has permission to call the specific method
func validateUPNPermissions(upn, method string, log *logs.Logger) error {
	// Extract domain and username from UPN
	parts := strings.Split(upn, "@")
	if len(parts) != 2 {
		return errors.New("invalid UPN format")
	}

	username := parts[0]
	domain := parts[1]

	// Validate that both username and domain are not empty
	if username == "" || domain == "" {
		return errors.New("invalid UPN - username or domain is empty")
	}

	// Only allow chainreactors.local domain
	if !strings.EqualFold(domain, "chainreactors.local") {
		return errors.New("invalid domain - only chainreactors.local allowed")
	}

	log.Debugf("[auth] validating permissions for user: %s, domain: %s, method: %s", username, domain, method)
	
	// Check username and method matching
	switch strings.ToLower(username) {
	case "root", "client":
		return nil
	case "listener":
		if strings.HasPrefix(method, "/listenerrpc.") {
			return nil
		}
		return errors.New("listener user can only access listenerrpc methods")
	default:
		return errors.New("invalid username - only root, client, listener allowed")
	}
}

// isRootOperation checks if the method is a root operation requiring localhost access
func isRootOperation(method string) bool {
	return strings.Contains(method, ".Root")
}
