package basic

import (
	"fmt"
	"strings"
	"time"

	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/spf13/cobra"
)

func GetCmd(cmd *cobra.Command, con *core.Console) error {
	session := con.GetInteractive()
	task, err := Get(con, session)
	if err != nil {
		return err
	}
	session.Console(task, string(*con.App.Shell().Line()))
	return nil
}

func PollingCmd(cmd *cobra.Command, con *core.Console) error {
	return PollingStartCmd(cmd, con)
}

func PollingStartCmd(cmd *cobra.Command, con *core.Console) error {
	session := con.GetInteractive()
	interval, _ := cmd.Flags().GetInt("interval")
	_, err := Polling(con, session, uint64(time.Duration(interval)*time.Second), true, nil)
	if err != nil {
		return err
	}
	con.Log.Infof("polling started for session %s, interval %ds\n", session.SessionId, interval)
	return nil
}

func PollingStopCmd(cmd *cobra.Command, con *core.Console) error {
	session := con.GetInteractive()
	err := StopPolling(con, session)
	if err != nil {
		return err
	}
	con.Log.Infof("polling stopped for session %s\n", session.SessionId)
	return nil
}

func PollingStatusCmd(cmd *cobra.Command, con *core.Console) error {
	session := con.GetInteractive()
	state, err := PollingStatus(con, session)
	if err != nil {
		return err
	}
	con.Log.Infof("%s\n", FormatPollingState(state))
	return nil
}

func RecoverCmd(cmd *cobra.Command, con *core.Console) error {
	_, err := con.UpdateSession(con.GetInteractive().SessionId)
	if err != nil {
		return err
	}
	return nil
}

func InitCmd(cmd *cobra.Command, con *core.Console) error {
	_, err := Init(con, con.GetInteractive())
	if err != nil {
		return err
	}
	return nil
}

func Init(con *core.Console, sess *client.Session) (bool, error) {
	_, err := con.Rpc.InitBindSession(sess.Context(), &implantpb.Init{
		Data: sess.Raw(),
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

func Get(con *core.Console, sess *client.Session) (*clientpb.Task, error) {
	return con.Rpc.Ping(sess.Context(), &implantpb.Ping{Nonce: int32(cryptography.RandomInRange(0, 0x0fffffff))})
}

func Polling(con *core.Console, sess *client.Session, interval uint64, force bool, tasks []uint32) (bool, error) {
	u32tasks := make([]uint32, len(tasks))
	for i, task := range tasks {
		u32tasks[i] = uint32(task)
	}

	_, err := con.Rpc.Polling(sess.Context(), &clientpb.Polling{
		Id:        pollingID(sess),
		SessionId: sess.SessionId,
		Interval:  interval,
		Force:     force,
		Tasks:     u32tasks,
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

func StopPolling(con *core.Console, sess *client.Session) error {
	_, err := con.Rpc.StopPolling(sess.Context(), &clientpb.Polling{
		Id:        pollingID(sess),
		SessionId: sess.SessionId,
	})
	return err
}

func PollingStatus(con *core.Console, sess *client.Session) (*clientpb.PollingState, error) {
	return con.Rpc.PollingStatus(sess.Context(), &clientpb.Polling{
		Id:        pollingID(sess),
		SessionId: sess.SessionId,
	})
}

func pollingID(sess *client.Session) string {
	return "bind-polling:" + sess.SessionId
}

func FormatPollingState(state *clientpb.PollingState) string {
	if state == nil || state.SessionId == "" {
		return "polling stopped"
	}
	status := "stopped"
	if state.Running {
		status = "running"
	}
	parts := []string{
		fmt.Sprintf("polling %s for session %s", status, state.SessionId),
	}
	if state.Interval > 0 {
		parts = append(parts, fmt.Sprintf("interval %s", time.Duration(state.Interval)))
	}
	if state.StartedAt > 0 {
		parts = append(parts, fmt.Sprintf("started %s", time.Unix(state.StartedAt, 0).Format(time.RFC3339)))
	}
	if state.LastTickAt > 0 {
		parts = append(parts, fmt.Sprintf("last tick %s", time.Unix(state.LastTickAt, 0).Format(time.RFC3339)))
	}
	if state.LastError != "" {
		parts = append(parts, fmt.Sprintf("last error %s", state.LastError))
	}
	return strings.Join(parts, ", ")
}
