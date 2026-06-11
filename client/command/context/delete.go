package context

import (
	"fmt"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
)

func DeleteContextCmd(cmd *cobra.Command, con *core.Console) error {
	contextID := cmd.Flags().Arg(0)
	if contextID == "" {
		return fmt.Errorf("context_id is required")
	}

	confirmed, err := common.Confirm(cmd, con, fmt.Sprintf("Delete context '%s'?", contextID))
	if err != nil {
		return fmt.Errorf("confirm error: %w", err)
	}
	if !confirmed {
		con.Log.Infof("Cancelled\n")
		return nil
	}

	_, err = con.Rpc.DeleteContext(con.Context(), &clientpb.Context{
		Id: contextID,
	})
	if err != nil {
		return fmt.Errorf("delete context failed: %w", err)
	}

	con.Log.Infof("Context '%s' deleted\n", contextID)
	return nil
}
