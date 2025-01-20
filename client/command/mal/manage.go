package mal

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/mals/m"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
	"io"
	"net/url"
	"path/filepath"
	"time"
)

var (
	defaultTimeout = 15 * time.Minute
)

func parseMalHTTPConfig(cmd *cobra.Command) m.MalHTTPConfig {
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
	return m.MalHTTPConfig{
		IgnoreCache:          ignoreCache,
		ProxyURL:             proxyURL,
		Timeout:              timeout,
		DisableTLSValidation: insecure,
	}
}

func MalCmd(cmd *cobra.Command, con *repl.Console) error {
	malHttpConfig := parseMalHTTPConfig(cmd)
	//malIndex, _ := DefaultMalIndexParser(malHttpConfig)
	malsJson, err := m.ParserMalYaml(m.DefaultMalRepoURL, filepath.Join(assets.GetConfigDir(), m.MalIndexFileName), malHttpConfig)
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

func printMals(maljson m.MalsYaml, malHttpConfig m.MalHTTPConfig, con *repl.Console) error {
	var rowEntries []table.Row
	var row table.Row

	tableModel := tui.NewTable([]table.Column{
		table.NewColumn("Name", "Name", 25),
		table.NewColumn("Version", "Version", 10),
		table.NewColumn("Repo_url", "Repo_url", 50),
		table.NewColumn("Help", "Help", 50),
	}, false)
	for _, mal := range maljson.Mals {
		row = table.NewRow(
			table.RowData{
				"Name":     mal.Name,
				"Version":  mal.Version,
				"Repo_url": mal.RepoURL,
				"Help":     mal.Help,
			})
		rowEntries = append(rowEntries, row)
	}
	newTable := tui.NewModel(tableModel, nil, false, false)

	tableModel.SetMultiline()
	tableModel.SetRows(rowEntries)
	tableModel.SetHandle(func() {
		InstallMal(tableModel, newTable.Buffer, malHttpConfig, con)
	})
	err := newTable.Run()
	if err != nil {
		return err
	}
	tui.Reset()
	return nil
}

func InstallMal(tableModel *tui.TableModel, writer io.Writer, malHttpConfig m.MalHTTPConfig, con *repl.Console) func() {
	selectRow := tableModel.GetHighlightedRow()
	if selectRow.Data == nil {
		return func() {
			con.Log.FErrorf(writer, "No row selected\n")
			return
		}
	}
	logs.Log.Infof("Installing mal: %s", selectRow.Data["Name"].(string))
	err := m.GithubMalPackageParser(
		selectRow.Data["Repo_url"].(string),
		selectRow.Data["Name"].(string),
		selectRow.Data["Version"].(string),
		assets.GetMalsDir(),
		malHttpConfig)
	if err != nil {
		return func() {
			con.Log.FErrorf(writer, "Error installing mal: %s\n", err)
		}
	}
	tarGzPath := filepath.Join(assets.GetMalsDir(), selectRow.Data["Name"].(string)+".tar.gz")
	InstallFromDir(tarGzPath, true, con)
	return func() {
	}
}
