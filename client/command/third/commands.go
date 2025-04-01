package third

import (
	"github.com/carapace-sh/carapace"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *repl.Console) []*cobra.Command {
	curlCmd := &cobra.Command{
		Use:   consts.ModuleCurl + " [url]",
		Short: "Send HTTP request",
		Long:  "Send HTTP request to specified URL",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return CurlCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModuleCurl,
		},
		Example: `~~~
curl http://example.com

curl -X POST -d "data" http://example.com

curl -H "Host: example.com" -H "User-Agent: custom" http://example.com
~~~`,
	}

	common.BindArgCompletions(curlCmd, nil,
		carapace.ActionValues().Usage("target url"))

	common.BindFlag(curlCmd, func(f *pflag.FlagSet) {
		f.StringP("method", "X", "GET", "HTTP method")
		f.IntP("timeout", "t", 30, "request timeout in seconds")
		f.StringP("body", "d", "", "request body")
		f.StringArrayP("header", "H", nil, "HTTP header (can be used multiple times)")
	})

	return []*cobra.Command{curlCmd}
}

func Register(con *repl.Console) {
	RegisterCurlFunc(con)
}
