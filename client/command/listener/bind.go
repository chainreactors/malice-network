package listener

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/spf13/cobra"
)

func NewBindPipelineCmd(cmd *cobra.Command, con *repl.Console) error {
	listenerID, _, _ := common.ParsePipelineFlags(cmd)
	if listenerID == "" {
		return fmt.Errorf("listener id is required")
	}
	name := cmd.Flags().Arg(0)
	if name == "" {
		name = fmt.Sprintf("%s-bind-%d", listenerID, cryptography.RandomInRange(0, 1000))
	}
	tls, err := common.ParseTLSFlags(cmd)
	if err != nil {
		return err
	}
	encryption := common.ParseEncryptionFlags(cmd)
	_, err = con.Rpc.RegisterPipeline(con.Context(), &clientpb.Pipeline{
		Encryption: encryption,
		Tls:        tls,
		Name:       name,
		ListenerId: listenerID,
		Enable:     false,
		Body: &clientpb.Pipeline_Bind{
			Bind: &clientpb.BindPipeline{},
		},
	})
	if err != nil {
		return err
	}

	con.Log.Importantf("Bind Pipeline %s regsiter\n", name)
	_, err = con.Rpc.StartPipeline(con.Context(), &clientpb.CtrlPipeline{
		Name:       name,
		ListenerId: listenerID,
	})
	if err != nil {
		return err
	}
	return nil
}
