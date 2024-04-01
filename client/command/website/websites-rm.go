package website

import (
	"context"
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
)

func websitesRmCmd(c *grumble.Context, con *console.Console) {
	name := c.Flags.String("name")
	if name == "" {
		console.Log.Errorf("Must specify a website name via --name, see --help")
		return
	}

	_, err := con.Rpc.WebsiteRemove(context.Background(), &clientpb.Website{
		Name: name,
	})
	if err != nil {
		console.Log.Errorf("%s", err)
		return
	}
}
