package listener

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
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

	if p, ok := con.Pipelines[name]; ok && p.Enable {
		con.Rpc.StopPipeline(con.Context(), &clientpb.CtrlPipeline{
			Name: name,
		})
	}
	certName, _ := cmd.Flags().GetString("cert-name")
	_, err := con.Rpc.StartPipeline(con.Context(), &clientpb.CtrlPipeline{
		Name:     name,
		CertName: certName,
	})
	if err != nil {
		return err
	}
	return nil
}

func StopPipelineCmd(cmd *cobra.Command, con *core.Console) error {
	name := cmd.Flags().Arg(0)
	_, err := con.Rpc.StopPipeline(con.Context(), &clientpb.CtrlPipeline{
		Name: name,
	})
	if err != nil {
		return err
	}
	return nil
}

func DeletePipelineCmd(cmd *cobra.Command, con *core.Console) error {
	name := cmd.Flags().Arg(0)
	_, err := con.Rpc.DeletePipeline(con.Context(), &clientpb.CtrlPipeline{
		Name: name,
	})
	if err != nil {
		return err
	}
	return nil
}
