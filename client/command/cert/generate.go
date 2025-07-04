package cert

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/spf13/cobra"
)

func SelfSignedCmd(cmd *cobra.Command, con *repl.Console) error {
	certSubject := common.ParseSelfSignFlags(cmd)
	cert, err := con.Rpc.GenerateSelfCertificate(con.Context(), &clientpb.TLS{
		CertSubject: certSubject,
		AutoCert:    false,
	})
	if err != nil {
		return err
	}
	con.Log.Infof("cert %s %s add success\n", cert.Name, cert.Type)
	return nil
}

func ImportCmd(cmd *cobra.Command, con *repl.Console) error {
	tls, err := common.ParseImportCertFlags(cmd)
	if err != nil {
		return err
	}
	cert, err := con.Rpc.GenerateSelfCertificate(con.Context(), tls)
	if err != nil {
		return err
	}
	con.Log.Infof("cert %s %s add success\n", cert.Name, cert.Type)
	return nil
}

func AcmeCmd(cmd *cobra.Command, con *repl.Console) error {
	pipelineID, _ := cmd.Flags().GetString("pipeline")
	domain, _ := cmd.Flags().GetString("domain")
	_, err := con.Rpc.GenerateAcmeCert(con.Context(), &clientpb.TLS{
		Domain:       domain,
		AutoCert:     true,
		PipelineName: pipelineID,
	})
	if err != nil {
		return err
	}
	return nil
}
