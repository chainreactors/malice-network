package middleware

import (
	"context"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/internal/configs"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"os"
	"path"
)

type contextKey int

const (
	Transport contextKey = iota
	Operator
	CertFile = "localhost_root_crt.pem"
	KeyFile  = "localhost_root_key.pem"
)

func chainUnaryInterceptors(interceptors ...grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		buildChain := func(current grpc.UnaryHandler, interceptors []grpc.UnaryServerInterceptor) grpc.UnaryHandler {
			for i := len(interceptors) - 1; i >= 0; i-- {
				current = func(current grpc.UnaryHandler, interceptor grpc.UnaryServerInterceptor) grpc.UnaryHandler {
					return func(ctx context.Context, req interface{}) (interface{}, error) {
						return interceptor(ctx, req, info, current)
					}
				}(current, interceptors[i])
			}
			return current
		}
		chain := buildChain(handler, interceptors)
		return chain(ctx, req)
	}
}

func AuthMiddleware(log *logs.Logger) []grpc.ServerOption {
	var interceptors []grpc.UnaryServerInterceptor
	interceptors = append(interceptors, logMiddleware(log))
	return append(authMiddleware(context.Background()), grpc.UnaryInterceptor(chainUnaryInterceptors(interceptors...)))
}

func CommonMiddleware(log *logs.Logger) []grpc.ServerOption {
	var interceptors []grpc.UnaryServerInterceptor
	interceptors = append(interceptors, logMiddleware(log))
	return []grpc.ServerOption{
		grpc.UnaryInterceptor(chainUnaryInterceptors(interceptors...)),
	}
}

// logMiddleware - Log middleware
func logMiddleware(log *logs.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		log.Infof("Method %s is called with request: %+v", info.FullMethod, req)
		resp, err := handler(ctx, req)
		log.Infof("Method %s returns response: %+v", info.FullMethod, resp)
		return resp, err
	}
}

// authMiddleware - Auth middleware
func authMiddleware(ctx context.Context) []grpc.ServerOption {
	return []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			grpc_auth.UnaryServerInterceptor(hasPermissions),
		),
		grpc.ChainStreamInterceptor(
			grpc_auth.StreamServerInterceptor(hasPermissions),
		),
	}
}

// hasPermissions - Check if client has permissions`
func hasPermissions(ctx context.Context) (context.Context, error) {
	certPath := path.Join(configs.CertsPath, CertFile)
	keyPath := path.Join(configs.CertsPath, KeyFile)
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		logs.Log.Errorf("Failed to load cert %v", err)
		return nil, status.Error(codes.Unauthenticated, "Authentication failed")
	}
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		logs.Log.Errorf("Failed to load key %v", err)
		return nil, status.Error(codes.Unauthenticated, "Authentication failed")
	}
	newCtx := context.WithValue(ctx, Transport, "mtls")
	// TODO - Add operator check
	newCtx = context.WithValue(newCtx, Operator, "test")
	return newCtx, nil
}
