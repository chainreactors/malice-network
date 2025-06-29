package license

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/spf13/cobra"
)

func GetLicenseCmd(cmd *cobra.Command, con *repl.Console) error {
	licenseInfo, err := con.Rpc.GetLicenseInfo(con.Context(), &clientpb.Empty{})
	if err != nil {
		return err
	}
	if licenseInfo.Type == consts.LicenseCommunity {
		con.Log.Infof("licence type %v, enjoy for builing\n", consts.LicenseCommunity)
		return nil
	}
	buildCounts := licenseInfo.MaxBuilds - licenseInfo.BuildCount
	var countInfo string
	if buildCounts > 0 {
		countInfo = fmt.Sprintf("can build %d times", buildCounts)
	} else {
		countInfo = "has no build times"
	}
	con.Log.Infof("licence type %v, expire at %v, %v\n", licenseInfo.Type, licenseInfo.ExpireAt, countInfo)
	return nil
}
