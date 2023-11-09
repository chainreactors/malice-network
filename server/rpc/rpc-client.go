package rpc

import (
	"context"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/core"
)

func (rpc *Server) GetClients(ctx context.Context, req *clientpb.Empty) (*clientpb.Clients, error) {
	clients := &clientpb.Clients{}
	for _, client := range core.Clients.ActiveClients() {
		clients.Clients = append(clients.Clients, client.ToProtobuf())
	}
	return clients, nil
}

func (rpc *Server) LoginClient(ctx context.Context, req *clientpb.LoginReq) (*clientpb.LoginResp, error) {
	//host, port := req.Host, uint16(req.Port)
	//if host == "" || port == 0 {
	//	logs.Log.Error("LoginClient: host or user is empty")
	//	return &clientpb.LoginResp{
	//		Success: false,
	//	}, nil
	//}
	//_, _, err := web.StartMtlsClientListener(host, port)
	//if err != nil {
	//	logs.Log.Errorf("LoginClient: %s", err.Error())
	//	return &clientpb.LoginResp{
	//		Success: false,
	//	}, nil
	//}
	//return &clientpb.LoginResp{
	//	Success: true,
	//}, nil
	return nil, nil
}
