package generic

import (
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
)

func GetLicenseCmd(cmd *cobra.Command, con *core.Console) error {
	licenseInfo, err := con.Rpc.GetLicenseInfo(con.Context(), &clientpb.Empty{})
	if err != nil {
		return err
	}
	printLicense(licenseInfo)
	return nil
}

func printLicense(license *clientpb.LicenseInfo) {
	var expireAtDisplay string
	if license.Type == consts.LicenseCommunity {
		expireAtDisplay = "Never expires"
	} else {
		expireAtDisplay = license.ExpireAt
	}

	licenseMap := map[string]interface{}{
		"Type":          license.Type,
		"ExpireAt":      expireAtDisplay,
		"ProBuildCount": license.BuildCount,
		"MaxBuilds":     license.MaxBuilds,
	}
	orderedKeys := []string{"Type", "ExpireAt", "ProBuildCount", "MaxBuilds"}
	tui.RenderKV(licenseMap, orderedKeys)
}
