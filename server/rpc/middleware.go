package rpc

import (
	"context"
	"errors"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/utils/mtls"
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
			host, _, err := net.SplitHostPort(client.Addr.String())
			if err != nil {
				log.Errorf("[auth] invalid remote address format")
				return ctx, errors.New("invalid remote address format")
			}

			parsed := net.ParseIP(host)
			if !(parsed != nil && parsed.IsLoopback()) {
				log.Errorf("[auth] invalid remote address")
				return ctx, errors.New("invalid remote address")
			}
		} else {
			if !strings.HasPrefix(info.FullMethod, "/"+caType) && caType == mtls.Listener {
				log.Errorf("[auth] certificate type does not match method")
				return ctx, errors.New("certificate type does not match method")
			}
		}
		return handler(ctx, req)
	}
}
