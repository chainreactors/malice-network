package cert

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/spf13/cobra"
)

func SelfSignedCmd(cmd *cobra.Command, con *repl.Console) error {
	certSubject := common.ParseSelfSignFlags(cmd)
	_, err := con.Rpc.GenerateSelfCertificate(con.Context(), &clientpb.TLS{
		CertSubject: certSubject,
		Acme:        false,
	})
	if err != nil {
		return err
	}
	return nil
}

func ImportCmd(cmd *cobra.Command, con *repl.Console) error {
	tls, err := common.ParseImportCertFlags(cmd)
	if err != nil {
		return err
	}
	_, err = con.Rpc.GenerateSelfCertificate(con.Context(), tls)
	if err != nil {
		return err
	}
	return nil
}

func AcmeCmd(cmd *cobra.Command, con *repl.Console) error {
	pipelineID, _ := cmd.Flags().GetString("pipeline")
	domain, _ := cmd.Flags().GetString("domain")
	_, err := con.Rpc.GenerateAcmeCert(con.Context(), &clientpb.TLS{
		Domain:       domain,
		Acme:         true,
		PipelineName: pipelineID,
	})
	if err != nil {
		return err
	}
	return nil
}
