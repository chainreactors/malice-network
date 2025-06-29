package license

import (
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/spf13/cobra"
)

func Commands(con *repl.Console) []*cobra.Command {
	licenseInfoCmd := &cobra.Command{
		Use:   consts.CommandLicense,
		Short: "show server license info",
		Long:  "show server license info",
		RunE: func(cmd *cobra.Command, args []string) error {
			return GetLicenseCmd(cmd, con)
		},
		Example: `~~~
license
~~~`,
	}

	return []*cobra.Command{licenseInfoCmd}
}
