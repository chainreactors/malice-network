package pipeline

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/spf13/cobra"
)

func NewTcpPipelineCmd(cmd *cobra.Command, con *repl.Console) error {
	listenerID, proxy, host, port := common.ParsePipelineFlags(cmd)
	name := cmd.Flags().Arg(0)
	if port == 0 {
		port = cryptography.RandomInRange(10240, 65535)
	}
	if name == "" {
		name = fmt.Sprintf("tcp_%s_%d", listenerID, port)
	}

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
		Parser:     parser,
		CertName:   certName,
		Enable:     false,
		Body: &clientpb.Pipeline_Tcp{
			Tcp: &clientpb.TCPPipeline{
				Name:  name,
				Host:  host,
				Port:  port,
				Proxy: proxy,
			},
		},
	}
	_, err = con.Rpc.RegisterPipeline(con.Context(), pipeline)
	if err != nil {
		return err
	}

	con.Log.Importantf("TCP Pipeline %s regsiter\n", name)
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
