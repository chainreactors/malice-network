package completer

import (
	"github.com/chainreactors/malice-network/client/console"
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

func SessionIDCompleter(con *console.Console, prefix string) (results []string) {
	for _, s := range con.Sessions {
		if strings.HasPrefix(s.SessionId, prefix) {
			results = append(results, s.SessionId)
		}
	}
	return results
}

func BasicSessionIDCompleter(con *console.Console, prefix string) (results []string) {
	if con.ActiveTarget.Get() != nil {
		results = append(results, con.GetInteractive().SessionId)
		return results
	}
	for _, s := range con.Sessions {
		if strings.HasPrefix(s.SessionId, prefix) {
			results = append(results, s.SessionId)
		}
	}
	return results
}

func AliveSessionIDCompleter(con *console.Console) (results []string) {
	sid := con.GetInteractive().SessionId
	results = append(results, sid)
	return results
}
