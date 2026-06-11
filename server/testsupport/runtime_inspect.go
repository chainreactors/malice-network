package testsupport

import (
	"fmt"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/core"
)

func (h *ControlPlaneHarness) GetRuntimeSession(sessionID string) (*clientpb.Session, error) {
	sess, err := core.Sessions.Get(sessionID)
	if err != nil {
		return nil, err
	}
	return sess.ToProtobuf(), nil
}

func (h *ControlPlaneHarness) GetRuntimeTask(sessionID string, taskID uint32) (*clientpb.Task, error) {
	sess, err := core.Sessions.Get(sessionID)
	if err != nil {
		return nil, err
	}

	task := sess.Tasks.GetOrRecover(sess, taskID)
	if task == nil {
		return nil, fmt.Errorf("runtime task %s-%d not found", sessionID, taskID)
	}
	return task.ToProtobuf(), nil
}

func (h *ControlPlaneHarness) RuntimeKeepaliveEnabled(sessionID string) (bool, error) {
	sess, err := core.Sessions.Get(sessionID)
	if err != nil {
		return false, err
	}
	return sess.IsKeepaliveEnabled(), nil
}
