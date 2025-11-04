package rpc

import (
	"context"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/saas"
)

func (rpc *Server) GetLicenseInfo(ctx context.Context, req *clientpb.Empty) (*clientpb.LicenseInfo, error) {
	saasConfig := configs.GetSaasConfig()

	if !saasConfig.Enable {
		return nil, types.ErrSaasUnable
	}
	if saasConfig.Token == "" {
		return nil, types.ErrLicenseTokenNotFound
	}

	client := saas.GetSaasClient()
	info, _, err := client.GetLicenseInfo()
	if err != nil {
		return nil, types.ErrSaasUnable
	}
	return info, nil
}
