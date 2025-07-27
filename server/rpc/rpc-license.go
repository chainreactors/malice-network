package rpc

import (
	"context"
	"github.com/chainreactors/malice-network/helper/errs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/saas"
)

func (rpc *Server) GetLicenseInfo(ctx context.Context, req *clientpb.Empty) (*clientpb.LicenseInfo, error) {
	saasConfig := configs.GetSaasConfig()

	if !saasConfig.Enable {
		return nil, errs.ErrSaasUnable
	}
	if saasConfig.Token == "" {
		return nil, errs.ErrLicenseTokenNotFound
	}

	client := saas.NewSaasClient()
	info, _, err := client.GetLicenseInfo()
	if err != nil {
		return nil, errs.ErrSaasUnable
	}
	return info, nil
}
