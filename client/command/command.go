package command

import (
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/command/cert"
	"github.com/chainreactors/malice-network/client/command/exec"
	"github.com/chainreactors/malice-network/client/command/jobs"
	"github.com/chainreactors/malice-network/client/command/login"
	"github.com/chainreactors/malice-network/client/command/sessions"
	"github.com/chainreactors/malice-network/client/command/use"
	"github.com/chainreactors/malice-network/client/command/version"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/desertbit/grumble"
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

func BindCommands(con *console.Console) {

	verCmd := &grumble.Command{
		Name: "version",
		Help: "List current aliases",
		//LongHelp: help.GetHelpFor([]string{consts.AliasesStr}),
		Run: func(ctx *grumble.Context) error {
			//con.Println()
			version.VersionCmd(ctx, con)
			//con.Println()
			return nil
		},
		//HelpGroup: consts.GenericHelpGroup,
	}
	con.App.AddCommand(verCmd)

	certCmd := &grumble.Command{
		Name: "cert",
		Help: "Register cert from server",
		Flags: func(f *grumble.Flags) {
			f.String("", "host", "", "Host to register")
			f.String("u", "user", "test", "User to register")
		},
		Run: func(ctx *grumble.Context) error {
			cert.CertCmd(ctx, con)
			return nil
		},
	}
	con.App.AddCommand(certCmd)

	loginCmd := &grumble.Command{
		Name: "login",
		Help: "Login to server",
		Flags: func(f *grumble.Flags) {
			f.String("c", "config", "", "server config")
		},
		Run: func(ctx *grumble.Context) error {
			login.LoginCmd(ctx, con)
			return nil
		},
	}
	con.App.AddCommand(loginCmd)

	sessionCmd := &grumble.Command{
		Name: "sessions",
		Help: "List sessions",
		Flags: func(f *grumble.Flags) {
			f.String("i", "interact", "", "interact with a session")
			f.String("k", "kill", "", "kill the designated session")
			f.Bool("K", "kill-all", false, "kill all the sessions")
			f.Bool("C", "clean", false, "clean out any sessions marked as [DEAD]")
			f.Bool("F", "force", false, "force session action without waiting for results")

			//f.String("f", "filter", "", "filter sessions by substring")
			//f.String("e", "filter-re", "", "filter sessions by regular expression")

			f.Int("t", "timeout", assets.DefaultSettings.DefaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			sessions.SessionsCmd(ctx, con)
			return nil
		},
	}
	con.App.AddCommand(sessionCmd)

	useCmd := &grumble.Command{
		Name: "use",
		Help: "Use session",
		Args: func(a *grumble.Args) {
			a.String("sid", "session id")
		},
		Run: func(ctx *grumble.Context) error {
			use.UseSessionCmd(ctx, con)
			return nil
		},
		Completer: func(prefix string, args []string) []string {
			return use.SessionIDCompleter(con, prefix)
		},
	}
	con.App.AddCommand(useCmd)

	executeCmd := &grumble.Command{
		Name: "execute",
		Help: "Execute command",
		Flags: func(f *grumble.Flags) {
			f.Bool("T", "token", false, "execute command with current token (windows only)")
			f.Bool("o", "output", false, "capture command output")
			f.Bool("s", "save", false, "save output to a file")
			f.Bool("X", "loot", false, "save output as loot")
			f.Bool("S", "ignore-stderr", false, "don't print STDERR output")
			f.String("O", "stdout", "", "remote path to redirect STDOUT to")
			f.String("E", "stderr", "", "remote path to redirect STDERR to")
			f.String("n", "name", "", "name to assign loot (optional)")
			f.Uint("P", "ppid", 0, "parent process id (optional, Windows only)")

			f.Int("t", "timeout", assets.DefaultSettings.DefaultTimeout, "command timeout in seconds")

		},
		Args: func(a *grumble.Args) {
			a.String("command", "command to execute")
			a.StringList("arguments", "arguments to the command")
		},
		Run: func(ctx *grumble.Context) error {
			exec.ExecuteCmd(ctx, con)
			return nil
		},
	}
	con.App.AddCommand(executeCmd)

	tcpCmd := &grumble.Command{
		Name: "tcp",
		Help: "Start a TCP pipeline",
		Flags: func(f *grumble.Flags) {
			f.String("l", "lhost", "0.0.0.0", "listen host")
			f.Int("p", "lport", 0, "listen port")
		},
		Run: func(ctx *grumble.Context) error {
			jobs.TcpPipelineCmd(ctx, con)
			return nil
		},
	}
	con.App.AddCommand(tcpCmd)

}
