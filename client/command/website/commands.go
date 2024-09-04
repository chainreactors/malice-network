package website

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
)

const (
	fileSampleSize  = 512
	defaultMimeType = "application/octet-stream"
)

func Commands(con *console.Console) []*grumble.Command {
	webCmd := &grumble.Command{
		Name: "website",
		Help: "website manager",
		Run: func(c *grumble.Context) error {
			// todo list listeners
			return nil
		},
		HelpGroup: consts.ListenerGroup,
	}

	// add-content
	webCmd.AddCommand(&grumble.Command{
		Name: "add-content",
		Help: "Add content to a website",
		Args: func(a *grumble.Args) {
			a.String("content-path", "path to the content file")
		},
		Flags: func(f *grumble.Flags) {
			f.StringL("web-path", "", "path to the website")
			f.String("n", "name", "", "name of the website")
			f.String("", "content-type", "", "content type")
			f.Bool("", "recursive", false, "add content recursively")
		},
		Run: func(c *grumble.Context) error {
			websiteAddCmd(c, con)
			return nil
		},
		Completer: func(prefix string, args []string) []string {
			return common.LocalPathCompleter(prefix, args, con)
		},
	},
	)

	// rm-content
	webCmd.AddCommand(&grumble.Command{
		Name: "rm-content",
		Help: "Remove specific content from a website",
		Flags: func(f *grumble.Flags) {
			f.String("n", "name", "", "name of the website")
			f.String("", "web-path", "", "path to the website")
			f.Bool("r", "recursive", false, "remove content recursively")
		},
		Run: func(c *grumble.Context) error {
			webRmContentCmd(c, con)
			return nil
		},
	})

	// rm website
	webCmd.AddCommand(&grumble.Command{
		Name: "rm",
		Help: "Remove a website",
		Flags: func(f *grumble.Flags) {
			f.String("n", "name", "", "name of the website")
		},
		Run: func(c *grumble.Context) error {
			websitesRmCmd(c, con)
			return nil
		},
	})

	// update-content
	webCmd.AddCommand(&grumble.Command{
		Name: "update-content",
		Help: "Update content of a website",
		Flags: func(f *grumble.Flags) {
			f.String("n", "name", "", "name of the website")
			f.String("", "web-path", "", "path to the website")
			f.String("", "content-type", "", "content type")
		},
		Run: func(c *grumble.Context) error {
			websiteUpdateContentCmd(c, con)
			return nil
		},
	})

	// list content
	webCmd.AddCommand(&grumble.Command{
		Name: "list-contents",
		Help: "List the contents of a website",
		Flags: func(f *grumble.Flags) {
			f.String("n", "name", "website name", "name of the website")
		},
		Run: func(c *grumble.Context) error {
			listWebsitesCmd(c, con)
			return nil
		},
	})

	return []*grumble.Command{webCmd}
}
