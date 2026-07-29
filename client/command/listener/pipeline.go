package listener

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

func ListPipelineCmd(cmd *cobra.Command, con *core.Console) error {
	listenerID := cmd.Flags().Arg(0)
	pipelines, err := con.Rpc.ListPipelines(con.Context(), &clientpb.Listener{
		Id: listenerID,
	})
	if err != nil {
		return err
	}
	if len(pipelines.Pipelines) == 0 {
		con.Log.Warnf("No pipelines found")
		return nil
	}
	var rowEntries []table.Row
	tableModel := tui.NewTable([]table.Column{
		table.NewFlexColumn("Name", "Name", 1),
		table.NewColumn("Enable", "Enable", 7),
		table.NewColumn("Type", "Type", 6),
		table.NewColumn("ListenerID", "Listener ID", 11),
		table.NewFlexColumn("Address", "Address", 1),
		table.NewColumn("Parser", "Parser", 7),
		table.NewColumn("Encryption", "Encryption", 12),
		table.NewColumn("TLS", "TLS", 6),
	}, true)
	for _, pipeline := range pipelines.GetPipelines() {
		if pipeline == nil || pipeline.Body == nil {
			continue
		}
		newRow := table.RowData{}
		var schema string
		if pipeline.Enable {
			newRow["Enable"] = tui.GreenFg.Render(strconv.FormatBool(pipeline.Enable))
		} else {
			newRow["Enable"] = tui.RedFg.Render(strconv.FormatBool(pipeline.Enable))
		}
		if pipeline.Tls != nil && pipeline.Tls.Enable {
			newRow["TLS"] = tui.GreenFg.Render(strconv.FormatBool(pipeline.Tls.Enable))
		} else if pipeline.Tls != nil {
			newRow["TLS"] = tui.RedFg.Render(strconv.FormatBool(pipeline.Tls.Enable))
		}
		if pipeline.Encryption != nil {
			encryption := make([]string, 0, len(pipeline.Encryption))
			for _, enc := range pipeline.Encryption {
				encryption = append(encryption, fmt.Sprintf("%s/%s", enc.Type, enc.Key))
			}
			newRow["Encryption"] = strings.Join(encryption, ",")
		} else {
			newRow["Encryption"] = "raw"
		}
		switch body := pipeline.Body.(type) {
		case *clientpb.Pipeline_Http:
			newRow["Name"] = pipeline.Name
			newRow["Type"] = consts.HTTPPipeline
			newRow["ListenerID"] = pipeline.ListenerId
			if pipeline.Tls != nil && pipeline.Tls.Enable {
				schema = "https://"
			} else {
				schema = "http://"
			}
			newRow["Address"] = schema + pipeline.Ip + ":" + strconv.Itoa(int(body.Http.Port))
			newRow["Parser"] = pipeline.Parser
		case *clientpb.Pipeline_Tcp:
			newRow["Name"] = pipeline.Name
			newRow["Type"] = consts.TCPPipeline
			newRow["ListenerID"] = pipeline.ListenerId
			if pipeline.Tls != nil && pipeline.Tls.Enable {
				schema = "tcp+tls://"
			} else {
				schema = "tcp://"
			}
			newRow["Address"] = schema + pipeline.Ip + ":" + strconv.Itoa(int(body.Tcp.Port))
			newRow["Parser"] = pipeline.Parser
		case *clientpb.Pipeline_Rem:
			newRow["Name"] = pipeline.Name
			newRow["Type"] = consts.RemPipeline
			newRow["ListenerID"] = pipeline.ListenerId
			newRow["Parser"] = pipeline.Parser
		case *clientpb.Pipeline_Bind:
			newRow["Name"] = pipeline.Name
			newRow["Type"] = consts.BindPipeline
			newRow["ListenerID"] = pipeline.ListenerId
			newRow["Parser"] = pipeline.Parser
		case *clientpb.Pipeline_Custom:
			newRow["Name"] = pipeline.Name
			newRow["Type"] = pipeline.Type
			newRow["ListenerID"] = pipeline.ListenerId
			if body.Custom.Host != "" {
				addr := body.Custom.Host
				if body.Custom.Port > 0 {
					addr += ":" + strconv.Itoa(int(body.Custom.Port))
				}
				newRow["Address"] = addr
			}
			newRow["Parser"] = pipeline.Parser
		default:
			newRow["Name"] = pipeline.Name
			newRow["Type"] = pipeline.Type
			newRow["ListenerID"] = pipeline.ListenerId
		}
		rowEntries = append(rowEntries, table.NewRow(newRow))
	}
	tableModel.SetRows(rowEntries)
	con.Log.Console(tableModel.View())
	return nil
}

func StartPipelineCmd(cmd *cobra.Command, con *core.Console) error {
	name := cmd.Flags().Arg(0)
	pipelineName, listenerID, cached := resolvePipelineCtrlTarget(con, name)
	certName, _ := cmd.Flags().GetString("cert-name")
	if certName != "" {
		_, err := con.Rpc.StartPipeline(con.Context(), &clientpb.CtrlPipeline{
			Name:       pipelineName,
			ListenerId: listenerID,
			CertName:   certName,
		})
		return err
	}

	p, _ := common.FindCachedPipeline(con, name, nil)
	if cached && p != nil && p.Enable {
		_, err := con.Rpc.StopPipeline(con.Context(), &clientpb.CtrlPipeline{
			Name:       pipelineName,
			ListenerId: listenerID,
		})
		if err != nil {
			return err
		}
	}
	_, err := con.Rpc.StartPipeline(con.Context(), &clientpb.CtrlPipeline{
		Name:       pipelineName,
		ListenerId: listenerID,
	})
	if err != nil {
		return err
	}
	return nil
}

func StopPipelineCmd(cmd *cobra.Command, con *core.Console) error {
	name := cmd.Flags().Arg(0)
	pipelineName, listenerID, _ := resolvePipelineCtrlTarget(con, name)
	_, err := con.Rpc.StopPipeline(con.Context(), &clientpb.CtrlPipeline{
		Name:       pipelineName,
		ListenerId: listenerID,
	})
	if err != nil {
		return err
	}
	return nil
}

func DeletePipelineCmd(cmd *cobra.Command, con *core.Console) error {
	name := cmd.Flags().Arg(0)
	pipelineName, listenerID, _ := resolvePipelineCtrlTarget(con, name)
	_, err := con.Rpc.DeletePipeline(con.Context(), &clientpb.CtrlPipeline{
		Name:       pipelineName,
		ListenerId: listenerID,
	})
	if err != nil {
		return err
	}
	return nil
}

func InspectPipelineCmd(cmd *cobra.Command, con *core.Console) error {
	name := cmd.Flags().Arg(0)
	pipeline, err := findPipeline(con, name)
	if err != nil {
		return err
	}
	printPipelineDetail(pipeline)
	return nil
}

func RestartPipelineCmd(cmd *cobra.Command, con *core.Console) error {
	name := cmd.Flags().Arg(0)
	pipelineName, listenerID, _ := resolvePipelineCtrlTarget(con, name)
	if _, err := con.Rpc.StopPipeline(con.Context(), &clientpb.CtrlPipeline{Name: pipelineName, ListenerId: listenerID}); err != nil {
		return err
	}
	_, err := con.Rpc.StartPipeline(con.Context(), &clientpb.CtrlPipeline{Name: pipelineName, ListenerId: listenerID})
	return err
}

func HealthPipelineCmd(cmd *cobra.Command, con *core.Console) error {
	listenerID, _ := cmd.Flags().GetString("listener")
	pipelines, err := con.Rpc.ListPipelines(con.Context(), &clientpb.Listener{Id: listenerID})
	if err != nil {
		return err
	}
	jobs, err := con.Rpc.ListJobs(con.Context(), &clientpb.Empty{})
	if err != nil {
		return err
	}
	running := map[string]struct{}{}
	for _, job := range jobs.GetPipelines() {
		running[job.GetListenerId()+":"+job.GetName()] = struct{}{}
	}
	total := 0
	enabled := 0
	live := 0
	for _, pipeline := range pipelines.GetPipelines() {
		total++
		if pipeline.GetEnable() {
			enabled++
		}
		if _, ok := running[pipeline.GetListenerId()+":"+pipeline.GetName()]; ok {
			live++
		}
	}
	tui.RenderKVWithOptions(map[string]interface{}{
		"Pipelines": total,
		"Enabled":   enabled,
		"Running":   live,
	}, []string{"Pipelines", "Enabled", "Running"}, tui.KVOptions{ShowHeader: true})
	return nil
}

func UpdatePipelineCmd(cmd *cobra.Command, con *core.Console) error {
	if cmd.Flags().Changed("cert-name") {
		return fmt.Errorf("--cert-name is not supported by pipeline update; use pipeline cert bind --pipeline <name> --cert-name <cert>")
	}
	name := cmd.Flags().Arg(0)
	pipelineName, _, cached := resolvePipelineCtrlTarget(con, name)
	if !cached {
		return fmt.Errorf("pipeline %s is not cached; refresh client state before update", name)
	}
	current, err := common.FindCachedPipeline(con, name, nil)
	if err != nil || current == nil {
		return fmt.Errorf("pipeline %s not found", name)
	}
	updated := proto.Clone(current).(*clientpb.Pipeline)
	if cmd.Flags().Changed("enable") {
		enable, _ := cmd.Flags().GetBool("enable")
		updated.Enable = enable
	}
	if cmd.Flags().Changed("disable") {
		disable, _ := cmd.Flags().GetBool("disable")
		if disable {
			updated.Enable = false
		}
	}
	if cmd.Flags().Changed("parser") {
		parser, _ := cmd.Flags().GetString("parser")
		updated.Parser = parser
	}
	if updated.Name == "" {
		updated.Name = pipelineName
	}
	_, err = con.Rpc.SyncPipeline(con.Context(), updated)
	return err
}

func findPipeline(con *core.Console, key string) (*clientpb.Pipeline, error) {
	if pipeline, err := common.FindCachedPipeline(con, key, nil); err == nil {
		return pipeline, nil
	}
	listenerID, name, ok := strings.Cut(key, ":")
	if !ok {
		name = key
		listenerID = ""
	}
	pipelines, err := con.Rpc.ListPipelines(con.Context(), &clientpb.Listener{Id: listenerID})
	if err != nil {
		return nil, err
	}
	for _, pipeline := range pipelines.GetPipelines() {
		if pipeline.GetName() == name && (listenerID == "" || pipeline.GetListenerId() == listenerID) {
			return pipeline, nil
		}
	}
	return nil, errors.New("pipeline not found")
}

func printPipelineDetail(pipeline *clientpb.Pipeline) {
	tlsEnabled := false
	if pipeline.GetTls() != nil {
		tlsEnabled = pipeline.GetTls().GetEnable()
	}
	detail := map[string]interface{}{
		"Name":       pipeline.GetName(),
		"ListenerID": pipeline.GetListenerId(),
		"Type":       pipeline.GetType(),
		"Enable":     pipeline.GetEnable(),
		"IP":         pipeline.GetIp(),
		"Parser":     pipeline.GetParser(),
		"CertName":   pipeline.GetCertName(),
		"TLS":        tlsEnabled,
	}
	tui.RenderKVWithOptions(detail, []string{"Name", "ListenerID", "Type", "Enable", "IP", "Parser", "CertName", "TLS"}, tui.KVOptions{ShowHeader: true})
}

func resolvePipelineCtrlTarget(con *core.Console, key string) (string, string, bool) {
	pipeline, err := common.FindCachedPipeline(con, key, nil)
	if err != nil || pipeline == nil {
		if listenerID, name, ok := strings.Cut(key, ":"); ok && listenerID != "" && name != "" {
			return name, listenerID, false
		}
		return key, "", false
	}
	return pipeline.Name, pipeline.ListenerId, true
}
