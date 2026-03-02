package cert

import (
	"fmt"
	"strings"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
)

func SelfSignedCmd(cmd *cobra.Command, con *core.Console) error {
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

func ImportCmd(cmd *cobra.Command, con *core.Console) error {
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

func AcmeCmd(cmd *cobra.Command, con *core.Console) error {
	domain, _ := cmd.Flags().GetString("domain")
	provider, _ := cmd.Flags().GetString("provider")
	email, _ := cmd.Flags().GetString("email")
	caURL, _ := cmd.Flags().GetString("ca-url")
	cred, _ := cmd.Flags().GetStringToString("cred")

	con.Log.Infof("Requesting ACME certificate for %s (this may take a few minutes for DNS propagation)...\n", domain)

	_, err := con.Rpc.ObtainAcmeCert(con.Context(), &clientpb.AcmeRequest{
		Domain:      domain,
		Provider:    provider,
		Email:       email,
		CaUrl:       caURL,
		Credentials: cred,
	})
	if err != nil {
		return err
	}

	con.Log.Infof("Successfully obtained ACME certificate for %s\n", domain)
	return nil
}

func AcmeConfigCmd(cmd *cobra.Command, con *core.Console) error {
	email, _ := cmd.Flags().GetString("email")
	caURL, _ := cmd.Flags().GetString("ca-url")
	provider, _ := cmd.Flags().GetString("provider")
	cred, _ := cmd.Flags().GetStringToString("cred")

	// If no flags set, show current config
	if email == "" && caURL == "" && provider == "" && len(cred) == 0 {
		config, err := con.Rpc.GetAcmeConfig(con.Context(), &clientpb.Empty{})
		if err != nil {
			return err
		}
		printAcmeConfig(config, con)
		return nil
	}

	// Update config
	config, err := con.Rpc.GetAcmeConfig(con.Context(), &clientpb.Empty{})
	if err != nil {
		return err
	}

	// Merge: only update fields that were explicitly set
	if email != "" {
		config.Email = email
	}
	if caURL != "" {
		config.CaUrl = caURL
	}
	if provider != "" {
		config.Provider = provider
	}
	if len(cred) > 0 {
		config.Credentials = cred
	}

	_, err = con.Rpc.UpdateAcmeConfig(con.Context(), config)
	if err != nil {
		return err
	}

	con.Log.Infof("ACME config updated\n")

	// Show updated config
	updated, err := con.Rpc.GetAcmeConfig(con.Context(), &clientpb.Empty{})
	if err != nil {
		return err
	}
	printAcmeConfig(updated, con)
	return nil
}

func printAcmeConfig(config *clientpb.AcmeConfig, con *core.Console) {
	// Mask credentials for display
	maskedCreds := make([]string, 0, len(config.Credentials))
	for k, v := range config.Credentials {
		if len(v) > 8 {
			maskedCreds = append(maskedCreds, fmt.Sprintf("%s=%s...%s", k, v[:4], v[len(v)-4:]))
		} else if v != "" {
			maskedCreds = append(maskedCreds, fmt.Sprintf("%s=****", k))
		}
	}

	data := map[string]interface{}{
		"Email":       config.Email,
		"CA URL":      config.CaUrl,
		"Provider":    config.Provider,
		"Credentials": strings.Join(maskedCreds, ", "),
	}
	orderedKeys := []string{"Email", "CA URL", "Provider", "Credentials"}
	tui.RenderKVWithOptions(data, orderedKeys, tui.KVOptions{ShowHeader: true})
}
