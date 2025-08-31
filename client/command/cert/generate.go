package cert

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/spf13/cobra"
)

func SelfSignedCmd(cmd *cobra.Command, con *repl.Console) error {
	certSubject := common.ParseSelfSignFlags(cmd)
	_, err := con.Rpc.GenerateSelfCert(con.Context(), &clientpb.Pipeline{
		Tls: &clientpb.TLS{
			CertSubject: certSubject,
			Acme:        false,
		},
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
	_, err = con.Rpc.GenerateSelfCert(con.Context(), &clientpb.Pipeline{
		Tls: tls,
	})
	if err != nil {
		return err
	}
	return nil
}

func AcmeCmd(cmd *cobra.Command, con *repl.Console) error {
	pipelineID, _ := cmd.Flags().GetString("pipeline")
	domain, _ := cmd.Flags().GetString("domain")
	_, err := con.Rpc.GenerateAcmeCert(con.Context(), &clientpb.Pipeline{
		Name: pipelineID,
		Tls: &clientpb.TLS{
			Domain: domain,
			Acme:   true,
		},
	})
	if err != nil {
		return err
	}
	return nil
}
