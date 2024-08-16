package listener

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
)

func Commands(con *console.Console) []*grumble.Command {
	websiteCmd := &grumble.Command{
		Name: "website",
		Help: "Listener website ctrl manager",
		Args: func(f *grumble.Args) {
			f.String("listener_id", "listener id")
		},
		Run: func(ctx *grumble.Context) error {
			listWebsitesCmd(ctx, con)
			return nil
		},
		HelpGroup: consts.ListenerGroup,
	}

	websiteCmd.AddCommand(&grumble.Command{
		Name: "start",
		Help: "Start a website pipeline",
		Flags: func(f *grumble.Flags) {
			f.StringL("web-path", "", "path to the website")
			f.String("", "content-type", "", "content type")
			f.IntL("port", 0, "website pipeline port")
			f.StringL("name", "", "website name")
			f.StringL("content-path", "", "path to the content file")
			f.StringL("listener_id", "", "listener id")
			f.StringL("cert_path", "", "website tls cert path")
			f.StringL("key_path", "", "website tls key path")
			f.Bool("", "recursive", false, "add content recursively")
		},
		Run: func(ctx *grumble.Context) error {
			startWebsiteCmd(ctx, con)
			return nil
		},
	})

	websiteCmd.AddCommand(&grumble.Command{
		Name: "stop",
		Help: "Stop a website pipeline",
		Args: func(a *grumble.Args) {
			a.String("name", "website pipeline name")
			a.String("listener_id", "listener id")
		},
		Run: func(ctx *grumble.Context) error {
			stopWebsitePipelineCmd(ctx, con)
			return nil
		},
	})

	tcpCmd := &grumble.Command{
		Name: "tcp",
		Help: "Listener tcp pipeline ctrl manager",
		Args: func(a *grumble.Args) {
			a.String("listener_id", "listener id")
		},
		Run: func(ctx *grumble.Context) error {
			listTcpPipelines(ctx, con)
			return nil
		},
		HelpGroup: consts.ListenerGroup,
	}

	tcpCmd.AddCommand(&grumble.Command{
		Name: "start",
		Help: "Start a TCP pipeline",
		Flags: func(f *grumble.Flags) {
			f.StringL("host", "", "tcp pipeline host")
			f.IntL("port", 0, "tcp pipeline port")
			f.StringL("name", "", "tcp pipeline name")
			f.StringL("listener_id", "", "listener id")
			f.StringL("cert_path", "", "tcp pipeline tls cert path")
			f.StringL("key_path", "", "tcp pipeline tls key path")
		},
		Run: func(ctx *grumble.Context) error {
			startTcpPipelineCmd(ctx, con)
			return nil
		},
	})

	tcpCmd.AddCommand(&grumble.Command{
		Name: "stop",
		Help: "Stop a TCP pipeline",
		Args: func(a *grumble.Args) {
			a.String("name", "tcp pipeline name")
			a.String("listener_id", "listener id")
		},
		Run: func(ctx *grumble.Context) error {
			stopTcpPipelineCmd(ctx, con)
			return nil
		},
	})
	return []*grumble.Command{websiteCmd, tcpCmd}
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
