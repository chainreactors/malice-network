package rpc

import (
	"context"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
)

func (rpc *Server) GetClients(ctx context.Context, req *clientpb.Empty) (*clientpb.Clients, error) {
	clients := &clientpb.Clients{}
	for _, client := range core.Clients.ActiveClients() {
		clients.Clients = append(clients.Clients, client.ToProtobuf())
	}
	return clients, nil
}

func (rpc *Server) LoginClient(ctx context.Context, req *clientpb.LoginReq) (*clientpb.LoginResp, error) {
	host, port := req.Host, uint16(req.Port)
	var operator []*models.Operator
	if host == "" || port == 0 {
		logs.Log.Error("AddClient: host or user is empty")
		return &clientpb.LoginResp{
			Success: false,
		}, nil
	}
	dbSession := db.Session()
	cert := models.Certificate{}
	err := dbSession.Where(&models.Certificate{
		CommonName: fmt.Sprintf("%s.%s", req.Host, req.Name),
	}).First(&cert).Error
	if err != nil {
		return &clientpb.LoginResp{
			Success: false,
		}, err
	}

	dbSession.Where(&models.Operator{Token: req.Token}).Find(&operator)
	if len(operator) != 0 {
		return &clientpb.LoginResp{
			Success: true,
		}, nil
	}
	err = dbSession.Create(&models.Operator{
		Name:  req.Name,
		Token: req.Token,
	}).Error
	if err != nil {
		return &clientpb.LoginResp{
			Success: false,
		}, err
	}
	return &clientpb.LoginResp{
		Success: true,
	}, nil
}

func (rpc *Server) AddClient(ctx context.Context, req *clientpb.LoginReq) (*clientpb.LoginResp, error) {
	host, port := req.Host, uint16(req.Port)
	if host == "" || port == 0 {
		logs.Log.Error("AddClient: host or user is empty")
		return &clientpb.LoginResp{
			Success: false,
		}, nil
	}
	dbSession := db.Session()
	cert := models.Certificate{}
	err := dbSession.Where(&models.Certificate{
		CommonName: fmt.Sprintf("%s.%s", "client", req.Name),
	}).First(&cert).Error
	if err != nil {
		return &clientpb.LoginResp{
			Success: false,
		}, err
	}

	err = dbSession.Create(&models.Operator{
		Name: req.Name,
	}).Error
	if err != nil {
		return &clientpb.LoginResp{
			Success: false,
		}, err
	}
	return &clientpb.LoginResp{
		Success: true,
	}, nil
}
