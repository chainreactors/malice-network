package server

import (
	"context"
	"time"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/saas"
)

const saasLicenseStartupTimeout = 5 * time.Second

func startSaasLicenseRegistration() {
	config := configs.GetSaasConfig()
	if config == nil || !config.Enable {
		return
	}
	registerSaasLicenseInBackground(saasLicenseStartupTimeout, saas.RegisterLicenseContext)
}

func registerSaasLicenseInBackground(timeout time.Duration, register func(context.Context) error) <-chan error {
	return core.GoGuarded("saas-license-registration", func() error {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		return register(ctx)
	}, func(err error) {
		logs.Log.Warnf("register community license error %v", err)
	})
}
