package website

import (
	"context"
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"strings"
)

func webRmContentCmd(c *grumble.Context, con *console.Console) {
	name := c.Flags.String("name")
	webPath := c.Flags.String("web-path")
	recursive := c.Flags.Bool("recursive")
	if name == "" {
		console.Log.Errorf("Must specify a website name via --name, see --help")
		return
	}
	if webPath == "" {
		console.Log.Errorf("Must specify a web path via --path, see --help")
		return
	}

	website, err := con.Rpc.Website(context.Background(), &lispb.Website{
		Name: name,
	})
	if err != nil {
		console.Log.Errorf("%s", err)
		return
	}

	rmWebContent := &lispb.WebsiteRemoveContent{
		Name:  name,
		Paths: []string{},
	}
	if recursive {
		for contentPath := range website.Contents {
			if strings.HasPrefix(contentPath, webPath) {
				rmWebContent.Paths = append(rmWebContent.Paths, contentPath)
			}
		}
	} else {
		rmWebContent.Paths = append(rmWebContent.Paths, webPath)
	}
	_, err = con.Rpc.WebsiteRemoveContent(context.Background(), rmWebContent)
	if err != nil {
		console.Log.Errorf("Failed to remove content %s", err)
		return
	}
	// TODO - PrintWebsite(web, con)
}
