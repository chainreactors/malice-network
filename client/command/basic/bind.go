package basic

import (
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/malice-network/helper/encoders/hash"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/spf13/cobra"
	"strconv"
	"time"
)

func GetCmd(cmd *cobra.Command, con *repl.Console) error {
	session := con.GetInteractive()
	_, err := Get(con, session)
	if err != nil {
		return err
	}
	return nil
}

func WaitCmd(cmd *cobra.Command, con *repl.Console) error {
	session := con.GetInteractive()
	interval, _ := cmd.Flags().GetInt("interval")
	taskList := cmd.Flags().Args()
	var tasks []uint32
	for _, task := range taskList {
		t, err := strconv.Atoi(task)
		if err != nil {
			return err
		}
		tasks = append(tasks, uint32(t))
	}
	_, err := Polling(con, session, uint64(time.Duration(interval)*time.Second), false, tasks)
	if err != nil {
		return err
	}
	return nil
}

func PollingCmd(cmd *cobra.Command, con *repl.Console) error {
	session := con.GetInteractive()
	interval, _ := cmd.Flags().GetInt("interval")
	_, err := Polling(con, session, uint64(time.Duration(interval)*time.Second), true, nil)
	if err != nil {
		return err
	}
	return nil
}

func RecoverCmd(cmd *cobra.Command, con *repl.Console) error {
	_, err := con.UpdateSession(con.GetInteractive().SessionId)
	if err != nil {
		return err
	}
	return nil
}

func InitCmd(cmd *cobra.Command, con *repl.Console) error {
	_, err := Init(con, con.GetInteractive())
	if err != nil {
		return err
	}
	return nil
}

func Init(con *repl.Console, sess *core.Session) (bool, error) {
	_, err := con.Rpc.InitBindSession(sess.Context(), &implantpb.Request{
		Name: consts.ModuleInit,
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

func Get(con *repl.Console, sess *core.Session) (*clientpb.Task, error) {
	return con.Rpc.Ping(sess.Context(), &implantpb.Ping{Nonce: int32(cryptography.RandomInRange(0, 0x0fffffff))})
}

func Polling(con *repl.Console, sess *core.Session, interval uint64, force bool, tasks []uint32) (bool, error) {
	u32tasks := make([]uint32, len(tasks))
	for i, task := range tasks {
		u32tasks[i] = uint32(task)
	}

	_, err := con.Rpc.Polling(sess.Context(), &clientpb.Polling{
		Id:        hash.Md5Hash(cryptography.RandomBytes(8)),
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
