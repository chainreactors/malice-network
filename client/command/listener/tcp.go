package listener

import (
	"context"
	"fmt"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/spf13/cobra"
)

func NewTcpPipelineCmd(cmd *cobra.Command, con *repl.Console) error {
	listenerID, host, port := common.ParsePipelineFlags(cmd)
	if listenerID == "" {
		return fmt.Errorf("listener id is required")
	}
	name := cmd.Flags().Arg(0)
	if port == 0 {
		port = cryptography.RandomInRange(10240, 65535)
	}
	if name == "" {
		name = fmt.Sprintf("%s-tcp-%d", listenerID, port)
	}
	tls, err := common.ParseTLSFlags(cmd)
	if err != nil {
		return err
	}
	encryption := common.ParseEncryptionFlags(cmd)
	_, err = con.LisRpc.RegisterPipeline(context.Background(), &clientpb.Pipeline{
		Encryption: encryption,
		Tls:        tls,
		Name:       name,
		ListenerId: listenerID,
		Enable:     false,
		Body: &clientpb.Pipeline_Tcp{
			Tcp: &clientpb.TCPPipeline{
				Host: host,
				Port: port,
			},
		},
	})
	if err != nil {
		return err
	}

	con.Log.Importantf("TCP Pipeline %s regsiter\n", name)
	_, err = con.LisRpc.StartPipeline(context.Background(), &clientpb.CtrlPipeline{
		Name:       name,
		ListenerId: listenerID,
	})
	if err != nil {
		return err
	}
	return nil
}
