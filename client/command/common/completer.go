package common

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/rsteube/carapace"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func LocalPathCompleter(prefix string, args []string, con *repl.Console) []string {
	var parent string
	var partial string
	//var sep string
	//
	//if runtime.GOOS == "windows" {
	//	sep = "\\"
	//} else {
	//	sep = "/"
	//}
	fi, err := os.Stat(prefix)
	if os.IsNotExist(err) {
		parent = filepath.Dir(prefix)
		partial = filepath.Base(prefix)
	} else {
		if fi.IsDir() {
			parent = prefix
			partial = ""
		} else {
			parent = filepath.Dir(prefix)
			partial = filepath.Base(prefix)
		}
	}
	results := []string{}
	ls, err := ioutil.ReadDir(parent)
	if err != nil {
		return results
	}
	for _, fi = range ls {
		if 0 < len(partial) {
			if strings.HasPrefix(fi.Name(), partial) {
				results = append(results, filepath.Join(parent, fi.Name()))
			}
		} else {
			results = append(results, filepath.Join(parent, fi.Name()))
		}
	}
	return results
}

func SessionIDCompleter(con *repl.Console) carapace.Action {
	callback := func(c carapace.Context) carapace.Action {
		results := make([]string, 0)

		for _, s := range con.AlivedSessions() {
			if s.Note != "" {
				results = append(results, s.SessionId, fmt.Sprintf("SessionAlias, %s，%s", s.Note, s.RemoteAddr))
			} else {
				results = append(results, s.SessionId, fmt.Sprintf("SessionID, %s", s.RemoteAddr))
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

func SessionModuleComplete(con *repl.Console) carapace.Action {
	callback := func(c carapace.Context) carapace.Action {
		results := make([]string, 0)

		for _, s := range con.GetInteractive().Modules {
			results = append(results, s, "")
		}
		return carapace.ActionValuesDescribed(results...).Tag("session modules")
	}
	return carapace.ActionCallback(callback)
}

func SessionAddonComplete(con *repl.Console) carapace.Action {
	callback := func(c carapace.Context) carapace.Action {
		results := make([]string, 0)
		for _, s := range con.GetInteractive().Addons.Addons {
			results = append(results, s.Name, "")
		}
		return carapace.ActionValuesDescribed(results...).Tag("session addons")
	}
	return carapace.ActionCallback(callback)
}

func SessionTaskComplete(con *repl.Console) carapace.Action {
	callback := func(c carapace.Context) carapace.Action {
		results := make([]string, 0)
		for _, s := range con.GetInteractive().Tasks.Tasks {
			results = append(results, fmt.Sprintf("%d", s.TaskId), "")
		}
		return carapace.ActionValuesDescribed(results...).Tag("session tasks")
	}
	return carapace.ActionCallback(callback)
}

func ResourceCompelete(con *repl.Console) carapace.Action {
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
