package common

import (
	"github.com/carapace-sh/carapace"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/chainreactors/mals"
	"github.com/spf13/cobra"
)

func Register(con *repl.Console) {
	con.RegisterServerFunc("bind_args_completer", func(con *repl.Console, cmd *cobra.Command, actions []carapace.Action) (bool, error) {
		BindArgCompletions(cmd, nil, actions...)
		return true, nil
	}, &mals.Helper{Group: intermediate.ClientGroup})

	con.RegisterServerFunc("bind_flags_completer", func(con *repl.Console, cmd *cobra.Command, actions map[string]carapace.Action) (bool, error) {
		BindFlagCompletions(cmd, func(comp carapace.ActionMap) {
			for k, v := range actions {
				comp[k] = v
			}
		})
		return true, nil
	}, &mals.Helper{Group: intermediate.ClientGroup})

	con.RegisterServerFunc("values_completer", func(con *repl.Console, values []string) (carapace.Action, error) {
		callback := func(c carapace.Context) carapace.Action {
			results := make([]string, 0)
			for _, v := range values {
				results = append(results, v, "")
			}
			return carapace.ActionValuesDescribed(results...).Tag("")
		}
		return carapace.ActionCallback(callback), nil
	}, &mals.Helper{Group: intermediate.ClientGroup})
	con.RegisterServerFunc("session_completer", intermediate.WrapFunctionReturn(SessionIDCompleter), &mals.Helper{Group: intermediate.ClientGroup})
	con.RegisterServerFunc("listener_completer", intermediate.WrapFunctionReturn(ListenerIDCompleter), &mals.Helper{Group: intermediate.ClientGroup})
	con.RegisterServerFunc("listener_with_pipeline_completer", intermediate.WrapFunctionReturn(ListenerPipelineNameCompleter), &mals.Helper{Group: intermediate.ClientGroup})
	con.RegisterServerFunc("addon_completer", intermediate.WrapFunctionReturn(SessionAddonCompleter), &mals.Helper{Group: intermediate.ClientGroup})
	con.RegisterServerFunc("module_completer", intermediate.WrapFunctionReturn(SessionModuleCompleter), &mals.Helper{Group: intermediate.ClientGroup})
	con.RegisterServerFunc("task_completer", intermediate.WrapFunctionReturn(SessionTaskCompleter), &mals.Helper{Group: intermediate.ClientGroup})
	con.RegisterServerFunc("resource_completer", intermediate.WrapFunctionReturn(ResourceCompleter), &mals.Helper{Group: intermediate.ClientGroup})
	con.RegisterServerFunc("target_completer", intermediate.WrapFunctionReturn(BuildTargetCompleter), &mals.Helper{Group: intermediate.ClientGroup})
	con.RegisterServerFunc("type_completer", intermediate.WrapFunctionReturn(BuildTypeCompleter), &mals.Helper{Group: intermediate.ClientGroup})
	con.RegisterServerFunc("profile_completer", intermediate.WrapFunctionReturn(ProfileCompleter), &mals.Helper{Group: intermediate.ClientGroup})
	con.RegisterServerFunc("artifact_completer", intermediate.WrapFunctionReturn(ArtifactCompleter), &mals.Helper{Group: intermediate.ClientGroup})
	con.RegisterServerFunc("artifact_name_completer", intermediate.WrapFunctionReturn(ArtifactNameCompleter), &mals.Helper{Group: intermediate.ClientGroup})
	con.RegisterServerFunc("sync_completer", intermediate.WrapFunctionReturn(SyncCompleter), &mals.Helper{Group: intermediate.ClientGroup})
	con.RegisterServerFunc("all_pipeline_completer", intermediate.WrapFunctionReturn(AllPipelineCompleter), &mals.Helper{Group: intermediate.ClientGroup})
	con.RegisterServerFunc("website_completer", intermediate.WrapFunctionReturn(WebsiteCompleter), &mals.Helper{Group: intermediate.ClientGroup})
	con.RegisterServerFunc("content_completer", intermediate.WrapFunctionReturn(WebContentCompleter), &mals.Helper{Group: intermediate.ClientGroup})
	con.RegisterServerFunc("rem_completer", intermediate.WrapFunctionReturn(RemPipelineCompleter), &mals.Helper{Group: intermediate.ClientGroup})
	con.RegisterServerFunc("rem_agent_completer", intermediate.WrapFunctionReturn(RemAgentCompleter), &mals.Helper{Group: intermediate.ClientGroup})
}
