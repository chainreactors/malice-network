package common

import (
	"context"
	"fmt"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
				if use == consts.CommandTcp {
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
			results = append(results, s, fmt.Sprintf("build target"))
		}
		return carapace.ActionValuesDescribed(results...).Tag("build")
	}
	return carapace.ActionCallback(callback)
}

//func BuildFormatCompleter(con *repl.Console) carapace.Action {
//	callback := func(c carapace.Context) carapace.Action {
//		results := make([]string, 0)
//		for _, s := range consts.BuildFormats {
//			results = append(results, s, fmt.Sprintf("build type"))
//		}
//		return carapace.ActionValuesDescribed(results...).Tag("build")
//	}
//	return carapace.ActionCallback(callback)
//}

func ProfileCompleter(con *repl.Console) carapace.Action {
	callback := func(c carapace.Context) carapace.Action {
		results := make([]string, 0)
		profiles, err := con.Rpc.GetProfiles(context.Background(), &clientpb.Empty{})
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
		builders, err := con.Rpc.ListArtifact(context.Background(), &clientpb.Empty{})
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
		builders, err := con.Rpc.ListArtifact(context.Background(), &clientpb.Empty{})
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

func SyncFileCompleter(con *repl.Console) carapace.Action {
	callback := func(c carapace.Context) carapace.Action {
		results := make([]string, 0)
		files, err := con.Rpc.GetTaskFiles(context.Background(),
			&clientpb.Session{SessionId: con.GetInteractive().SessionId})
		if err != nil {
			con.Log.Errorf("Error get files: %v\n", err)
			return carapace.Action{}
		}
		for _, f := range files.Files {
			results = append(results, f.TaskId, fmt.Sprintf("sync file type %s, remote %s, local %s ", f.Op, f.Remote, f.Local))
		}
		return carapace.ActionValuesDescribed(results...).Tag("sync")
	}
	return carapace.ActionCallback(callback)
}

func AllPipelineCompleter(con *repl.Console) carapace.Action {
	callback := func(c carapace.Context) carapace.Action {
		results := make([]string, 0)
		for _, listener := range con.Listeners {
			for _, pipeline := range listener.GetPipelines().GetPipelines() {
				results = append(results, pipeline.Name, fmt.Sprintf("%s: %s", pipeline.ListenerId, pipeline.Name))
			}
		}
		return carapace.ActionValuesDescribed(results...).Tag("pipeline name")
	}
	return carapace.ActionCallback(callback)
}
