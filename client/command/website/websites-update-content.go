package website

import (
	"context"
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/proto/listener/lispb"
)

func websiteUpdateContentCmd(c *grumble.Context, con *repl.Console) {
	name := c.Flags.String("name")
	webPath := c.Flags.String("web-path")
	contentType := c.Flags.String("content-type")
	if name == "" {
		repl.Log.Errorf("Must specify a website name via --name, see --help")
		return
	}
	if webPath == "" {
		repl.Log.Errorf("Must specify a web path via --wen-path, see --help")
		return
	}
	if contentType == "" {
		repl.Log.Errorf("Must specify a content type via --content-type, see --help")
		return
	}

	updateWeb := &lispb.WebsiteAddContent{
		Name:     name,
		Contents: map[string]*lispb.WebContent{},
	}
	updateWeb.Contents[webPath] = &lispb.WebContent{
		ContentType: contentType,
	}
	_, err := con.Rpc.WebsiteUpdateContent(context.Background(), updateWeb)
	if err != nil {
		repl.Log.Errorf("Failed to update content %s", err)
		return
	}
	// TODO - PrintWebsite(web, con)
}
