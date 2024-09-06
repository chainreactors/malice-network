package website

import (
	"context"
	"fmt"
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/bubbles/table"
	"strconv"
)

func listWebsitesCmd(c *grumble.Context, con *repl.Console) {
	name := c.Flags.String("name")
	if name == "" {
		websites, err := con.Rpc.Websites(context.Background(), &clientpb.Empty{})
		if err != nil {
			repl.Log.Errorf("Failed to list websites %s", err)
			return
		}
		for _, website := range websites.Websites {
			fmt.Printf("List the contents of website '%s':\n", website.Name)
		}
		return
	} else {
		website, err := con.Rpc.Website(context.Background(), &lispb.Website{
			Name: name,
		})
		if err != nil {
			fmt.Printf("Failed to list website content %s", err)
			return
		}
		if 0 < len(website.Contents) {
			PrintWebsite(website)
		} else {
			fmt.Printf("No content for '%s'", name)
		}
	}
	return
}

func PrintWebsite(web *lispb.Website) {
	var rowEntries []table.Row
	var row table.Row
	tableModel := tui.NewTable([]table.Column{
		{Title: "Path", Width: 20},
		{Title: "Content-type", Width: 20},
		{Title: "Size", Width: 10},
	}, true)
	for _, content := range web.Contents {
		row = table.Row{
			content.Path,
			content.ContentType,
			strconv.FormatUint(content.Size, 10),
		}
		rowEntries = append(rowEntries, row)
	}
	tableModel.SetRows(rowEntries)
	newTable := tui.NewModel(tableModel, nil, false, false)
	err := newTable.Run()
	if err != nil {
		return
	}
}
