package middleware

import (
	"context"
	"github.com/chainreactors/logs"
	"google.golang.org/grpc"
	"os"
	"path"
)

const (
	configDir = ".config/certs"
	certFile  = "local_root.cert"
	keyFile   = "local_root.key"
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
	interceptors = append(interceptors, authMiddleware)
	interceptors = append(interceptors, logMiddleware(log))
	return []grpc.ServerOption{
		grpc.UnaryInterceptor(chainUnaryInterceptors(interceptors...)),
	}
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
func authMiddleware(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (interface{}, error) {
	if !hasPermissions(ctx) {
		logs.Log.Errorf("Client not authorized")
		return nil, nil
	}
	return handler(ctx, req)
}

// hasPermissions - Check if client has permissions`
func hasPermissions(ctx context.Context) bool {
	certPath := path.Join(configDir, certFile)
	keyPath := path.Join(configDir, keyFile)

	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		logs.Log.Errorf("Failed to load cert %v", err)
		return false
	}
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		logs.Log.Errorf("Failed to load key %v", err)
		return false
	}
	return true
}
