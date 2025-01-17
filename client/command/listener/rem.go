package listener

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/rem"
	"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
	"strconv"
)

func ListRemCmd(cmd *cobra.Command, con *repl.Console) error {
	listenerID := cmd.Flags().Arg(0)
	pipes, err := con.Rpc.ListRems(con.Context(), &clientpb.Listener{
		Id: listenerID,
	})
	if err != nil {
		return err
	}
	if len(pipes.Pipelines) == 0 {
		con.Log.Warnf("No REMs found")
		return nil
	}
	var rems []*clientpb.REM
	for _, pipe := range pipes.Pipelines {
		rems = append(rems, pipe.GetRem())
	}

	fmt.Println(tui.RendStructDefault(rems))
	return nil
}

func NewRemCmd(cmd *cobra.Command, con *repl.Console) error {
	name := cmd.Flags().Arg(0)
	listenerID, _ := cmd.Flags().GetString("listener")
	console, _ := cmd.Flags().GetString("console")

	if name == "" {
		name = fmt.Sprintf("%s_rem_%s", listenerID, console)
	}

	parse, err := rem.ParseConsole(console)
	if err != nil {
		return err
	}
	port, err := strconv.Atoi(parse.URL.Port())
	pipeline := &clientpb.Pipeline{
		Name:       name,
		ListenerId: listenerID,
		Enable:     true,
		Body: &clientpb.Pipeline_Rem{
			Rem: &clientpb.REM{
				Host:    parse.Host,
				Port:    uint32(port),
				Console: parse.String(),
			},
		},
	}

	_, err = con.Rpc.RegisterRem(con.Context(), pipeline)
	if err != nil {
		return err
	}

	_, err = con.Rpc.StartRem(con.Context(), &clientpb.CtrlPipeline{
		Name:       name,
		ListenerId: listenerID,
	})
	if err != nil {
		return err
	}

	return nil
}

func StartRemCmd(cmd *cobra.Command, con *repl.Console) error {
	name := cmd.Flags().Arg(0)
	_, err := con.Rpc.StartRem(con.Context(), &clientpb.CtrlPipeline{
		Name: name,
	})
	if err != nil {
		return err
	}
	return nil
}

func StopRemCmd(cmd *cobra.Command, con *repl.Console) error {
	name := cmd.Flags().Arg(0)
	_, err := con.Rpc.StopRem(con.Context(), &clientpb.CtrlPipeline{
		Name: name,
	})
	if err != nil {
		return err
	}
	return nil
}

func DeleteRemCmd(cmd *cobra.Command, con *repl.Console) error {
	name := cmd.Flags().Arg(0)
	_, err := con.Rpc.DeleteRem(con.Context(), &clientpb.CtrlPipeline{
		Name: name,
	})
	if err != nil {
		return err
	}
	return nil
}
