package sessions

import (
	"fmt"

	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/malice-network/helper/encoders"
	"github.com/chainreactors/malice-network/helper/encoders/hash"
	"github.com/chainreactors/mals"
	"github.com/spf13/cobra"
)

func NewBindSessionCmd(cmd *cobra.Command, con *core.Console) error {
	name, _ := cmd.Flags().GetString("name")
	if name == "" {
		name = cmd.Flags().Arg(0)
	}
	target, _ := cmd.Flags().GetString("target")
	pipelineID, _ := cmd.Flags().GetString("pipeline")

	sess, err := NewBindSession(con, pipelineID, target, name)
	if err != nil {
		return err
	}
	con.Log.Infof("session %s created\n", sess.SessionId)
	return nil
}

func NewBindSession(con *core.Console, pipelineID string, target string, name string) (*client.Session, error) {
	pipeline, err := common.FindCachedPipeline(con, pipelineID, nil)
	if err != nil {
		return nil, fmt.Errorf("resolve bind pipeline %q: %w", pipelineID, err)
	}
	if pipeline.GetBind() == nil {
		return nil, fmt.Errorf("pipeline %q is not a bind pipeline", pipelineID)
	}
	if pipeline.GetListenerId() == "" {
		return nil, fmt.Errorf("bind pipeline %q has no listener", pipeline.GetName())
	}

	rid := cryptography.RandomBytes(4)
	sid := hash.Md5Hash(rid)
	_, err = con.Rpc.Register(con.Context(), &clientpb.RegisterSession{
		PipelineId: pipeline.GetName(),
		ListenerId: pipeline.GetListenerId(),
		RawId:      encoders.BytesToUint32(rid),
		SessionId:  sid,
		Target:     target,
		Type:       consts.ImplantMaleficBind,
		RegisterData: &implantpb.Register{
			Name:  name,
			Timer: &implantpb.Timer{},
		},
	})
	if err != nil {
		return nil, err
	}
	sess, err := con.UpdateSession(sid)
	if err != nil {
		return nil, err
	}
	_, err = con.Rpc.InitBindSession(sess.Context(), &implantpb.Init{
		Data: sess.Raw(),
	})
	if err != nil {
		return nil, err
	}
	return sess, nil
}

func RegisterNewSessionFunc(con *core.Console) {
	con.RegisterServerFunc("new_bind_session", NewBindSession, &mals.Helper{
		Short:   "new bind session",
		Example: `new_bind_session("listener-a:bind-main", "10.0.0.8:5001", "bind-01")`,
		Input: []string{
			"pipeline_id",
			"target",
			"name",
		},
		Output: []string{
			"session",
		},
	})
}
