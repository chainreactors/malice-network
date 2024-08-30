package rpc

import (
	"context"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/client/rootpb"
	"github.com/chainreactors/malice-network/server/internal/certs"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"gopkg.in/yaml.v3"
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
	//cert := models.Certificate{}
	//err := dbSession.Where(&models.Certificate{
	//	CommonName: req.Name,
	//	CAType:     certs.OperatorCA,
	//}).First(&cert).Error
	//if err != nil {
	//	if errors.Is(err, db.ErrRecordNotFound) {
	//		return &clientpb.LoginResp{
	//			Success: false,
	//		}, errors.New("certificate not found")
	//	}
	//	return &clientpb.LoginResp{
	//		Success: false,
	//	}, err
	//}

	dbSession.Where(&models.Operator{Name: req.Name}).Find(&operator)
	if len(operator) != 0 {
		return &clientpb.LoginResp{
			Success: true,
		}, nil
	}
	err := dbSession.Create(&models.Operator{
		Name: req.Name,
	}).Error

	core.Clients.Add(core.NewClient(req.Name))
	if err != nil {
		return &clientpb.LoginResp{
			Success: false,
		}, err
	}
	return &clientpb.LoginResp{
		Success: true,
	}, nil
}

func (rpc *Server) AddClient(ctx context.Context, req *rootpb.Operator) (*rootpb.Response, error) {
	cfg := configs.GetServerConfig()
	clientConf, err := certs.GenerateClientCert(cfg.GRPCHost, req.Args[0], int(cfg.GRPCPort))
	if err != nil {
		return &rootpb.Response{
			Status: 1,
			Error:  err.Error(),
		}, err
	}
	data, err := yaml.Marshal(clientConf)
	if err != nil {
		return &rootpb.Response{
			Status: 1,
			Error:  err.Error(),
		}, err
	}
	return &rootpb.Response{
		Status:   0,
		Response: string(data),
	}, nil
}

func (rpc *Server) RemoveClient(ctx context.Context, req *rootpb.Operator) (*rootpb.Response, error) {
	err := certs.RemoveCertificate(certs.OperatorCA, certs.RSAKey, req.Args[0])
	if err != nil {
		return &rootpb.Response{
			Status: 1,
			Error:  err.Error(),
		}, err
	}
	return &rootpb.Response{
		Status:   0,
		Response: "",
	}, nil
}

func (rpc *Server) ListClients(ctx context.Context, req *rootpb.Operator) (*clientpb.Clients, error) {
	clients, err := db.ListOperators()
	if err != nil {
		return nil, err
	}
	return clients, nil
}
