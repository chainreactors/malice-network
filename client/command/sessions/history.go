package sessions

import (
	"fmt"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
	"strconv"
)

func historyCmd(cmd *cobra.Command, con *core.Console) error {
	if con.GetInteractive() == nil {
		return fmt.Errorf("No session selected")
	}

	rawLen := cmd.Flags().Arg(0)
	if rawLen == "" {
		rawLen = "10"
	}
	length, err := strconv.Atoi(rawLen)
	if err != nil {
		return err
	}
	sess := con.GetInteractive()
	contexts, err := con.Rpc.GetSessionHistory(sess.Context(), &clientpb.Int{
		Limit: int32(length),
	})
	if err != nil {
		return err
	}
	for _, context := range contexts.Contexts {
		core.HandlerTask(sess, sess.Log, context, []byte{}, consts.CalleeCMD, true)
	}
	return nil
}

// GetHistoryWithTaskID retrieves and renders history data for a specific task ID
func GetHistoryWithTaskID(con *core.Console, taskID uint32, sessionId string) (string, error) {
	if sessionId == "" {
		return "", fmt.Errorf("session_id is required")
	}

	session, ok := con.Sessions[sessionId]
	if !ok || session == nil {
		return "", fmt.Errorf("session %s not found", sessionId)
	}

	ctx := session.Context()
	taskCtx, err := con.Rpc.GetTaskContent(ctx, &clientpb.Task{
		SessionId: sessionId,
		TaskId:    taskID,
	})
	if err != nil {
		return "", err
	}

	core.HandlerTask(session, session.Log, taskCtx, []byte{}, consts.CalleeCMD, true)
	return "task rendered", nil
}
