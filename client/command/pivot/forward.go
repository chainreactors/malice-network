package pivot

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/malice-network/helper/rem"
	"github.com/spf13/cobra"
	"net"
	"strconv"
)

func ForwardCmd(cmd *cobra.Command, con *repl.Console) error {
	remName := cmd.Flags().Arg(0)
	port, _ := cmd.Flags().GetString("port")
	if port == "" {
		port = strconv.Itoa(int(cryptography.RandomInRange(20000, 40000)))
	}

	target, _ := cmd.Flags().GetString("target")
	sess := con.GetInteractive()
	host, tport, err := net.SplitHostPort(target)
	if err != nil {
		return err
	}
	localURL := rem.NewURL("port", "", "", host, tport)
	remoteURL := rem.NewURL("raw", "", "", "", port)
	args, err := FormatRemCmdLine(con, remName, "", remoteURL, localURL)
	if err != nil {
		return err
	}
	task, err := RemDial(con.Rpc, sess, args)
	if err != nil {
		return err
	}
	sess.Console(task, fmt.Sprintf("pivoting portforward on %s:%s", con.Pipelines[remName].Ip, port))
	return nil
}

func ReversePortForwardCmd(cmd *cobra.Command, con *repl.Console) error {
	remName := cmd.Flags().Arg(0)
	port, _ := cmd.Flags().GetString("port")
	if port == "" {
		port = strconv.Itoa(int(cryptography.RandomInRange(20000, 40000)))
	}

	remote, _ := cmd.Flags().GetString("remote")
	sess := con.GetInteractive()
	host, tport, err := net.SplitHostPort(remote)
	if err != nil {
		return err
	}
	localURL := rem.NewURL("raw", "", "", "", port)
	remoteURL := rem.NewURL("port", "", "", host, tport)
	args, err := FormatRemCmdLine(con, remName, "proxy", remoteURL, localURL)
	if err != nil {
		return err
	}
	task, err := RemDial(con.Rpc, sess, args)
	if err != nil {
		return err
	}
	sess.Console(task, fmt.Sprintf("pivoting portforward on %s:%s", con.Pipelines[remName].Ip, port))
	return nil
}
