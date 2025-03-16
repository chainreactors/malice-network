package common

import (
	"fmt"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/chainreactors/mals"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/carapace-sh/carapace"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/spf13/cobra"
)

//func LocalPathCompleter(prefix string, args []string, con *repl.Console) []string {
//	var parent string
//	var partial string
//	//var sep string
//	//
//	//if runtime.GOOS == "windows" {
//	//	sep = "\\"
//	//} else {
//	//	sep = "/"
//	//}
//	fi, err := os.Stat(prefix)
//	if os.IsNotExist(err) {
//		parent = filepath.Dir(prefix)
//		partial = filepath.Base(prefix)
//	} else {
//		if fi.IsDir() {
//			parent = prefix
//			partial = ""
//		} else {
//			parent = filepath.Dir(prefix)
//			partial = filepath.Base(prefix)
//		}
//	}
//	results := []string{}
//	ls, err := ioutil.ReadDir(parent)
//	if err != nil {
//		return results
//	}
//	for _, fi = range ls {
//		if 0 < len(partial) {
//			if strings.HasPrefix(fi.Name(), partial) {
//				results = append(results, filepath.Join(parent, fi.Name()))
//			}
//		} else {
//			results = append(results, filepath.Join(parent, fi.Name()))
//		}
//	}
//	return results
//}

func SessionIDCompleter(con *repl.Console) carapace.Action {
	callback := func(c carapace.Context) carapace.Action {
		con.UpdateSessions(false)
		results := make([]string, 0)
		for _, s := range con.AlivedSessions() {
			if s.Note != "" {
				results = append(results, s.SessionId, fmt.Sprintf("SessionAlias, %s，%s", s.Note, s.Target))
			} else {
				results = append(results, s.SessionId, fmt.Sprintf("SessionID, %s", s.Target))
			}
		}
		return carapace.ActionValuesDescribed(results...).Tag("session id")
	}
	return carapace.ActionCallback(callback)
}

func ListenerIDCompleter(con *repl.Console) carapace.Action {
	callback := func(c carapace.Context) carapace.Action {
		results := make([]string, 0)

		for _, listener := range con.Listeners {
			results = append(results, listener.Id, fmt.Sprintf("ListenerID, %s", listener.Id))
		}
		return carapace.ActionValuesDescribed(results...).Tag("listener id")
	}
	return carapace.ActionCallback(callback)

}

func ListenerPipelineNameCompleter(con *repl.Console, cmd *cobra.Command) carapace.Action {
	callback := func(c carapace.Context) carapace.Action {
		results := make([]string, 0)
		listenerID := cmd.Flags().Arg(0)
		if listenerID == "" {
			return carapace.ActionValuesDescribed(results...).Tag("pipeline name")
		}
		var lis *clientpb.Listener
		for _, listener := range con.Listeners {
			if listener.Id == listenerID {
				lis = listener
				break
			}
		}
		for _, pipeline := range lis.GetPipelines().GetPipelines() {
			switch pipeline.Body.(type) {
			case *clientpb.Pipeline_Tcp:
				results = append(results, pipeline.Name, fmt.Sprintf("type tcp %s:%v",
					pipeline.GetTcp().Host, pipeline.GetTcp().Port))
			case *clientpb.Pipeline_Web:
				results = append(results, pipeline.Name, "type web")
			}
		}
		return carapace.ActionValuesDescribed(results...).Tag("pipeline name")
	}
	return carapace.ActionCallback(callback)

}

func SessionAddonCompleter(con *repl.Console) carapace.Action {
	callback := func(c carapace.Context) carapace.Action {
		results := make([]string, 0)
		for _, s := range con.GetInteractive().Addons {
			results = append(results, s.Name, "")
		}
		return carapace.ActionValuesDescribed(results...).Tag("session addons")
	}
	return carapace.ActionCallback(callback)
}

func SessionTaskCompleter(con *repl.Console) carapace.Action {
	callback := func(c carapace.Context) carapace.Action {
		results := make([]string, 0)
		for _, s := range con.GetInteractive().Tasks.Tasks {
			results = append(results, fmt.Sprintf("%d", s.TaskId), "")
		}
		return carapace.ActionValuesDescribed(results...).Tag("session tasks")
	}
	return carapace.ActionCallback(callback)
}

func ResourceCompleter(con *repl.Console) carapace.Action {
	callback := func(c carapace.Context) carapace.Action {
		results := make([]string, 0)
		err := filepath.WalkDir(assets.GetConfigDir(), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if !d.IsDir() && strings.HasSuffix(d.Name(), ".auth") {
				fileName := d.Name()
				prefix := strings.TrimSuffix(fileName, ".auth")
				results = append(results, prefix, "client auth file")
			}
			return nil
		})
		if err != nil {
			fmt.Printf("Error walking the directory: %v\n", err)
			return carapace.Action{}
		}
		return carapace.ActionValuesDescribed(results...).Tag("session resources")
	}
	return carapace.ActionCallback(callback)
}

func JobsCompleter(con *repl.Console, cmd *cobra.Command, use string) carapace.Action {
	callback := func(c carapace.Context) carapace.Action {
		results := make([]string, 0)
		listenerID := cmd.Flags().Arg(0)
		var lis *clientpb.Listener
		for _, listener := range con.Listeners {
			if listener.Id == listenerID {
				lis = listener
				break
			}
		}
		for _, pipeline := range lis.GetPipelines().Pipelines {
			switch pipeline.Body.(type) {
			case *clientpb.Pipeline_Tcp:
				if use == consts.CommandPipelineTcp {
					results = append(results, pipeline.Name,
						fmt.Sprintf("tcp job %s:%v", pipeline.GetTcp().Host, pipeline.GetTcp().Port))
				}
			case *clientpb.Pipeline_Web:
				if use == consts.CommandWebsite {
					results = append(results, pipeline.Name,
						fmt.Sprintf("web job %v, path %s", pipeline.GetWeb().Port, pipeline.GetWeb().Root))
				}
			}
		}
		return carapace.ActionValuesDescribed(results...).Tag("session jobs")
	}
	return carapace.ActionCallback(callback)
}

func BuildTargetCompleter(con *repl.Console) carapace.Action {
	callback := func(c carapace.Context) carapace.Action {
		results := make([]string, 0)
		for s, _ := range consts.BuildTargetMap {
			results = append(results, s, "")
		}
		return carapace.ActionValuesDescribed(results...).Tag("build")
	}
	return carapace.ActionCallback(callback)
}

func BuildTypeCompleter(con *repl.Console) carapace.Action {
	callback := func(c carapace.Context) carapace.Action {
		results := make([]string, 0)
		for _, s := range consts.BuildType {
			results = append(results, s, fmt.Sprintf("build type"))
		}
		return carapace.ActionValuesDescribed(results...).Tag("build")
	}
	return carapace.ActionCallback(callback)
}

func ProfileCompleter(con *repl.Console) carapace.Action {
	callback := func(c carapace.Context) carapace.Action {
		results := make([]string, 0)
		profiles, err := con.Rpc.GetProfiles(con.Context(), &clientpb.Empty{})
		if err != nil {
			con.Log.Errorf("Error get profiles: %v\n", err)
			return carapace.Action{}
		}
		for _, s := range profiles.Profiles {
			results = append(results, s.Name, fmt.Sprintf("profile %s, type %s, target %s", s.Name, s.Type, s.Target))
		}
		return carapace.ActionValuesDescribed(results...).Tag("profile")
	}
	return carapace.ActionCallback(callback)
}

func ArtifactCompleter(con *repl.Console) carapace.Action {
	callback := func(c carapace.Context) carapace.Action {
		results := make([]string, 0)
		builders, err := con.Rpc.ListBuilder(con.Context(), &clientpb.Empty{})
		if err != nil {
			con.Log.Errorf("Error get builder: %v\n", err)
			return carapace.Action{}
		}
		for _, s := range builders.Builders {
			results = append(results, strconv.Itoa(int(s.Id)), fmt.Sprintf("builder %s, type %s, target %s", s.Name, s.Type, s.Target))
		}
		return carapace.ActionValuesDescribed(results...).Tag("builder")
	}
	return carapace.ActionCallback(callback)
}

func ArtifactNameCompleter(con *repl.Console) carapace.Action {
	callback := func(c carapace.Context) carapace.Action {
		results := make([]string, 0)
		builders, err := con.Rpc.ListBuilder(con.Context(), &clientpb.Empty{})
		if err != nil {
			con.Log.Errorf("Error get builder: %v\n", err)
			return carapace.Action{}
		}
		for _, s := range builders.Builders {
			results = append(results, s.Name, fmt.Sprintf("builder %s, type %s, target %s", s.Name, s.Type, s.Target))
		}
		return carapace.ActionValuesDescribed(results...).Tag("builder")
	}
	return carapace.ActionCallback(callback)
}

func SyncCompleter(con *repl.Console) carapace.Action {
	callback := func(c carapace.Context) carapace.Action {
		results := make([]string, 0)
		ctxs, err := con.Rpc.GetContexts(con.Context(), &clientpb.Context{})
		if err != nil {
			con.Log.Errorf("Error get ctxs: %v\n", err)
			return carapace.Action{}
		}
		for _, f := range ctxs.Contexts {
			results = append(results, f.Id, fmt.Sprintf("%s %s", f.Type, f.Session.SessionId))
		}
		return carapace.ActionValuesDescribed(results...).Tag("sync")
	}
	return carapace.ActionCallback(callback)
}

func AllPipelineCompleter(con *repl.Console) carapace.Action {
	callback := func(c carapace.Context) carapace.Action {
		results := make([]string, 0)
		for _, pipeline := range con.Pipelines {
			results = append(results, pipeline.Name, fmt.Sprintf("%s: %s", pipeline.ListenerId, pipeline.Name))
		}
		return carapace.ActionValuesDescribed(results...).Tag("pipeline name")
	}
	return carapace.ActionCallback(callback)
}

func SessionModuleCompleter(con *repl.Console) carapace.Action {
	callback := func(c carapace.Context) carapace.Action {
		results := make([]string, 0)

		for _, s := range con.GetInteractive().Modules {
			results = append(results, s, "")
		}
		return carapace.ActionValuesDescribed(results...).Tag("session modules")
	}
	return carapace.ActionCallback(callback)
}

func ModulesCompleter() carapace.Action {
	callback := func(c carapace.Context) carapace.Action {
		results := make([]string, 0)
		for _, s := range consts.Modules {
			results = append(results, s, fmt.Sprintf("modules"))
		}
		return carapace.ActionValuesDescribed(results...).Tag("modules")
	}
	return carapace.ActionCallback(callback)
}

func WebsiteCompleter(con *repl.Console) carapace.Action {
	callback := func(c carapace.Context) carapace.Action {
		results := make([]string, 0)
		for _, pipeline := range con.Pipelines {
			if web := pipeline.GetWeb(); web != nil {
				results = append(results, pipeline.Name,
					fmt.Sprintf("port: %d, root: %s", web.Port, web.Root))
			}
		}
		return carapace.ActionValuesDescribed(results...).Tag("website name")
	}
	return carapace.ActionCallback(callback)
}

func WebContentCompleter(con *repl.Console) carapace.Action {
	callback := func(c carapace.Context) carapace.Action {
		results := make([]string, 0)
		con.UpdateListener()
		// List all contents from all websites since content ID is globally unique
		for _, pipeline := range con.Pipelines {
			if web := pipeline.GetWeb(); web != nil {
				for path, content := range web.Contents {
					results = append(results, content.Id,
						fmt.Sprintf("website: %s, path: %s, type: %s",
							pipeline.Name, path, content.Type))
				}
			}
		}

		return carapace.ActionValuesDescribed(results...).Tag("content id")
	}
	return carapace.ActionCallback(callback)
}

func RemPipelineCompleter(con *repl.Console) carapace.Action {
	callback := func(c carapace.Context) carapace.Action {
		results := make([]string, 0)
		for _, pipeline := range con.Pipelines {
			if rem := pipeline.GetRem(); rem != nil {
				results = append(results, pipeline.Name,
					fmt.Sprintf("console: %s", rem.Console))
			}
		}
		return carapace.ActionValuesDescribed(results...).Tag("rem pipeline name")
	}
	return carapace.ActionCallback(callback)
}

func RemAgentCompleter(con *repl.Console) carapace.Action {
	callback := func(c carapace.Context) carapace.Action {
		results := make([]string, 0)
		for _, pipeline := range con.Pipelines {
			if rem := pipeline.GetRem(); rem != nil {
				ctxs, err := con.Rpc.GetContexts(con.Context(), &clientpb.Context{
					Type: consts.ContextPivoting,
				})
				if err != nil {
					return carapace.ActionValuesDescribed(results...).Tag("rem agent name")
				}
				contexts, err := output.ToContexts[*output.PivotingContext](ctxs.Contexts)
				if err != nil {
					return carapace.ActionValuesDescribed(results...).Tag("rem agent name")
				}
				for _, ctx := range contexts {
					results = append(results, ctx.RemAgentID, ctx.String())
				}
			}
		}
		return carapace.ActionValuesDescribed(results...).Tag("rem agent")
	}
	return carapace.ActionCallback(callback)
}

func TaskTriggerTypeCompleter() carapace.Action {
	return carapace.ActionValuesDescribed(
		"Daily", "Triggers every day",
		"Monthly", "Triggers every month",
		"Weekly", "Triggers every week",
		"AtLogon", "Triggers at user logon",
		"StartUp", "Triggers at system startup",
	).Tag("task trigger type")
}

func ServiceStartTypeCompleter() carapace.Action {
	return carapace.ActionValuesDescribed(
		"BootStart", "Starts when the system starts",
		"SystemStart", "Starts when the system starts",
		"AutoStart", "Starts automatically",
		"DemandStart", "Starts on demand",
		"Disabled", "Starts disabled",
	).Tag("service start type")
}

func ServiceErrorControlCompleter() carapace.Action {
	return carapace.ActionValuesDescribed(
		"Ignore", "Ignore errors",
		"Normal", "Normal error control",
		"Severe", "Severe error control",
		"Critical", "Critical error control",
	).Tag("service error control")
}

func Register(con *repl.Console) {
	con.RegisterServerFunc("bind_args_completer", func(con *repl.Console, cmd *cobra.Command, actions []carapace.Action) (bool, error) {
		BindArgCompletions(cmd, nil, actions...)
		return true, nil
	}, &mals.Helper{Group: intermediate.GroupClient})

	con.RegisterServerFunc("bind_flags_completer", func(con *repl.Console, cmd *cobra.Command, actions map[string]carapace.Action) (bool, error) {
		BindFlagCompletions(cmd, func(comp carapace.ActionMap) {
			for k, v := range actions {
				comp[k] = v
			}
		})
		return true, nil
	}, nil)
	con.RegisterServerFunc("session_completer", intermediate.WrapFunctionReturn(SessionIDCompleter), &mals.Helper{Group: intermediate.GroupClient})
	con.RegisterServerFunc("listener_completer", intermediate.WrapFunctionReturn(ListenerIDCompleter), &mals.Helper{Group: intermediate.GroupClient})
	con.RegisterServerFunc("listener_with_pipeline_completer", intermediate.WrapFunctionReturn(ListenerPipelineNameCompleter), &mals.Helper{Group: intermediate.GroupClient})
	con.RegisterServerFunc("addon_completer", intermediate.WrapFunctionReturn(SessionAddonCompleter), &mals.Helper{Group: intermediate.GroupClient})
	con.RegisterServerFunc("module_completer", intermediate.WrapFunctionReturn(SessionModuleCompleter), &mals.Helper{Group: intermediate.GroupClient})
	con.RegisterServerFunc("task_completer", intermediate.WrapFunctionReturn(SessionTaskCompleter), &mals.Helper{Group: intermediate.GroupClient})
	con.RegisterServerFunc("resource_completer", intermediate.WrapFunctionReturn(ResourceCompleter), &mals.Helper{Group: intermediate.GroupClient})
	con.RegisterServerFunc("target_completer", intermediate.WrapFunctionReturn(BuildTargetCompleter), &mals.Helper{Group: intermediate.GroupClient})
	con.RegisterServerFunc("type_completer", intermediate.WrapFunctionReturn(BuildTypeCompleter), &mals.Helper{Group: intermediate.GroupClient})
	con.RegisterServerFunc("profile_completer", intermediate.WrapFunctionReturn(ProfileCompleter), &mals.Helper{Group: intermediate.GroupClient})
	con.RegisterServerFunc("artifact_completer", intermediate.WrapFunctionReturn(ArtifactCompleter), &mals.Helper{Group: intermediate.GroupClient})
	con.RegisterServerFunc("artifact_name_completer", intermediate.WrapFunctionReturn(ArtifactNameCompleter), &mals.Helper{Group: intermediate.GroupClient})
	con.RegisterServerFunc("sync_completer", intermediate.WrapFunctionReturn(SyncCompleter), &mals.Helper{Group: intermediate.GroupClient})
	con.RegisterServerFunc("all_pipeline_completer", intermediate.WrapFunctionReturn(AllPipelineCompleter), &mals.Helper{Group: intermediate.GroupClient})
	con.RegisterServerFunc("website_completer", intermediate.WrapFunctionReturn(WebsiteCompleter), &mals.Helper{Group: intermediate.GroupClient})
	con.RegisterServerFunc("content_completer", intermediate.WrapFunctionReturn(WebContentCompleter), &mals.Helper{Group: intermediate.GroupClient})
	con.RegisterServerFunc("rem_completer", intermediate.WrapFunctionReturn(RemPipelineCompleter), &mals.Helper{Group: intermediate.GroupClient})
	con.RegisterServerFunc("rem_agent_completer", intermediate.WrapFunctionReturn(RemAgentCompleter), &mals.Helper{Group: intermediate.GroupClient})
}
