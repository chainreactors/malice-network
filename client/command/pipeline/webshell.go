package pipeline

import (
	"fmt"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
)

const webshellPipelineType = "webshell"

// ListWebShellCmd lists all webshell pipelines for a given listener.
func ListWebShellCmd(cmd *cobra.Command, con *core.Console) error {
	listenerID := cmd.Flags().Arg(0)
	pipes, err := con.Rpc.ListPipelines(con.Context(), &clientpb.Listener{
		Id: listenerID,
	})
	if err != nil {
		return err
	}

	var webshells []*clientpb.CustomPipeline
	for _, pipe := range pipes.Pipelines {
		if pipe.Type == webshellPipelineType {
			if custom := pipe.GetCustom(); custom != nil {
				webshells = append(webshells, custom)
			}
		}
	}

	if len(webshells) == 0 {
		con.Log.Warnf("No webshell pipelines found\n")
		return nil
	}

	con.Log.Console(tui.RendStructDefault(webshells) + "\n")
	return nil
}

// NewWebShellCmd registers a new webshell pipeline using the CustomPipeline mechanism.
// The actual bridge binary (webshell-bridge) connects to this pipeline externally.
func NewWebShellCmd(cmd *cobra.Command, con *core.Console) error {
	name := cmd.Flags().Arg(0)
	listenerID, _ := cmd.Flags().GetString("listener")

	if listenerID == "" {
		return fmt.Errorf("listener id is required")
	}
	if name == "" {
		name = fmt.Sprintf("webshell_%s", listenerID)
	}

	pipeline := &clientpb.Pipeline{
		Name:       name,
		ListenerId: listenerID,
		Enable:     true,
		Type:       webshellPipelineType,
		Body: &clientpb.Pipeline_Custom{
			Custom: &clientpb.CustomPipeline{
				Name:       name,
				ListenerId: listenerID,
				Host:       resolveWebShellListenerHost(con, listenerID),
			},
		},
	}

	_, err := con.Rpc.RegisterPipeline(con.Context(), pipeline)
	if err != nil {
		return webShellBridgeHint(listenerID, fmt.Errorf("register webshell pipeline %s: %w", name, err))
	}

	con.Log.Importantf("WebShell pipeline %s registered\n", name)

	_, err = con.Rpc.StartPipeline(con.Context(), &clientpb.CtrlPipeline{
		Name:       name,
		ListenerId: listenerID,
		Pipeline:   pipeline,
	})
	if err != nil {
		return webShellBridgeHint(listenerID, fmt.Errorf("start webshell pipeline %s: %w", name, err))
	}

	con.Log.Importantf("WebShell pipeline %s started\n", name)
	con.Log.Infof("The bridge should already be running for listener %s and waiting on pipeline control.\n", listenerID)
	con.Log.Infof("If the DLL is not loaded yet, the bridge will keep retrying until the rem server becomes reachable.\n")
	return nil
}

// StartWebShellCmd starts a stopped webshell pipeline.
func StartWebShellCmd(cmd *cobra.Command, con *core.Console) error {
	name := cmd.Flags().Arg(0)
	listenerID, _ := cmd.Flags().GetString("listener")
	pipeline, err := resolveWebShellPipeline(con, name, listenerID)
	if err != nil {
		return err
	}
	listenerID = pipeline.GetListenerId()
	_, err = con.Rpc.StartPipeline(con.Context(), &clientpb.CtrlPipeline{
		Name:       name,
		ListenerId: listenerID,
	})
	if err != nil {
		return webShellBridgeHint(listenerID, fmt.Errorf("start webshell pipeline %s: %w", name, err))
	}
	con.Log.Importantf("WebShell pipeline %s started\n", name)
	return nil
}

// StopWebShellCmd stops a running webshell pipeline.
func StopWebShellCmd(cmd *cobra.Command, con *core.Console) error {
	name := cmd.Flags().Arg(0)
	listenerID, _ := cmd.Flags().GetString("listener")
	pipeline, err := resolveWebShellPipeline(con, name, listenerID)
	if err != nil {
		return err
	}
	_, err = con.Rpc.StopPipeline(con.Context(), &clientpb.CtrlPipeline{
		Name:       name,
		ListenerId: pipeline.GetListenerId(),
	})
	if err != nil {
		return err
	}
	con.Log.Importantf("WebShell pipeline %s stopped\n", name)
	return nil
}

// DeleteWebShellCmd deletes a webshell pipeline.
func DeleteWebShellCmd(cmd *cobra.Command, con *core.Console) error {
	name := cmd.Flags().Arg(0)
	listenerID, _ := cmd.Flags().GetString("listener")
	pipeline, err := resolveWebShellPipeline(con, name, listenerID)
	if err != nil {
		return err
	}
	_, err = con.Rpc.DeletePipeline(con.Context(), &clientpb.CtrlPipeline{
		Name:       name,
		ListenerId: pipeline.GetListenerId(),
	})
	if err != nil {
		return err
	}
	con.Log.Importantf("WebShell pipeline %s deleted\n", name)
	return nil
}

func resolveWebShellListenerHost(con *core.Console, listenerID string) string {
	if listenerID == "" || con == nil {
		return ""
	}
	if listener, ok := con.Listeners[listenerID]; ok && listener.GetIp() != "" {
		return listener.GetIp()
	}
	listeners, err := con.Rpc.GetListeners(con.Context(), &clientpb.Empty{})
	if err != nil {
		return ""
	}
	for _, listener := range listeners.GetListeners() {
		if listener.GetId() == listenerID {
			return listener.GetIp()
		}
	}
	return ""
}

func resolveWebShellPipeline(con *core.Console, name, listenerID string) (*clientpb.Pipeline, error) {
	if name == "" {
		return nil, fmt.Errorf("webshell pipeline name is required")
	}
	if listenerID == "" {
		if pipe, ok := con.Pipelines[name]; ok {
			if pipe.GetType() != webshellPipelineType {
				return nil, fmt.Errorf("pipeline %s is type %s, not %s", name, pipe.GetType(), webshellPipelineType)
			}
			return pipe, nil
		}
	}

	pipes, err := con.Rpc.ListPipelines(con.Context(), &clientpb.Listener{Id: listenerID})
	if err != nil {
		return nil, err
	}

	var match *clientpb.Pipeline
	for _, pipe := range pipes.GetPipelines() {
		if pipe == nil || pipe.GetName() != name {
			continue
		}
		if pipe.GetType() != webshellPipelineType {
			return nil, fmt.Errorf("pipeline %s is type %s, not %s", name, pipe.GetType(), webshellPipelineType)
		}
		if match != nil && match.GetListenerId() != pipe.GetListenerId() {
			return nil, fmt.Errorf("multiple webshell pipelines named %s found, please specify --listener", name)
		}
		match = pipe
	}
	if match == nil {
		if listenerID != "" {
			return nil, fmt.Errorf("webshell pipeline %s not found on listener %s", name, listenerID)
		}
		return nil, fmt.Errorf("webshell pipeline %s not found", name)
	}
	return match, nil
}

func webShellBridgeHint(listenerID string, err error) error {
	if listenerID == "" {
		return err
	}
	return fmt.Errorf("%w; start webshell-bridge for listener %s first", err, listenerID)
}
