package common

import (
	"encoding/json"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/certs"
	"github.com/chainreactors/malice-network/helper/types"
	"os"
	"path/filepath"
	"strings"

	"github.com/carapace-sh/carapace"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/utils/formatutils"
	"github.com/chainreactors/malice-network/helper/utils/output"
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

		// 添加文件系统中的资源
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

func PipelineCompleter(con *repl.Console, use string) carapace.Action {
	callback := func(c carapace.Context) carapace.Action {
		results := make([]string, 0)
		for name, pipe := range con.Pipelines {
			if use == "" || pipe.Type == use {
				results = append(results, name, fmt.Sprintf("pipeline %s, type %s, listener %s", name, pipe.Type, pipe.ListenerId))
			}
		}

		return carapace.ActionValuesDescribed(results...).Tag("pipeline")
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

func BuildResourceCompleter(con *repl.Console) carapace.Action {
	callback := func(c carapace.Context) carapace.Action {
		results := make([]string, 0)
		for _, s := range consts.BuildSource {
			results = append(results, s, fmt.Sprintf("build source"))
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
			results = append(results, s.Name, fmt.Sprintf("profile %s, target %s", s.Name, s.Target))
		}
		return carapace.ActionValuesDescribed(results...).Tag("profile")
	}
	return carapace.ActionCallback(callback)
}

func ArtifactCompleter(con *repl.Console) carapace.Action {
	callback := func(c carapace.Context) carapace.Action {
		results := make([]string, 0)
		artifacts, err := con.Rpc.ListArtifact(con.Context(), &clientpb.Empty{})
		if err != nil {
			con.Log.Errorf("Error get builder: %v\n", err)
			return carapace.Action{}
		}
		for _, s := range artifacts.Artifacts {
			results = append(results, s.Name, fmt.Sprintf("id: %d, type %s, target %s", s.Id, s.Type, s.Target))
		}
		return carapace.ActionValuesDescribed(results...).Tag("artifact")
	}
	return carapace.ActionCallback(callback)
}

func ModuleArtifactsCompleter(con *repl.Console) carapace.Action {
	callback := func(c carapace.Context) carapace.Action {
		results := make([]string, 0)
		artifacts, err := con.Rpc.ListArtifact(con.Context(), &clientpb.Empty{})
		if err != nil {
			con.Log.Errorf("Error get builder: %v\n", err)
			return carapace.Action{}
		}
		for _, a := range artifacts.Artifacts {
			if a.Type == consts.CommandBuildModules {
				var params types.ProfileParams
				err = json.Unmarshal(a.ParamsBytes, &params)
				if err != nil {
					return carapace.Action{}
				}
				results = append(results, a.Name, fmt.Sprintf("target %s, module %s", a.Target, params.Modules))
			}
		}
		return carapace.ActionValuesDescribed(results...).Tag("artifact")
	}
	return carapace.ActionCallback(callback)
}

func ArtifactFormatCompleter() carapace.Action {
	// Get supported formats from formatter
	formatter := formatutils.NewFormatter()
	formatsWithDesc := formatter.GetFormatsWithDescriptions()

	// Convert to slice for carapace
	descriptions := make([]string, 0, len(formatsWithDesc)*2)
	for formatName, desc := range formatsWithDesc {
		descriptions = append(descriptions, formatName, desc)
	}

	return carapace.ActionValuesDescribed(descriptions...).Tag("artifact format")
}

func ArtifactNameCompleter(con *repl.Console) carapace.Action {
	callback := func(c carapace.Context) carapace.Action {
		results := make([]string, 0)
		artifacts, err := con.Rpc.ListArtifact(con.Context(), &clientpb.Empty{})
		if err != nil {
			con.Log.Errorf("Error get builder: %v\n", err)
			return carapace.Action{}
		}
		for _, s := range artifacts.Artifacts {
			results = append(results, s.Name, fmt.Sprintf("artifact %s, type %s, target %s", s.Name, s.Type, s.Target))
		}
		return carapace.ActionValuesDescribed(results...).Tag("artifact")
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

func HttpPipelineCompleter(con *repl.Console) carapace.Action {
	callback := func(c carapace.Context) carapace.Action {
		results := make([]string, 0)
		err := con.UpdatePipeline()
		if err != nil {
			logs.Log.Errorf("failed to get pipelines: %s", err)
			return carapace.Action{}
		}
		for _, pipeline := range con.Pipelines {
			if http := pipeline.GetHttp(); http != nil {
				results = append(results, pipeline.Name,
					fmt.Sprintf(" host: %s:%d", http.Host, http.Port))
			}
		}
		return carapace.ActionValuesDescribed(results...).Tag("http pipeline name")
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

func MalCompleter(con *repl.Console) carapace.Action {
	callback := func(c carapace.Context) carapace.Action {
		results := make([]string, 0)

		if con.MalManager == nil {
			return carapace.ActionValuesDescribed(results...).Tag("mal plugins")
		}

		// 添加外部插件
		for name, plugin := range con.MalManager.GetAllExternalPlugins() {
			manifest := plugin.Manifest()
			results = append(results, name, fmt.Sprintf("external mal: %s v%s", manifest.Name, manifest.Version))
		}

		// 添加嵌入式插件（只读）
		for name, plugin := range con.MalManager.GetAllEmbeddedPlugins() {
			manifest := plugin.Manifest()
			results = append(results, name, fmt.Sprintf("embedded mal: %s v%s (read-only)", manifest.Name, manifest.Version))
		}

		return carapace.ActionValuesDescribed(results...).Tag("mal plugins")
	}
	return carapace.ActionCallback(callback)
}

func ExternalMalCompleter(con *repl.Console) carapace.Action {
	callback := func(c carapace.Context) carapace.Action {
		results := make([]string, 0)

		if con.MalManager == nil {
			return carapace.ActionValuesDescribed(results...).Tag("external mal plugins")
		}

		// 只添加外部插件
		for name, plugin := range con.MalManager.GetAllExternalPlugins() {
			manifest := plugin.Manifest()
			results = append(results, name, fmt.Sprintf("external mal: %s v%s", manifest.Name, manifest.Version))
		}

		return carapace.ActionValuesDescribed(results...).Tag("external mal plugins")
	}
	return carapace.ActionCallback(callback)
}

func CertNameCompleter(con *repl.Console) carapace.Action {
	callback := func(c carapace.Context) carapace.Action {
		results := make([]string, 0)
		certificates, err := con.Rpc.GetAllCertificates(con.Context(), &clientpb.Empty{})
		if err != nil {
			con.Log.Errorf("Error get certs: %v\n", err)
			return carapace.Action{}
		}
		if len(certificates.Certs) < 0 {
			return carapace.Action{}
		}
		for _, c := range certificates.Certs {
			results = append(results, c.Cert.Name, fmt.Sprintf("cert %s, type %s", c.Cert.Name, c.Cert.Type))
		}
		return carapace.ActionValuesDescribed(results...).Tag("certs")
	}
	return carapace.ActionCallback(callback)
}

func CertTypeCompleter() carapace.Action {
	callback := func(c carapace.Context) carapace.Action {
		results := make([]string, 0)
		for _, c := range certs.CertTypes {
			results = append(results, c)
		}
		return carapace.ActionValuesDescribed(results...).Tag("cert type")
	}
	return carapace.ActionCallback(callback)
}
