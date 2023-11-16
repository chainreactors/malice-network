package middleware

import (
	"context"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/configs"
	"google.golang.org/grpc"
	"os"
	"path/filepath"
)

const (
	configDir = ".config/certs"
	certFile  = "local_root.cert"
	keyFile   = "local_root.key"
)

// InitMiddleware - Init middleware
func InitMiddleware(rootAuth bool, logName string) []grpc.ServerOption {
	var opts []grpc.ServerOption
	if rootAuth {
		opts = append(opts, grpc.UnaryInterceptor(authMiddleware))
	}
	opts = append(opts, grpc.UnaryInterceptor(logMiddleware(logName)))
	return opts
}

// logMiddleware - Log middleware
func logMiddleware(logName string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		grpcLogger := configs.SetLogFilePath(logName, logs.Info)
		grpcLogger.Infof("Method %s is called with request: %+v", info.FullMethod, req)
		resp, err := handler(ctx, req)
		grpcLogger.Infof("Method %s returns response: %+v", info.FullMethod, resp)
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
	certPath := filepath.Join(configDir, certFile)
	keyPath := filepath.Join(configDir, keyFile)

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
