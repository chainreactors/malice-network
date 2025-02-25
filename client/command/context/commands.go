package context

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/spf13/cobra"
)

func Commands(con *repl.Console) []*cobra.Command {
	contextCmd := &cobra.Command{
		Use:   "context",
		Short: "Context management",
		Long:  "Manage different types of contexts (download, upload, credential, etc)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ListContexts(cmd, con)
		},
	}

	downloadCmd := &cobra.Command{
		Use:   "download",
		Short: "List download contexts",
		RunE: func(cmd *cobra.Command, args []string) error {
			return GetDownloadsCmd(cmd, con)
		},
	}

	uploadCmd := &cobra.Command{
		Use:   "upload",
		Short: "List upload contexts",
		RunE: func(cmd *cobra.Command, args []string) error {
			return GetUploadsCmd(cmd, con)
		},
	}

	credentialCmd := &cobra.Command{
		Use:   "credential",
		Short: "List credential contexts",
		RunE: func(cmd *cobra.Command, args []string) error {
			return GetCredentialsCmd(cmd, con)
		},
	}

	portCmd := &cobra.Command{
		Use:   "port",
		Short: "List port scan contexts",
		RunE: func(cmd *cobra.Command, args []string) error {
			return GetPortsCmd(cmd, con)
		},
	}

	screenshotCmd := &cobra.Command{
		Use:   "screenshot",
		Short: "List screenshot contexts",
		RunE: func(cmd *cobra.Command, args []string) error {
			return GetScreenshotsCmd(cmd, con)
		},
	}

	keyloggerCmd := &cobra.Command{
		Use:   "keylogger",
		Short: "List keylogger contexts",
		RunE: func(cmd *cobra.Command, args []string) error {
			return GetKeyloggersCmd(cmd, con)
		},
	}

	contextCmd.AddCommand(
		downloadCmd,
		uploadCmd,
		credentialCmd,
		portCmd,
		screenshotCmd,
		keyloggerCmd,
	)
	syncCmd := &cobra.Command{
		Use:   consts.CommandSync + " [file_id]",
		Short: "Sync file",
		Long:  "sync download file in server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return SyncCmd(cmd, con)
		},
		Example: `~~~
sync [context_id]
~~~`,
	}

	common.BindArgCompletions(syncCmd, nil,
		common.SyncFileCompleter(con))

	return []*cobra.Command{
		contextCmd,
		syncCmd,
	}
}

func Register(con *repl.Console) {
	RegisterScreenshot(con)
	RegisterKeylogger(con)
	RegisterPort(con)
	RegisterCredential(con)
	RegisterUpload(con)
	RegisterDownload(con)

	con.RegisterServerFunc("callback_context", func(con *repl.Console, sess *core.Session) (intermediate.BuiltinCallback, error) {
		nonce, err := sess.Value("nonce")
		if err != nil {
			return nil, err
		}
		typ, err := sess.Value("context")
		if err != nil {
			return nil, err
		}
		return func(content interface{}) (interface{}, error) {
			contexts, err := con.Rpc.GetContexts(sess.Context(), &clientpb.Context{
				Nonce: nonce,
			})
			if err != nil {
				return "", err
			}
			var ctxs output.Contexts
			for _, c := range contexts.Contexts {
				var ctx output.Context
				switch typ {
				case consts.ContextPort, output.GOGOPortType:
					ctx, err = output.ToContext[*output.PortContext](c)
				case "zombie", consts.ContextCredential:
					ctx, err = output.ToContext[*output.CredentialContext](c)
				}
				if err != nil {
					return nil, err
				}
				ctxs = append(ctxs, ctx)
			}

			return ctxs.String(), nil
		}, nil
	}, nil)
}
