package listener

import (
	"errors"
	"fmt"

	"github.com/carapace-sh/carapace"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func newPipelineCertificateCommand(con *core.Console) *cobra.Command {
	certCmd := &cobra.Command{
		Use:   "cert",
		Short: "Manage a pipeline TLS certificate",
		Long:  "Bind, generate, inspect, or remove the stored TLS certificate association for an HTTP or TCP pipeline.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		Example: `~~~
pipeline cert show --pipeline http-main --listener edge-a
pipeline cert bind --pipeline http-main --listener edge-a --cert-name prod-cert
~~~`,
	}

	bindCmd := &cobra.Command{
		Use:   "bind",
		Short: "Bind or rebind a stored certificate",
		Long:  "Bind a stored certificate to an HTTP or TCP pipeline. A running pipeline is restarted so the new certificate takes effect; rebinding the same name forces a reload.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return PipelineCertBindCmd(cmd, con)
		},
		Example: `~~~
pipeline cert bind --pipeline http-main --listener edge-a --cert-name prod-cert
~~~`,
	}
	bindPipelineCertificateFlags(bindCmd, con, true)

	unbindCmd := &cobra.Command{
		Use:   "unbind",
		Short: "Disable TLS for a pipeline",
		Long:  "Remove the certificate association and disable TLS for an HTTP or TCP pipeline. A running pipeline is restarted with the updated configuration.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return PipelineCertUnbindCmd(cmd, con)
		},
		Example: `~~~
pipeline cert unbind --pipeline http-main --listener edge-a
~~~`,
	}
	bindPipelineCertificateFlags(unbindCmd, con, false)

	generateCmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate and bind a self-signed certificate",
		Long:  "Generate a self-signed certificate and bind it to an HTTP or TCP pipeline. Use --save-as to retain the generated certificate in the certificate store.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return PipelineCertGenerateCmd(cmd, con)
		},
		Example: `~~~
pipeline cert generate --pipeline http-main --listener edge-a

pipeline cert generate --pipeline http-main --listener edge-a --save-as http-main-cert
~~~`,
	}
	bindPipelineCertificateFlags(generateCmd, con, false)
	common.BindFlag(generateCmd, func(flags *pflag.FlagSet) {
		flags.String("save-as", "", "save the generated certificate under this name")
		flags.String("comment", "", "comment for the saved certificate")
	})

	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show a pipeline certificate binding",
		Long:  "Show the current TLS state and stored certificate association for an HTTP or TCP pipeline.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return PipelineCertShowCmd(cmd, con)
		},
		Example: `~~~
pipeline cert show --pipeline http-main --listener edge-a
~~~`,
	}
	bindPipelineCertificateFlags(showCmd, con, false)

	certCmd.AddCommand(bindCmd, unbindCmd, generateCmd, showCmd)
	return certCmd
}

func bindPipelineCertificateFlags(cmd *cobra.Command, con *core.Console, withCertName bool) {
	common.BindFlag(cmd, func(flags *pflag.FlagSet) {
		flags.String("pipeline", "", "pipeline name")
		flags.String("listener", "", "listener ID used to disambiguate the pipeline")
		if withCertName {
			flags.String("cert-name", "", "stored certificate name")
		}
	})
	_ = cmd.MarkFlagRequired("pipeline")
	if withCertName {
		_ = cmd.MarkFlagRequired("cert-name")
	}
	common.BindFlagCompletions(cmd, func(comp carapace.ActionMap) {
		comp["pipeline"] = common.PipelineNameFlagCompleter(con, cmd, consts.HTTPPipeline, consts.TCPPipeline)
		comp["listener"] = common.ListenerIDCompleter(con)
		if withCertName {
			comp["cert-name"] = common.CertNameCompleter(con)
		}
	})
}

func PipelineCertBindCmd(cmd *cobra.Command, con *core.Console) error {
	name, listenerID := pipelineCertificateTarget(cmd)
	certName, _ := cmd.Flags().GetString("cert-name")
	updated, err := con.Rpc.UpdatePipelineTLS(con.Context(), &clientpb.PipelineTLSUpdate{
		Name:       name,
		ListenerId: listenerID,
		Mode:       clientpb.TLSUpdateMode_TLS_UPDATE_MODE_EXISTING_CERT,
		CertName:   certName,
	})
	if err != nil {
		return err
	}
	printPipelineDetail(updated)
	return nil
}

func PipelineCertUnbindCmd(cmd *cobra.Command, con *core.Console) error {
	name, listenerID := pipelineCertificateTarget(cmd)
	updated, err := con.Rpc.UpdatePipelineTLS(con.Context(), &clientpb.PipelineTLSUpdate{
		Name:       name,
		ListenerId: listenerID,
		Mode:       clientpb.TLSUpdateMode_TLS_UPDATE_MODE_DISABLE,
	})
	if err != nil {
		return err
	}
	printPipelineDetail(updated)
	return nil
}

func PipelineCertGenerateCmd(cmd *cobra.Command, con *core.Console) error {
	name, listenerID := pipelineCertificateTarget(cmd)
	saveName, _ := cmd.Flags().GetString("save-as")
	comment, _ := cmd.Flags().GetString("comment")
	updated, err := con.Rpc.UpdatePipelineTLS(con.Context(), &clientpb.PipelineTLSUpdate{
		Name:         name,
		ListenerId:   listenerID,
		Mode:         clientpb.TLSUpdateMode_TLS_UPDATE_MODE_INLINE_CERT,
		Tls:          &clientpb.TLS{Enable: true},
		SaveCert:     saveName != "",
		SaveCertName: saveName,
		CertComment:  comment,
	})
	if err != nil {
		return err
	}
	printPipelineDetail(updated)
	return nil
}

func PipelineCertShowCmd(cmd *cobra.Command, con *core.Console) error {
	name, listenerID := pipelineCertificateTarget(cmd)
	pipelines, err := con.Rpc.ListPipelines(con.Context(), &clientpb.Listener{Id: listenerID})
	if err != nil {
		return err
	}
	var matches []*clientpb.Pipeline
	for _, pipeline := range pipelines.GetPipelines() {
		if pipeline.GetName() == name && (listenerID == "" || pipeline.GetListenerId() == listenerID) {
			matches = append(matches, pipeline)
		}
	}
	switch len(matches) {
	case 0:
		return errors.New("pipeline not found")
	case 1:
		printPipelineDetail(matches[0])
		return nil
	default:
		return fmt.Errorf("multiple pipelines named %q found; specify --listener", name)
	}
}

func pipelineCertificateTarget(cmd *cobra.Command) (string, string) {
	name, _ := cmd.Flags().GetString("pipeline")
	listenerID, _ := cmd.Flags().GetString("listener")
	return name, listenerID
}
