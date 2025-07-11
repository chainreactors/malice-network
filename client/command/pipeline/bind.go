package pipeline

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/spf13/cobra"
)

func NewBindPipelineCmd(cmd *cobra.Command, con *repl.Console) error {
	listenerID, _, _, _ := common.ParsePipelineFlags(cmd)
	if listenerID == "" {
		return fmt.Errorf("listener id is required")
	}
	name := cmd.Flags().Arg(0)

	tls, certName, err := common.ParseTLSFlags(cmd)
	if err != nil {
		return err
	}
	parser, encryption := common.ParseEncryptionFlags(cmd)
	if parser == "default" {
		parser = consts.ImplantMalefic
	}
	pipeline := &clientpb.Pipeline{
		Encryption: encryption,
		Tls:        tls,
		Name:       name,
		ListenerId: listenerID,
		CertName:   certName,
		Enable:     false,
		Parser:     parser,
		Body: &clientpb.Pipeline_Bind{
			Bind: &clientpb.BindPipeline{
				Name: name,
			},
		},
	}
	_, err = con.Rpc.RegisterPipeline(con.Context(), pipeline)
	if err != nil {
		return err
	}

	con.Log.Importantf("Bind Pipeline %s regsiter\n", name)
	_, err = con.Rpc.StartPipeline(con.Context(), &clientpb.CtrlPipeline{
		Name:       name,
		ListenerId: listenerID,
		Pipeline:   pipeline,
	})
	if err != nil {
		return err
	}
	return nil
}
