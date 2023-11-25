package rpc

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	certs2 "github.com/chainreactors/malice-network/server/internal/certs"
	"github.com/chainreactors/malice-network/server/internal/generate"
	"golang.org/x/net/context"
)

var rpcRegLog = logs.Log

func (rpc *Server) RegisterCA(ctx context.Context, req *clientpb.RegisterReq) (*clientpb.RegisterResp, error) {
	cert, key, err := generate.GenerateClientCA(req.Host, req.User)
	if err != nil {
		rpcRegLog.Errorf("Failed to generate client %s CA: %s", req.Host, err)
		return nil, err
	}
	ca, _, err := certs2.GetCertificateAuthority(certs2.SERVERCA)
	if err != nil {
		rpcRegLog.Errorf("Failed to load CA %s", err)
		return nil, err
	}
	res := &clientpb.RegisterResp{
		Certs:      cert,
		PrivateKey: key,
		CA:         ca.Raw,
	}
	return res, nil
}
