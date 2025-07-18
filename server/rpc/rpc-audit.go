package rpc

import (
	"context"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/audit"
)

func (rpc *Server) GetAudit(ctx context.Context, req *clientpb.SessionRequest) (*clientpb.Audits, error) {
	result, err := audit.AuditTaskLog(req.SessionId)
	if err != nil {
		return nil, err
	}
	return result, nil
}
