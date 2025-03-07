package pivot

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/malice-network/helper/third/rem"
	"github.com/spf13/cobra"
	"strconv"
)

func RPortForwardLocalCmd(cmd *cobra.Command, con *repl.Console) error {
	pid := cmd.Flags().Arg(0)
	aid := cmd.Flags().Arg(1)
	port, _ := cmd.Flags().GetString("port")
	if port == "" {
		port = strconv.Itoa(int(cryptography.RandomInRange(20000, 40000)))
	}
	remote, _ := cmd.Flags().GetString("remote")
	remLink, err := GetRemLink(con, pid)
	if err != nil {
		return err
	}
	localURL := rem.NewURL("port", "", "", "", port)
	return LocalRemDial(remLink, aid, localURL.String(), remote)
}

func PortForwardLocalCmd(cmd *cobra.Command, con *repl.Console) error {
	pid := cmd.Flags().Arg(0)
	aid := cmd.Flags().Arg(1)
	port, _ := cmd.Flags().GetString("port")
	if port == "" {
		port = strconv.Itoa(int(cryptography.RandomInRange(20000, 40000)))
	}
	target, _ := cmd.Flags().GetString("local")
	remLink, err := GetRemLink(con, pid)
	if err != nil {
		return err
	}
	remote := rem.NewURL("port", "", "", "", port)
	return LocalRemDial(remLink, aid, target, remote.String())
}

func LocalRemDial(remLink, agentID string, local, remote string) error {
	args := []string{"-c", remLink, "-m", "proxy", "-d", agentID, "-l", local, "-r", remote}

	remCon, err := rem.NewRemClient(remLink, args)
	if err != nil {
		return err
	}
	go func() {
		err := remCon.Run()
		if err != nil {
			return
		}
		age, err := remCon.Dial(remCon.ConsoleURL)
		if err != nil {
			logs.Log.Error(err)
			return
		}
		go remCon.Handler(age)
	}()
	return nil
}
