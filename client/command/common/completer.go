package common

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/rsteube/carapace"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func LocalPathCompleter(prefix string, args []string, con *console.Console) []string {
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

func SessionIDCompleter(con *console.Console) carapace.Action {
	callback := func(c carapace.Context) carapace.Action {
		results := make([]string, 0)

		for _, s := range con.AlivedSessions() {
			results = append(results, s.SessionId, fmt.Sprintf("SessionID, %s", s.RemoteAddr))
			if s.Note != "" {
				results = append(results, s.Note, fmt.Sprintf("SessionAlias, %s", s.RemoteAddr))
			}
		}
		return carapace.ActionValuesDescribed(results...).Tag("session id")
	}
	return carapace.ActionCallback(callback)
}
