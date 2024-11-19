package rpc

import (
	"context"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/client/rootpb"
	"github.com/chainreactors/malice-network/helper/utils/mtls"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"gopkg.in/yaml.v3"
)

func (rpc *Server) AddClient(ctx context.Context, req *rootpb.Operator) (*rootpb.Response, error) {
	cfg := configs.GetServerConfig()
	clientConf, err := certutils.GenerateClientCert(cfg.IP, req.Args[0], int(cfg.GRPCPort))
	if err != nil {
		return &rootpb.Response{
			Status: 1,
			Error:  err.Error(),
		}, err
	}
	err = db.CreateOperator(req.Args[0], mtls.Client, getRemoteAddr(ctx))
	if err != nil {
		return nil, err
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
	err := certutils.RemoveCertificate(certutils.OperatorCA, certutils.RSAKey, req.Args[0])
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
	operators, err := db.ListClients()
	if err != nil {
		return nil, err
	}
	var clients []*clientpb.Client
	for _, op := range operators {
		client := &clientpb.Client{
			Name: op.Name,
		}
		clients = append(clients, client)
	}
	return &clientpb.Clients{
		Clients: clients,
	}, nil
}

func (rpc *Server) AddListener(ctx context.Context, req *rootpb.Operator) (*rootpb.Response, error) {
	cfg := configs.GetServerConfig()
	clientConf, err := certutils.GenerateListenerCert(cfg.IP, req.Args[0], int(cfg.GRPCPort))
	if err != nil {
		return &rootpb.Response{
			Status: 1,
			Error:  err.Error(),
		}, err
	}
	err = db.CreateOperator(req.Args[0], mtls.Listener, getRemoteAddr(ctx))
	if err != nil {
		return nil, err
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

func (rpc *Server) RemoveListener(ctx context.Context, req *rootpb.Operator) (*rootpb.Response, error) {
	err := certutils.RemoveCertificate(certutils.ListenerCA, certutils.RSAKey, req.Args[0])
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

func (rpc *Server) ListListeners(ctx context.Context, req *rootpb.Operator) (*clientpb.Listeners, error) {
	dbListeners, err := db.ListListeners()
	if err != nil {
		return nil, err
	}
	listeners := &clientpb.Listeners{}
	for _, listener := range dbListeners {
		listeners.Listeners = append(listeners.Listeners, &clientpb.Listener{
			Id:   listener.Name,
			Addr: listener.Remote,
		})
	}

	return listeners, nil
}
