package rpc

import (
	"context"
	"fmt"
	"os"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/server/internal/core"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UploadChunk receives one sequential chunk of a browser-safe file upload,
// appends it to a server-side staging file, and on the final chunk creates a
// single implant upload Task via dispatchUpload.
func (rpc *Server) UploadChunk(ctx context.Context, req *clientpb.UploadChunkRequest) (*clientpb.UploadChunkResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "missing request")
	}

	id, ok := identityFromContext(ctx)
	if !ok || id == nil {
		var err error
		id, err = extractPeerIdentity(ctx)
		if err != nil {
			// Local/insecure tests may not present TLS; use a stable synthetic key.
			id = &PeerIdentity{CommonName: "anonymous", Fingerprint: "anonymous"}
		}
	}
	identityKey := id.Fingerprint
	if identityKey == "" {
		identityKey = id.CommonName
	}
	if identityKey == "" {
		identityKey = "anonymous"
	}

	sessionID, err := getSessionID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "session_id is required")
	}
	session, err := getSession(ctx)
	if err != nil {
		return nil, err
	}

	meta := uploadImmutableMeta{
		name:      req.GetName(),
		target:    req.GetTarget(),
		priv:      req.GetPriv(),
		hidden:    req.GetHidden(),
		override:  req.GetOverride(),
		totalSize: req.GetTotalSize(),
	}

	u, result, err := globalUploadStaging.prepareAppend(
		identityKey,
		sessionID,
		req.GetUploadId(),
		meta,
		req.GetOffset(),
		req.GetData(),
	)
	if err != nil {
		return nil, err
	}
	// prepareAppend returns u locked.

	resp := &clientpb.UploadChunkResponse{
		UploadId:   req.GetUploadId(),
		NextOffset: result.nextOffset,
	}

	// Idempotent replay that already has a Task.
	if result.replayed && result.complete {
		if task, ok := u.taskPB.(*clientpb.Task); ok && task != nil {
			resp.Task = task
		}
		u.mu.Unlock()
		return resp, nil
	}

	if !result.complete {
		u.mu.Unlock()
		return resp, nil
	}

	// Final chunk: create exactly one Task.
	if u.taskPB != nil {
		if task, ok := u.taskPB.(*clientpb.Task); ok {
			resp.Task = task
		}
		u.mu.Unlock()
		return resp, nil
	}

	if err := u.markDelivering(); err != nil {
		u.mu.Unlock()
		return nil, status.Errorf(codes.FailedPrecondition, "%v", err)
	}

	stagingPath := u.stagingPath
	uploadMeta := cloneUploadMetadata(&implantpb.UploadRequest{
		Name:     u.meta.name,
		Target:   u.meta.target,
		Priv:     u.meta.priv,
		Hidden:   u.meta.hidden,
		Override: u.meta.override,
	})
	totalSize := int64(u.meta.totalSize)
	key := globalUploadStaging.key(identityKey, sessionID, req.GetUploadId())

	f, err := os.Open(stagingPath)
	if err != nil {
		u.markFailed()
		u.mu.Unlock()
		globalUploadStaging.remove(key)
		return nil, status.Errorf(codes.Internal, "open staging: %v", err)
	}

	// Ownership of f transfers to dispatchUpload (closes via closeIfCloser).
	task, err := rpc.dispatchUpload(ctx, uploadMeta, f, totalSize)
	if err != nil {
		u.markFailed()
		u.mu.Unlock()
		globalUploadStaging.remove(key)
		return nil, err
	}

	u.markDispatched(task, globalUploadStaging.now())
	u.mu.Unlock()
	resp.Task = task
	trackUploadTaskCompletion(globalUploadStaging, u, session, task)
	return resp, nil
}

func trackUploadTaskCompletion(
	m *uploadStagingManager,
	u *stagedUpload,
	session *core.Session,
	taskPB *clientpb.Task,
) {
	if m == nil || u == nil || session == nil || taskPB == nil {
		return
	}
	task := session.Tasks.Get(taskPB.TaskId)
	if task == nil {
		u.mu.Lock()
		u.markCompleted(m.now(), m.ttl)
		u.mu.Unlock()
		return
	}

	label := fmt.Sprintf("upload-staging:%s:%d", session.ID, taskPB.TaskId)
	core.GoGuarded(label, func() error {
		<-task.Ctx.Done()
		u.mu.Lock()
		if current, ok := u.taskPB.(*clientpb.Task); ok && current != nil && current.TaskId == taskPB.TaskId {
			u.markCompleted(m.now(), m.ttl)
		}
		u.mu.Unlock()
		return nil
	}, core.LogGuardedError(label))
}
