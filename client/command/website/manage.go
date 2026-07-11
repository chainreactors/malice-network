package website

import (
	"fmt"
	"strconv"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
)

func InspectWebsiteCmd(cmd *cobra.Command, con *core.Console) error {
	name := cmd.Flags().Arg(0)
	website, err := findWebsite(con, name)
	if err != nil {
		return err
	}
	contents, err := listWebsiteContents(con, website)
	if err != nil {
		return err
	}
	web := website.GetWeb()
	tui.RenderKVWithOptions(map[string]interface{}{
		"Name":       website.GetName(),
		"ListenerID": website.GetListenerId(),
		"Port":       strconv.Itoa(int(web.GetPort())),
		"Root":       web.GetRoot(),
		"Enable":     website.GetEnable(),
		"CertName":   website.GetCertName(),
		"Contents":   len(contents.GetContents()),
	}, []string{"Name", "ListenerID", "Port", "Root", "Enable", "CertName", "Contents"}, tui.KVOptions{ShowHeader: true})
	return nil
}

func WebsiteCertCmd(cmd *cobra.Command, con *core.Console) error {
	return WebsiteTLSCmd(cmd, con)
}

func findWebsite(con *core.Console, key string) (*clientpb.Pipeline, error) {
	if pipeline, err := common.FindCachedPipeline(con, key, func(candidate *clientpb.Pipeline) bool {
		return candidate.GetWeb() != nil
	}); err == nil {
		return pipeline, nil
	}
	websiteName, listenerID, _ := resolveWebsiteTarget(con, key)
	websites, err := con.Rpc.ListWebsites(con.Context(), &clientpb.Listener{Id: listenerID})
	if err != nil {
		return nil, err
	}
	for _, pipeline := range websites.GetPipelines() {
		if pipeline.GetName() == websiteName && pipeline.GetWeb() != nil && (listenerID == "" || pipeline.GetListenerId() == listenerID) {
			return pipeline, nil
		}
	}
	return nil, fmt.Errorf("website %s not found", key)
}

func listWebsiteContents(con *core.Console, pipeline *clientpb.Pipeline) (*clientpb.WebContents, error) {
	return con.Rpc.ListWebContent(con.Context(), &clientpb.Website{
		Name:       pipeline.GetName(),
		ListenerId: pipeline.GetListenerId(),
	})
}
