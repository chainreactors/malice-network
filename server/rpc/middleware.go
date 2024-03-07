package rpc

import (
	"context"
	"errors"
	"github.com/chainreactors/logs"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"reflect"
	"strings"
)

type contextKey int

const (
	Transport contextKey = iota
	Operator
	rootName = "Root"
	rootAddr = "127.0.0.1"
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
		caType := client.AuthInfo.(credentials.TLSInfo).State.ServerName
		if len(caType) == 0 {
			log.Errorf("[auth] certificate type not found")
			return ctx, errors.New("certificate type not found")
		}
		if caType == rootName {
			if !strings.Contains(info.FullMethod, "."+caType) {
				log.Errorf("[auth] certificate type does not match method")
				return ctx, errors.New("certificate type does not match method")
			}
			parts := strings.Split(client.Addr.String(), ":")
			if len(parts) != 2 {
				log.Errorf("[auth] invalid remote address format")
				return ctx, errors.New("invalid remote address format")
			}
			if parts[0] != rootAddr {
				log.Errorf("[auth] invalid remote address")
				return ctx, errors.New("invalid remote address")
			}
		} else {
			if !strings.HasPrefix(info.FullMethod, "/"+caType) {
				log.Errorf("[auth] certificate type does not match method")
				return ctx, errors.New("certificate type does not match method")
			}
		}
		return handler(ctx, req)
	}
}
