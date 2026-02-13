package context

import (
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/spf13/cobra"
)

func Commands(con *core.Console) []*cobra.Command {
	contextCmd := &cobra.Command{
		Use:   "context",
		Short: "Context management",
		Long:  "Manage different types of contexts (download, upload, credential, etc)",
		Annotations: map[string]string{
			"resource": "true",
		},
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

	mediaCmd := &cobra.Command{
		Use:   "media",
		Short: "List media contexts",
		RunE: func(cmd *cobra.Command, args []string) error {
			return GetMediaCmd(cmd, con)
		},
	}

	deleteCmd := &cobra.Command{
		Use:   "delete [context_id]",
		Short: "Delete a context",
		Long:  "Delete a context and its associated files from the server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return DeleteContextCmd(cmd, con)
		},
		Example: `~~~
context delete [context_id]
context delete [context_id] --yes
~~~`,
	}
	deleteCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")
	common.BindArgCompletions(deleteCmd, nil,
		common.SyncCompleter(con))

	contextCmd.AddCommand(
		downloadCmd,
		uploadCmd,
		credentialCmd,
		portCmd,
		screenshotCmd,
		keyloggerCmd,
		mediaCmd,
		deleteCmd,
	)
	syncCmd := &cobra.Command{
		Use:   consts.CommandSync + " [context_id]",
		Short: "Sync context",
		Long:  "sync context from server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return SyncCmd(cmd, con)
		},
		Example: `~~~
sync [context_id]
~~~`,
	}

	common.BindArgCompletions(syncCmd, nil,
		common.SyncCompleter(con))

	return []*cobra.Command{
		contextCmd,
		syncCmd,
	}
}

func Register(con *core.Console) {
	RegisterScreenshot(con)
	RegisterKeylogger(con)
	RegisterPort(con)
	RegisterCredential(con)
	RegisterUpload(con)
	RegisterDownload(con)
	RegisterMedia(con)

	con.RegisterServerFunc("callback_context", func(con *core.Console, sess *client.Session) (intermediate.BuiltinCallback, error) {
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
				case "zombie", "mimikatz", consts.ContextCredential:
					ctx, err = output.ToContext[*output.CredentialContext](c)
				case consts.ContextKeyLogger:
					ctx, err = output.ToContext[*output.KeyLoggerContext](c)
				case consts.ContextMedia:
					ctx, err = output.ToContext[*output.MediaContext](c)
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
