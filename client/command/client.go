package command

import (
	"github.com/chainreactors/malice-network/client/command/listener"
	"github.com/chainreactors/malice-network/client/command/login"
	"github.com/chainreactors/malice-network/client/command/sessions"
	"github.com/chainreactors/malice-network/client/command/use"
	"github.com/chainreactors/malice-network/client/command/version"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func LocalPathCompleter(prefix string, args []string, con *console.Console) []string {
	var parent string
	var partial string
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

func BindClientsCommands(con *console.Console) {
	bind := makeBind(con)

	bind("",
		version.Command)

	bind(consts.GenericGroup,
		login.Command,
		sessions.Command,
		use.Command,
		listener.Commands,
	)

	//certCmd := &grumble.Command{
	//	Name: "cert",
	//	Help: "Register cert from server",
	//	Flags: func(f *grumble.Flags) {
	//		f.String("", "host", "", "Host to register")
	//		f.String("u", "user", "test", "User to register")
	//		f.Int("p", "port", 40000, "Port to register")
	//	},
	//	Run: func(ctx *grumble.Context) error {
	//		cert.CertCmd(ctx, con)
	//		return nil
	//	},
	//}
	//con.App.AddCommand(certCmd)

}
