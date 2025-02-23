package context

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/spf13/cobra"
)

func ListContexts(cmd *cobra.Command, con *repl.Console) error {
	contexts, err := con.Rpc.GetContexts(con.Context(), &clientpb.Context{})
	if err != nil {
		return err
	}

	// 格式化输出所有contexts
	for _, ctx := range contexts.GetContexts() {
		fmt.Printf("[%s] %s\n", ctx.Type, ctx.Value)
	}
	return nil
}

func GetContextsByType(con *repl.Console, contextType string) (*clientpb.Contexts, error) {
	allContexts, err := con.Rpc.GetContexts(con.Context(), &clientpb.Context{
		Type: contextType,
	})
	if err != nil {
		return nil, err
	}

	return allContexts, nil
}

func GetContextsByTask(con *repl.Console, contextType string, task *clientpb.Task) (*clientpb.Contexts, error) {
	allContexts, err := con.Rpc.GetContexts(con.Context(), &clientpb.Context{
		Task: task,
	})
	if err != nil {
		return nil, err
	}

	return allContexts, nil
}
