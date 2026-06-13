package listener

import (
	"fmt"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/internal/core"
)

func dispatchForwardTaskRequest(pipelineType, pipelineName string, msg *clientpb.SpiteRequest) {
	sessionID := ""
	if msg != nil && msg.Session != nil {
		sessionID = msg.Session.SessionId
	}
	if sessionID == "" {
		logs.Log.Debugf("%s pipeline %s received task without session", pipelineType, pipelineName)
		return
	}

	if err := core.Connections.Push(sessionID, msg); err != nil {
		logs.Log.Debugf("%s pipeline %s push to %s: %s", pipelineType, pipelineName, sessionID, err)
		cancelUndeliverableForwardTask(sessionID, msg, err)
	}
}

func cancelUndeliverableForwardTask(sessionID string, msg *clientpb.SpiteRequest, dispatchErr error) {
	if msg == nil || msg.Task == nil {
		return
	}
	if core.Sessions == nil {
		logs.Log.Debugf("forward dispatch task cancel skipped for %s task=%d: sessions not initialized", sessionID, msg.Task.TaskId)
		return
	}

	session, err := core.Sessions.Get(sessionID)
	if err != nil {
		logs.Log.Debugf("forward dispatch task cancel skipped for %s task=%d: %s", sessionID, msg.Task.TaskId, err)
		return
	}

	task := session.Tasks.GetOrRecover(session, msg.Task.TaskId)
	if task == nil {
		logs.Log.Debugf("forward dispatch task cancel skipped for %s task=%d: task not found", sessionID, msg.Task.TaskId)
		return
	}

	task.CancelTask(msg.Spite, fmt.Sprintf("forward dispatch failed: %s", dispatchErr))
}
