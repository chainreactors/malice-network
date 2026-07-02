package website

import (
	"crypto/tls"
	"fmt"
	"github.com/chainreactors/malice-network/client/core"
	"strconv"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
)

// NewWebsiteCmd - 创建新的网站
func NewWebsiteCmd(cmd *cobra.Command, con *core.Console) error {
	name := cmd.Flags().Arg(0)
	root, _ := cmd.Flags().GetString("root")
	auth, _ := cmd.Flags().GetString("auth")
	listenerID, _, host, port := common.ParsePipelineFlags(cmd)
	if port == 0 {
		port = cryptography.RandomInRange(10240, 65535)
	}
	tls, certName, err := common.ParseTLSFlags(cmd)
	if err != nil {
		return err
	}
	if err := validateNewWebsiteTLS(certName, tls); err != nil {
		return err
	}
	if err := NewWebsite(con, name, root, host, port, listenerID, certName, tls, auth); err != nil {
		return err
	}
	saveCert, _ := cmd.Flags().GetBool("save-cert")
	if saveCert {
		update, err := buildWebsiteTLSUpdateFromFlags(cmd, name, listenerID)
		if err != nil {
			return err
		}
		_, err = con.Rpc.UpdateWebsiteTLS(con.Context(), update)
		return err
	}
	return nil
}

// NewWebsite
func NewWebsite(con *core.Console, websiteName, root, host string, port uint32, listenerId, certName string, tls *clientpb.TLS, auth ...string) error {
	var err error
	if err := validateNewWebsiteTLS(certName, tls); err != nil {
		return err
	}
	if root == "" {
		root = "/"
	}
	websiteAuth := ""
	if len(auth) > 0 {
		websiteAuth = auth[0]
	}
	host = "0.0.0.0"
	req := &clientpb.Pipeline{
		Name:       websiteName,
		ListenerId: listenerId,
		Enable:     false,
		Tls:        tls,
		CertName:   certName,
		Ip:         host,
		Body: &clientpb.Pipeline_Web{
			Web: &clientpb.Website{
				Name:     websiteName,
				Root:     root,
				Port:     port,
				Auth:     websiteAuth,
				Contents: make(map[string]*clientpb.WebContent),
			},
		},
	}
	_, err = con.Rpc.RegisterWebsite(con.Context(), req)
	if err != nil {
		return err
	}

	_, err = con.Rpc.StartWebsite(con.Context(), &clientpb.CtrlPipeline{
		Name:       websiteName,
		ListenerId: listenerId,
		Pipeline:   req,
	})
	if err != nil {
		return err
	}
	con.Log.Importantf("Website %s created on port %d\n", websiteName, port)
	return nil
}

func validateNewWebsiteTLS(certName string, tlsConfig *clientpb.TLS) error {
	if tlsConfig == nil || !tlsConfig.Enable {
		return nil
	}
	if certName != "" || tlsConfig.Acme {
		return nil
	}
	if tlsConfig.Cert == nil || (tlsConfig.Cert.Cert == "" && tlsConfig.Cert.Key == "") {
		return nil
	}
	if tlsConfig.Cert.Cert == "" || tlsConfig.Cert.Key == "" {
		return fmt.Errorf("cert and key must be provided together")
	}
	if _, err := tls.X509KeyPair([]byte(tlsConfig.Cert.Cert), []byte(tlsConfig.Cert.Key)); err != nil {
		return fmt.Errorf("invalid certificate key pair: %w", err)
	}
	return nil
}

// StartWebsitePipelineCmd
func StartWebsitePipelineCmd(cmd *cobra.Command, con *core.Console) error {
	websiteName := cmd.Flags().Arg(0)
	certName, _ := cmd.Flags().GetString("cert-name")
	listenerID, _ := cmd.Flags().GetString("listener")
	return startWebsite(con, websiteName, certName, listenerID)
}

func StartWebsite(con *core.Console, websiteName, certName string) error {
	return startWebsite(con, websiteName, certName, "")
}

func startWebsite(con *core.Console, websiteName, certName, listenerID string) error {
	pipelineName, resolvedListenerID, cached := resolveWebsiteTarget(con, websiteName)
	if listenerID == "" {
		listenerID = resolvedListenerID
	}
	if cached {
		_, err := con.Rpc.StopWebsite(con.Context(), &clientpb.CtrlPipeline{
			Name:       pipelineName,
			ListenerId: listenerID,
		})
		if err != nil {
			return err
		}
	}
	_, err := con.Rpc.StartWebsite(con.Context(), &clientpb.CtrlPipeline{
		Name:       pipelineName,
		ListenerId: listenerID,
		CertName:   certName,
	})
	if err != nil {
		return err
	}
	return nil
}

func StopWebsitePipelineCmd(cmd *cobra.Command, con *core.Console) error {
	name := cmd.Flags().Arg(0)
	listenerID, _ := cmd.Flags().GetString("listener")
	return stopWebsite(con, name, listenerID)
}

// StopWebsite
func StopWebsite(con *core.Console, name string) error {
	return stopWebsite(con, name, "")
}

func stopWebsite(con *core.Console, name, listenerID string) error {
	pipelineName, resolvedListenerID, _ := resolveWebsiteTarget(con, name)
	if listenerID == "" {
		listenerID = resolvedListenerID
	}
	_, err := con.Rpc.StopWebsite(con.Context(), &clientpb.CtrlPipeline{
		Name:       pipelineName,
		ListenerId: listenerID,
	})
	if err != nil {
		return err
	}
	return nil
}

func RestartWebsitePipelineCmd(cmd *cobra.Command, con *core.Console) error {
	name := cmd.Flags().Arg(0)
	listenerID, _ := cmd.Flags().GetString("listener")
	return restartWebsite(con, name, listenerID)
}

func RestartWebsite(con *core.Console, name string) error {
	return restartWebsite(con, name, "")
}

func restartWebsite(con *core.Console, name, listenerID string) error {
	pipelineName, resolvedListenerID, _ := resolveWebsiteTarget(con, name)
	if listenerID == "" {
		listenerID = resolvedListenerID
	}
	if err := stopWebsite(con, pipelineName, listenerID); err != nil {
		return err
	}
	_, err := con.Rpc.StartWebsite(con.Context(), &clientpb.CtrlPipeline{
		Name:       pipelineName,
		ListenerId: listenerID,
	})
	return err
}

func WebsiteTLSCmd(cmd *cobra.Command, con *core.Console) error {
	name := cmd.Flags().Arg(0)
	listenerID, _ := cmd.Flags().GetString("listener")
	update, err := buildWebsiteTLSUpdateFromFlags(cmd, name, listenerID)
	if err != nil {
		return err
	}
	_, err = con.Rpc.UpdateWebsiteTLS(con.Context(), update)
	return err
}

func buildWebsiteTLSUpdateFromFlags(cmd *cobra.Command, name, listenerID string) (*clientpb.PipelineTLSUpdate, error) {
	disable, _ := cmd.Flags().GetBool("disable")
	certName, _ := cmd.Flags().GetString("cert-name")
	certPath, _ := cmd.Flags().GetString("cert")
	keyPath, _ := cmd.Flags().GetString("key")
	generateTLS := boolFlag(cmd, "generate")
	saveCert, _ := cmd.Flags().GetBool("save-cert")
	saveCertName, _ := cmd.Flags().GetString("save-cert-name")
	certComment, _ := cmd.Flags().GetString("cert-comment")
	if boolFlag(cmd, "tls") && certName == "" && certPath == "" && keyPath == "" {
		generateTLS = true
	}

	sources := 0
	if disable {
		sources++
	}
	if certName != "" {
		sources++
	}
	if certPath != "" || keyPath != "" {
		sources++
	}
	if generateTLS {
		sources++
	}
	if sources != 1 {
		return nil, fmt.Errorf("specify exactly one TLS mode: --disable, --cert-name, --cert/--key, or --generate")
	}
	if saveCert && (certName != "" || disable) {
		return nil, fmt.Errorf("--save-cert can only be used with generated or inline certs")
	}
	if saveCert && saveCertName == "" {
		return nil, fmt.Errorf("--save-cert-name is required when --save-cert is set")
	}
	update := &clientpb.PipelineTLSUpdate{
		Name:         name,
		ListenerId:   listenerID,
		SaveCert:     saveCert,
		SaveCertName: saveCertName,
		CertComment:  certComment,
	}
	if disable {
		update.Mode = clientpb.TLSUpdateMode_TLS_UPDATE_MODE_DISABLE
		return update, nil
	}
	if certName != "" {
		update.Mode = clientpb.TLSUpdateMode_TLS_UPDATE_MODE_EXISTING_CERT
		update.CertName = certName
		return update, nil
	}
	if generateTLS {
		update.Mode = clientpb.TLSUpdateMode_TLS_UPDATE_MODE_INLINE_CERT
		update.Tls = &clientpb.TLS{Enable: true}
		return update, nil
	}
	if certPath == "" || keyPath == "" {
		return nil, fmt.Errorf("cert and key must be provided together")
	}
	certPEM, err := cryptography.ProcessPEM(certPath)
	if err != nil {
		return nil, err
	}
	keyPEM, err := cryptography.ProcessPEM(keyPath)
	if err != nil {
		return nil, err
	}
	if _, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM)); err != nil {
		return nil, fmt.Errorf("invalid certificate key pair: %w", err)
	}
	update.Mode = clientpb.TLSUpdateMode_TLS_UPDATE_MODE_INLINE_CERT
	update.Tls = &clientpb.TLS{
		Enable: true,
		Cert: &clientpb.Cert{
			Cert:    certPEM,
			Key:     keyPEM,
			Type:    "imported",
			Name:    saveCertName,
			Comment: certComment,
		},
	}
	return update, nil
}

func boolFlag(cmd *cobra.Command, name string) bool {
	if cmd == nil || cmd.Flags().Lookup(name) == nil {
		return false
	}
	value, _ := cmd.Flags().GetBool(name)
	return value
}

func ListWebsitesCmd(cmd *cobra.Command, con *core.Console) error {
	listenerID := cmd.Flags().Arg(0)
	websites, err := con.Rpc.ListWebsites(con.Context(), &clientpb.Listener{
		Id: listenerID,
	})
	if err != nil {
		return err
	}
	var rowEntries []table.Row
	var row table.Row
	tableModel := tui.NewTable([]table.Column{
		table.NewFlexColumn("Name", "Name", 1),
		table.NewColumn("Port", "Port", 7),
		table.NewFlexColumn("RootPath", "Root Path", 1),
		table.NewColumn("Enable", "Enable", 7),
	}, true)
	if len(websites.Pipelines) == 0 {
		con.Log.Importantf("No websites found")
		return nil
	}
	for _, p := range websites.Pipelines {
		w := p.GetWeb()
		row = table.NewRow(
			table.RowData{
				"Name":     p.Name,
				"Port":     strconv.Itoa(int(w.Port)),
				"RootPath": w.Root,
				"Enable":   p.Enable,
			})
		rowEntries = append(rowEntries, row)
	}
	tableModel.SetMultiline()
	tableModel.SetRows(rowEntries)
	con.Log.Console(tableModel.View())
	return nil
}
