package pipeline

import (
	"fmt"
	"strconv"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/malice-network/helper/third/rem"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
)

func ListRemCmd(cmd *cobra.Command, con *core.Console) error {
	listenerID := cmd.Flags().Arg(0)
	pipes, err := con.Rpc.ListPipelines(con.Context(), &clientpb.Listener{
		Id: listenerID,
	})
	if err != nil {
		return err
	}
	if len(pipes.Pipelines) == 0 {
		con.Log.Warnf("No REMs found\n")
		return nil
	}
	var rems []*clientpb.REM
	for _, pipe := range pipes.Pipelines {
		if pipe.Enable && pipe.Type == consts.RemPipeline {
			rems = append(rems, pipe.GetRem())
		}
	}

	con.Log.Console(tui.RendStructDefault(rems) + "\n")
	return nil
}

func NewRemCmd(cmd *cobra.Command, con *core.Console) error {
	name := cmd.Flags().Arg(0)
	listenerID, _, _, _ := common.ParsePipelineFlags(cmd)
	console, _ := cmd.Flags().GetString("console")

	parse, err := rem.ParseConsole(console)
	if err != nil {
		return err
	}
	if parse.Port() == 34996 {
		parse.SetPort(int(cryptography.RandomInRange(20000, 60000)))
	}
	port, err := strconv.Atoi(parse.URL.Port())
	if err != nil {
		return err
	}
	if name == "" {
		name = fmt.Sprintf("rem_%s_%d", listenerID, port)
	}
	pipeline := &clientpb.Pipeline{
		Name:       name,
		ListenerId: listenerID,
		Enable:     true,
		Body: &clientpb.Pipeline_Rem{
			Rem: &clientpb.REM{
				Host:    parse.Hostname(),
				Port:    uint32(port),
				Console: parse.String(),
			},
		},
	}

	_, err = con.Rpc.RegisterRem(con.Context(), pipeline)
	if err != nil {
		return err
	}

	_, err = con.Rpc.StartRem(con.Context(), &clientpb.CtrlPipeline{
		Name:       name,
		ListenerId: listenerID,
	})
	if err != nil {
		return err
	}

	return nil
}

func StartRemCmd(cmd *cobra.Command, con *core.Console) error {
	name := cmd.Flags().Arg(0)
	_, err := con.Rpc.StartRem(con.Context(), &clientpb.CtrlPipeline{
		Name: name,
	})
	if err != nil {
		return err
	}
	return nil
}

func StopRemCmd(cmd *cobra.Command, con *core.Console) error {
	name := cmd.Flags().Arg(0)
	_, err := con.Rpc.StopRem(con.Context(), &clientpb.CtrlPipeline{
		Name: name,
	})
	if err != nil {
		return err
	}
	return nil
}

func DeleteRemCmd(cmd *cobra.Command, con *core.Console) error {
	name := cmd.Flags().Arg(0)
	_, err := con.Rpc.DeleteRem(con.Context(), &clientpb.CtrlPipeline{
		Name: name,
	})
	if err != nil {
		return err
	}
	return nil
}

func RemUpdateIntervalCmd(cmd *cobra.Command, con *core.Console) error {
	sessionID, _ := cmd.Flags().GetString("session-id")
	pipelineID, _ := cmd.Flags().GetString("pipeline-id")
	agentID, _ := cmd.Flags().GetString("agent-id")
	intervalStr := cmd.Flags().Arg(0)

	if intervalStr == "" {
		return fmt.Errorf("interval (ms) is required as positional argument")
	}
	interval, err := strconv.ParseInt(intervalStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid interval: %w", err)
	}

	// Resolve via session-id if provided
	if sessionID != "" {
		session, ok := con.Sessions[sessionID]
		if !ok {
			return fmt.Errorf("session %s not found", sessionID)
		}
		pipelineID = session.PipelineId
		pipe, ok := con.Pipelines[pipelineID]
		if !ok {
			return fmt.Errorf("pipeline %s not found for session %s", pipelineID, sessionID)
		}
		rem := pipe.GetRem()
		if rem == nil || len(rem.Agents) == 0 {
			return fmt.Errorf("no REM agents found on pipeline %s", pipelineID)
		}
		for id := range rem.Agents {
			agentID = id
			break
		}
	}

	// Resolve pipeline from agent-id by querying PivotingContexts
	if agentID != "" && pipelineID == "" {
		ctxs, err := con.Rpc.GetContexts(con.Context(), &clientpb.Context{
			Type: consts.ContextPivoting,
		})
		if err != nil {
			return fmt.Errorf("failed to query pivot contexts: %w", err)
		}
		pivots, err := output.ToContexts[*output.PivotingContext](ctxs.Contexts)
		if err != nil {
			return fmt.Errorf("failed to parse pivot contexts: %w", err)
		}
		var matched []string
		seen := make(map[string]struct{})
		for _, p := range pivots {
			if p.RemAgentID == agentID && p.Enable {
				if _, dup := seen[p.Pipeline]; !dup {
					seen[p.Pipeline] = struct{}{}
					matched = append(matched, p.Pipeline)
				}
			}
		}
		switch len(matched) {
		case 0:
			return fmt.Errorf("agent %s not found in any active pipeline", agentID)
		case 1:
			pipelineID = matched[0]
		default:
			return fmt.Errorf("agent %s found in multiple pipelines %v, please specify --pipeline-id", agentID, matched)
		}
	}

	if pipelineID == "" || agentID == "" {
		return fmt.Errorf("either --session-id, --agent-id, or both --pipeline-id and --agent-id are required")
	}

	_, err = con.Rpc.RemAgentCtrl(con.Context(), &clientpb.REMAgent{
		PipelineId: pipelineID,
		Id:         agentID,
		Args:       []string{"reconfigure", strconv.FormatInt(interval, 10)},
	})
	if err != nil {
		return err
	}
	con.Log.Importantf("Set polling interval to %dms for agent %s on %s\n", interval, agentID, pipelineID)
	return nil
}
