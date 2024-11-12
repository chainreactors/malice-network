package sessions

import (
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/core/intermediate"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/malice-network/helper/encoders"
	"github.com/chainreactors/malice-network/helper/encoders/hash"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/spf13/cobra"
)

func NewBindSessionCmd(cmd *cobra.Command, con *repl.Console) error {
	name, _ := cmd.Flags().GetString("name")
	target, _ := cmd.Flags().GetString("target")
	pipelineID, _ := cmd.Flags().GetString("pipeline")

	sess, err := NewBindSession(con, pipelineID, target, name)
	if err != nil {
		return err
	}
	con.Log.Infof("session %s created\n", sess.SessionId)
	return nil
}

func NewBindSession(con *repl.Console, PipelineID string, target string, name string) (*core.Session, error) {
	rid := cryptography.RandomBytes(4)
	sid := hash.Md5Hash(rid)
	_, err := con.Rpc.Register(con.Context(), &clientpb.RegisterSession{
		PipelineId: PipelineID,
		RawId:      encoders.BytesToUint32(rid),
		SessionId:  sid,
		Target:     target,
		Type:       consts.ImplantTypeBind,
		RegisterData: &implantpb.Register{
			Name: name,
		},
	})
	if err != nil {
		return nil, err
	}
	sess, err := con.UpdateSession(sid)
	if err != nil {
		return nil, err
	}
	_, err = con.Rpc.InitBindSession(sess.Context(), &implantpb.Request{
		Name: consts.ModuleInit,
	})
	if err != nil {
		return nil, err
	}
	return sess, nil
}

func RegisterNewSessionFunc(con *repl.Console) {
	con.RegisterServerFunc("new_bind_session", NewBindSession, &intermediate.InternalHelper{
		Short:   "new bind session",
		Example: `new_bind_session("listener_id", "target", "name")`,
		Input: []string{
			"listener_id",
			"target",
			"name",
		},
		Output: []string{
			"session",
		},
	})
}
