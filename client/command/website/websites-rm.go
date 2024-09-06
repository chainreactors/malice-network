package website

import (
	"context"
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/proto/listener/lispb"
)

func websitesRmCmd(c *grumble.Context, con *repl.Console) {
	name := c.Flags.String("name")
	if name == "" {
		repl.Log.Errorf("Must specify a website name via --name, see --help")
		return
	}

	_, err := con.Rpc.WebsiteRemove(context.Background(), &lispb.Website{
		Name: name,
	})
	if err != nil {
		repl.Log.Errorf("%s", err)
		return
	}
}
