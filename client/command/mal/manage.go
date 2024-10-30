package mal

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/bubbles/table"
	"github.com/spf13/cobra"
	"net/url"
	"path/filepath"
	"time"
)

var (
	defaultTimeout = 15 * time.Minute
)

// MalHTTPConfig - Configuration for armory HTTP client
type MalHTTPConfig struct {
	MalConfig            *assets.MalConfig
	IgnoreCache          bool
	ProxyURL             *url.URL
	Timeout              time.Duration
	DisableTLSValidation bool
}

func parseMalHTTPConfig(cmd *cobra.Command) MalHTTPConfig {
	var proxyURL *url.URL
	rawProxyURL, _ := cmd.Flags().GetString("proxy")
	if rawProxyURL != "" {
		proxyURL, _ = url.Parse(rawProxyURL)
	}

	timeout := defaultTimeout
	rawTimeout, _ := cmd.Flags().GetString("timeout")
	if rawTimeout != "" {
		var err error
		timeout, err = time.ParseDuration(rawTimeout)
		if err != nil {
			timeout = defaultTimeout
		}
	}
	ignoreCache, _ := cmd.Flags().GetBool("ignore-cache")
	insecure, _ := cmd.Flags().GetBool("insecure")
	return MalHTTPConfig{
		IgnoreCache:          ignoreCache,
		ProxyURL:             proxyURL,
		Timeout:              timeout,
		DisableTLSValidation: insecure,
	}
}

func MalCmd(cmd *cobra.Command, con *repl.Console) error {
	malHttpConfig := parseMalHTTPConfig(cmd)
	//malIndex, _ := DefaultMalIndexParser(malHttpConfig)
	malsJson, err := parserMalJson(malHttpConfig)
	if err != nil {
		return err
	}
	if len(malsJson.Mals) > 0 {
		err = printMals(malsJson, malHttpConfig, con)
		if err != nil {
			return err
		}
	} else {
		logs.Log.Infof("No mals found")
	}
	return nil
}

func printMals(maljson MalsJson, malHttpConfig MalHTTPConfig, con *repl.Console) error {
	var rowEntries []table.Row
	var row table.Row

	tableModel := tui.NewTable([]table.Column{
		{Title: "Name", Width: 25},
		{Title: "Version", Width: 10},
		{Title: "Repo_url", Width: 50},
		{Title: "Help", Width: 50},
	}, false)
	for _, mal := range maljson.Mals {
		row = table.Row{
			mal.Name,
			mal.Version,
			mal.RepoURL,
			mal.Help,
		}
		rowEntries = append(rowEntries, row)
	}
	tableModel.SetRows(rowEntries)
	tableModel.SetHandle(func() {
		installMal(tableModel, malHttpConfig, con)
	})
	newTable := tui.NewModel(tableModel, tableModel.ConsoleHandler, true, false)
	err := newTable.Run()
	if err != nil {
		return err
	}
	tui.Reset()
	return nil
}

func installMal(tableModel *tui.TableModel, malHttpConfig MalHTTPConfig, con *repl.Console) func() {
	selectRow := tableModel.GetSelectedRow()
	logs.Log.Infof("Installing mal: %s", selectRow[0])
	err := GithubMalPackageParser(selectRow[2], selectRow[0], selectRow[1], malHttpConfig)
	if err != nil {
		return func() {
			con.Log.Errorf("Error installing mal: %s", err)
		}
	}
	tarGzPath := filepath.Join(assets.GetMalsDir(), selectRow[0]+".tar.gz")
	InstallFromDir(tarGzPath, true, con)
	return func() {
	}
}
