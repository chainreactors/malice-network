package rpc

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
	"reflect"
)

type contextKey int

const (
	Transport contextKey = iota
	Operator
	CertFile = "localhost_root_crt.pem"
	KeyFile  = "localhost_root_key.pem"
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
		if err == nil && sess.Logger() != nil {
			sess.Logger().Consolef("[request] %s %s \n", info.FullMethod, reflect.TypeOf(req))
			sess.Logger().Debugf("%+v", req)
			resp, err := handler(ctx, req)
			sess.Logger().Consolef("[response] %s %s \n", info.FullMethod, reflect.TypeOf(resp))
			sess.Logger().Debugf("%+v", resp)
			return resp, err
		}
		return handler(ctx, req)
	}
}

// authInterceptor - Auth middleware
func authInterceptor() []grpc.ServerOption {
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
		logs.Log.Errorf("Failed  to load key %v", err)
		return nil, status.Error(codes.Unauthenticated, "Authentication failed")
	}
	ctx = context.WithValue(ctx, Transport, "mtls")
	// TODO - Add operator check
	ctx = context.WithValue(ctx, Operator, "test")
	return ctx, nil
}
