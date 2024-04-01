package website

import (
	"context"
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
)

func websiteUpdateContentCmd(c *grumble.Context, con *console.Console) {
	name := c.Flags.String("name")
	webPath := c.Flags.String("web-path")
	contentType := c.Flags.String("content-type")
	if name == "" {
		console.Log.Errorf("Must specify a website name via --name, see --help")
		return
	}
	if webPath == "" {
		console.Log.Errorf("Must specify a web path via --wen-path, see --help")
		return
	}
	if contentType == "" {
		console.Log.Errorf("Must specify a content type via --content-type, see --help")
		return
	}

	updateWeb := &clientpb.WebsiteAddContent{
		Name:     name,
		Contents: map[string]*clientpb.WebContent{},
	}
	updateWeb.Contents[webPath] = &clientpb.WebContent{
		ContentType: contentType,
	}
	_, err := con.Rpc.WebsiteUpdateContent(context.Background(), updateWeb)
	if err != nil {
		console.Log.Errorf("Failed to update content %s", err)
		return
	}
	// TODO - PrintWebsite(web, con)
}
