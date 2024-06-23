package explorer

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
)

func Commands(con *console.Console) []*grumble.Command {
	return []*grumble.Command{
		{
			Name: "explorer",
			Help: "file explorer",
			Run: func(ctx *grumble.Context) error {
				explorerCmd(ctx, con)
				return nil
			},
		},
	}
}
