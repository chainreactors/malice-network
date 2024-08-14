package listener

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
)

func Commands(con *console.Console) []*grumble.Command {
	lisCmd := &grumble.Command{
		Name: "listener",
		Help: "listener manager",
		Run: func(c *grumble.Context) error {
			ListenerCmd(c, con)
			return nil
		},
	}
	lisCmd.AddCommand(TcpCmd(con))
	lisCmd.AddCommand(WebsiteCmd(con))
	return []*grumble.Command{lisCmd}
}

//	tcpCmd := &grumble.Command{
//		Name: "tcp",
//		Help: "Start a TCP pipeline",
//		Flags: func(f *grumble.Flags) {
//			f.String("l", "lhost", "0.0.0.0", "listen host")
//			f.Int("p", "lport", 0, "listen port")
//		},
//		Run: func(ctx *grumble.Context) error {
//			jobs.TcpPipelineCmd(ctx, con)
//			return nil
//		},
//	}
