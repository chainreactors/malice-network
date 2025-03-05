package pivot

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/malice-network/helper/third/rem"
	"github.com/spf13/cobra"
	"strconv"
)

func ReverseCmd(cmd *cobra.Command, con *repl.Console) error {
	pid := cmd.Flags().Arg(0)
	port, _ := cmd.Flags().GetString("port")
	username, _ := cmd.Flags().GetString("username")
	password, _ := cmd.Flags().GetString("password")
	sess := con.GetInteractive()
	if port == "" {
		port = strconv.Itoa(int(cryptography.RandomInRange(20000, 40000)))
	}

	remoteURL := rem.NewURL("socks5", username, password, "0.0.0.0", port)
	args, err := FormatRemCmdLine(con, pid, "reverse", remoteURL, nil)
	if err != nil {
		return err
	}
	task, err := RemDial(con.Rpc, sess, pid, args)
	if err != nil {
		return err
	}
	sess.Console(task, fmt.Sprintf("pivoting socks5 on %s:%s", con.Pipelines[pid].Ip, port))

	return nil
}
