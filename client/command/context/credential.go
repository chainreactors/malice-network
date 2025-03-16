package context

import (
	"fmt"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"

	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/tui"
)

func GetCredentialsCmd(cmd *cobra.Command, con *repl.Console) error {
	credentials, err := GetCredentials(con)
	if err != nil {
		return err
	}

	var rowEntries []table.Row
	for _, ctx := range credentials {
		cred, err := output.ToContext[*output.CredentialContext](ctx)
		if err != nil {
			return err
		}

		row := table.NewRow(table.RowData{
			"ID":       ctx.Id,
			"Session":  ctx.Session.SessionId,
			"Task":     getTaskId(ctx.Task),
			"Type":     cred.CredentialType,
			"Username": cred.Params["username"],
			"Password": cred.Params["password"],
		})
		rowEntries = append(rowEntries, row)
	}

	tableModel := tui.NewTable([]table.Column{
		table.NewColumn("ID", "ID", 36),
		table.NewColumn("Session", "Session", 16),
		table.NewColumn("Task", "Task", 8),
		table.NewColumn("Type", "Type", 12),
		table.NewColumn("Username", "Username", 20),
		table.NewColumn("Password", "Password", 20),
	}, true)

	tableModel.SetRows(rowEntries)
	fmt.Printf(tableModel.View())
	return nil
}

func GetCredentials(con *repl.Console) ([]*clientpb.Context, error) {
	contexts, err := GetContextsByType(con, consts.ContextCredential)
	if err != nil {
		return nil, err
	}
	return contexts.Contexts, nil
}

func AddCredential(con *repl.Console, sess *core.Session, task *clientpb.Task, credType string, params map[string]string) (bool, error) {
	_, err := con.Rpc.AddCredential(con.Context(), &clientpb.Context{
		Session: sess.Session,
		Task:    task,
		Type:    consts.ContextCredential,
		Value:   output.MarshalContext(&output.CredentialContext{CredentialType: credType, Params: params}),
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

func RegisterCredential(con *repl.Console) {
	con.RegisterServerFunc("credentials", func(con *repl.Console) ([]*output.CredentialContext, error) {
		ctxs, err := GetCredentials(con)
		if err != nil {
			return nil, err
		}
		return output.ToContexts[*output.CredentialContext](ctxs)
	}, nil)
	con.RegisterServerFunc("add_credential", AddCredential, nil)
}
