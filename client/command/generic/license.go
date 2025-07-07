package generic

import (
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
)

func GetLicenseCmd(cmd *cobra.Command, con *repl.Console) error {
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
	tui.RenderKV(licenseMap)
}
