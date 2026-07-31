package sessions

import (
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
	target, _ := cmd.Flags().GetString("target")
	pipelineID, _ := cmd.Flags().GetString("pipeline")

	sess, err := NewBindSession(con, pipelineID, target, name)
	if err != nil {
		return err
	}
	con.Log.Infof("session %s created\n", sess.SessionId)
	return nil
}

func NewBindSession(con *core.Console, PipelineID string, target string, name string) (*client.Session, error) {
	rid := cryptography.RandomBytes(4)
	sid := hash.Md5Hash(rid)
	_, err := con.Rpc.Register(con.Context(), &clientpb.RegisterSession{
		PipelineId: PipelineID,
		ListenerId: resolvePipelineListenerID(con, PipelineID),
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

// resolvePipelineListenerID looks up the listener that owns the given pipeline
// from the client's pipeline cache. Bind sessions must carry the listener id in
// RegisterSession, otherwise the server cannot route the init spite through the
// listener's stream (loadPipelineStreamForSession requires listenerID+pipelineID
// to match the key under which the listener registered its stream).
// Returns "" when the pipeline is not cached; the server then falls back to its
// legacy lookup behaviour.
func resolvePipelineListenerID(con *core.Console, pipelineID string) string {
	if pipelineID == "" {
		return ""
	}
	pipe, err := common.FindCachedPipeline(con, pipelineID, func(candidate *clientpb.Pipeline) bool {
		return candidate.GetName() == pipelineID
	})
	if err != nil || pipe == nil {
		return ""
	}
	return pipe.GetListenerId()
}

func RegisterNewSessionFunc(con *core.Console) {
	con.RegisterServerFunc("new_bind_session", NewBindSession, &mals.Helper{
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
