package generic

import (
	"fmt"
	"strings"

	iomclient "github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
)

func PivotCommand(con *core.Console) *cobra.Command {
	pivotCmd := &cobra.Command{
		Use:   consts.CommandPivot,
		Short: "Manage pivot agents",
		Long:  "List and manage active pivot agents",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return ListPivotCmd(cmd, con)
		},
		Example: `List all pivot agents:
~~~
pivot
pivot list --all
~~~

Manage a pivot:
~~~
pivot status <agent_id>
pivot stop <agent_id>
pivot log <agent_id>
~~~`,
	}
	bindPivotListFlags(pivotCmd)

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List pivot agents",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return ListPivotCmd(cmd, con)
		},
	}
	bindPivotListFlags(listCmd)

	statusCmd := &cobra.Command{
		Use:   "status <agent_id>",
		Short: "Query a pivot agent status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return PivotStatusCmd(cmd, con, args[0])
		},
	}
	bindPivotTargetFlags(statusCmd)

	stopCmd := &cobra.Command{
		Use:   "stop <agent_id>",
		Short: "Stop a managed pivot agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return PivotStopCmd(cmd, con, args[0])
		},
	}
	bindPivotTargetFlags(stopCmd)

	logCmd := &cobra.Command{
		Use:   "log <agent_id>",
		Short: "Read a pivot agent log",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return PivotLogCmd(cmd, con, args[0])
		},
	}
	bindPivotTargetFlags(logCmd)

	pivotCmd.AddCommand(listCmd, statusCmd, stopCmd, logCmd)
	return pivotCmd
}

func bindPivotListFlags(cmd *cobra.Command) {
	cmd.Flags().BoolP("all", "a", false, "list all pivot agents")
}

func bindPivotTargetFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("pipeline", "p", "", "select a pivot by pipeline or listener:pipeline")
}

func ListPivotCmd(cmd *cobra.Command, con *core.Console) error {
	all, _ := cmd.Flags().GetBool("all")
	pivots, err := con.Rpc.GetContexts(con.Context(), &clientpb.Context{
		Type: consts.ContextPivoting,
	})
	if err != nil {
		return err
	}

	if len(pivots.Contexts) == 0 {
		logs.Log.Info("No pivots\n")
		return nil
	}

	PrintPivots(pivots.Contexts, con, all)
	return nil
}

func ListPivot(con *core.Console) ([]*output.PivotingContext, error) {
	pivots, err := con.Rpc.GetContexts(con.Context(), &clientpb.Context{
		Type: consts.ContextPivoting,
	})
	if err != nil {
		return nil, err
	}
	ctxs, err := output.ToContexts[*output.PivotingContext](pivots.Contexts)
	return ctxs, nil
}

func PivotStatusCmd(cmd *cobra.Command, con *core.Console, agentID string) error {
	return pivotRemDialControl(cmd, con, "status", agentID)
}

func PivotStopCmd(cmd *cobra.Command, con *core.Console, agentID string) error {
	return pivotRemDialControl(cmd, con, "stop", agentID)
}

func PivotLogCmd(cmd *cobra.Command, con *core.Console, agentID string) error {
	_, pivot, err := resolvePivotContext(cmd, con, agentID)
	if err != nil {
		return err
	}
	remLog, err := con.Rpc.RemAgentLog(con.Context(), &clientpb.REMAgent{
		PipelineId: pivotPipelineID(pivot),
		Id:         agentID,
	})
	if err != nil {
		return err
	}
	if remLog.GetLog() != "" {
		con.Log.Console(remLog.GetLog())
		if !strings.HasSuffix(remLog.GetLog(), "\n") {
			con.Log.Console("\n")
		}
	}
	return nil
}

func pivotRemDialControl(cmd *cobra.Command, con *core.Console, action, agentID string) error {
	ctx, pivot, err := resolvePivotContext(cmd, con, agentID)
	if err != nil {
		return err
	}
	sess, err := sessionForPivot(con, ctx, agentID)
	if err != nil {
		return err
	}
	task, err := con.Rpc.RemDial(sess.Context(), &implantpb.Request{
		Name: consts.ModuleRemDial,
		Args: []string{action, agentID},
		Params: map[string]string{
			"pipeline_id": pivotPipelineID(pivot),
		},
	})
	if err != nil {
		return err
	}
	sess.Console(task, string(*con.App.Shell().Line()))
	return nil
}

func resolvePivotContext(cmd *cobra.Command, con *core.Console, agentID string) (*clientpb.Context, *output.PivotingContext, error) {
	pipeline, _ := cmd.Flags().GetString("pipeline")
	pivots, err := con.Rpc.GetContexts(con.Context(), &clientpb.Context{
		Type: consts.ContextPivoting,
	})
	if err != nil {
		return nil, nil, err
	}

	var matches []struct {
		ctx   *clientpb.Context
		pivot *output.PivotingContext
	}
	for _, ctx := range pivots.Contexts {
		pivot, err := output.ToContext[*output.PivotingContext](ctx)
		if err != nil || pivot.RemAgentID != agentID {
			continue
		}
		if !pivotMatchesPipeline(ctx, pivot, pipeline) {
			continue
		}
		matches = append(matches, struct {
			ctx   *clientpb.Context
			pivot *output.PivotingContext
		}{ctx: ctx, pivot: pivot})
	}

	switch len(matches) {
	case 0:
		return nil, nil, fmt.Errorf("pivot agent %s not found", agentID)
	case 1:
		return matches[0].ctx, matches[0].pivot, nil
	default:
		return nil, nil, fmt.Errorf("multiple pivots match agent %s; use --pipeline to select one", agentID)
	}
}

func pivotMatchesPipeline(ctx *clientpb.Context, pivot *output.PivotingContext, filter string) bool {
	if filter == "" {
		return true
	}
	for _, candidate := range pivotPipelineCandidates(ctx, pivot) {
		if candidate == filter {
			return true
		}
	}
	return false
}

func pivotPipelineCandidates(ctx *clientpb.Context, pivot *output.PivotingContext) []string {
	candidates := []string{pivot.Pipeline}
	if pivot.Listener != "" && pivot.Pipeline != "" {
		candidates = append(candidates, pivot.Listener+":"+pivot.Pipeline)
	}
	if ctx.GetPipeline() != nil {
		candidates = append(candidates, ctx.GetPipeline().GetName())
		if ctx.GetPipeline().GetListenerId() != "" && ctx.GetPipeline().GetName() != "" {
			candidates = append(candidates, ctx.GetPipeline().GetListenerId()+":"+ctx.GetPipeline().GetName())
		}
	}
	return candidates
}

func pivotPipelineID(pivot *output.PivotingContext) string {
	if pivot.Listener != "" && pivot.Pipeline != "" {
		return pivot.Listener + ":" + pivot.Pipeline
	}
	return pivot.Pipeline
}

func sessionForPivot(con *core.Console, ctx *clientpb.Context, agentID string) (*iomclient.Session, error) {
	sessionID := ctx.GetSession().GetSessionId()
	if sessionID == "" {
		return nil, fmt.Errorf("pivot agent %s has no owning session", agentID)
	}
	if sess, ok := con.GetLocalSession(sessionID); ok {
		return sess, nil
	}
	sessionPB, err := con.Rpc.GetSession(con.Context(), &clientpb.SessionRequest{SessionId: sessionID})
	if err != nil {
		return nil, err
	}
	return con.AddSession(sessionPB), nil
}

func PrintPivots(contexts []*clientpb.Context, con *core.Console, all bool) {
	var rowEntries []table.Row
	for _, ctx := range contexts {
		pivot, err := output.ToContext[*output.PivotingContext](ctx)
		if err != nil {
			continue
		}

		sessionID := ""
		if ctx.Session != nil {
			sessionID = ctx.Session.SessionId
		}

		row := table.NewRow(
			table.RowData{
				"Session":     sessionID,
				"Enable":      fmt.Sprintf("%t", pivot.Enable),
				"Listener":    pivot.Listener,
				"Pipeline":    pivot.Pipeline,
				"RemAgentID":  pivot.RemAgentID,
				"LocalURL":    pivot.LocalURL,
				"RemoteURL":   pivot.RemoteURL,
				"InboundSide": pivot.InboundSide,
			})
		if all || pivot.Enable {
			rowEntries = append(rowEntries, row)
		}
	}

	tableModel := tui.NewTable([]table.Column{
		table.NewColumn("Session", "Session", 10),
		table.NewColumn("Enable", "Enable", 6),
		table.NewColumn("Listener", "Listener", 10),
		table.NewColumn("Pipeline", "Pipeline", 10),
		table.NewColumn("RemAgentID", "Rem Agent ID", 10),
		table.NewFlexColumn("LocalURL", "Local URL", 1),
		table.NewFlexColumn("RemoteURL", "Remote URL", 1),
		table.NewColumn("InboundSide", "Inbound", 10),
	}, true)

	tableModel.SetMultiline()
	tableModel.SetRows(rowEntries)
	con.Log.Console(tableModel.View())
}
