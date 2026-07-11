package rpc

import (
	"context"
	"strings"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var revokeRetiredListenerOperator = db.RevokeOperator

func (rpc *Server) RetireListener(ctx context.Context, req *clientpb.ListenerRetire) (*clientpb.ForwardListenerStatus, error) {
	if err := requireAdminRole(ctx); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "listener retire request is nil")
	}
	listenerID := strings.TrimSpace(req.GetListenerId())
	if listenerID == "" {
		return nil, status.Error(codes.InvalidArgument, "listener_id is required")
	}
	lns, err := core.Listeners.Get(listenerID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "listener %s is not active", listenerID)
	}

	if !req.GetNoRevoke() {
		if err := revokeRetiredListenerOperator(listenerID); err != nil {
			return nil, status.Errorf(codes.Internal, "revoke listener %s failed: %v", listenerID, err)
		}
		opCache.InvalidateByName(listenerID)
	}

	retireReq := &clientpb.ListenerRetire{
		ListenerId:     listenerID,
		PurgeConfig:    req.GetPurgeConfig(),
		PurgeAuth:      req.GetPurgeAuth(),
		NoRevoke:       req.GetNoRevoke(),
		TimeoutSeconds: req.GetTimeoutSeconds(),
	}
	ctrlID := lns.PushCtrl(&clientpb.JobCtrl{
		Ctrl:   consts.CtrlListenerRetire,
		Retire: retireReq,
	})
	waitTimeout := time.Duration(req.GetTimeoutSeconds()) * time.Second
	if waitTimeout <= 0 {
		waitTimeout = core.DefaultCtrlTimeout
	}
	ctrlStatus := lns.WaitCtrlWithTimeout(ctrlID, waitTimeout)
	if err := waitForCtrlStatus("retire listener", listenerID, ctrlStatus); err != nil {
		return nil, status.Error(codes.Unavailable, err.Error())
	}

	cleanupRetiredListener(listenerID)
	return inactiveForwardListenerStatus(listenerID), nil
}

func cleanupRetiredListener(listenerID string) {
	if _, ok := getForwardListenerRuntime(listenerID); ok {
		_, _ = stopForwardListenerClient(listenerID)
		return
	}
	lns, err := core.Listeners.Get(listenerID)
	if err != nil {
		return
	}
	for _, pipe := range lns.AllPipelines() {
		deletePipelineStream(pipe.ListenerId, pipe.Name)
	}
	core.Listeners.Remove(lns)
}
