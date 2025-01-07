package intermediate

import (
	"fmt"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
)

type OnEventFunc func(*clientpb.Event) (bool, error)

type EventCondition struct {
	Type        string
	Op          string
	MessageType string
	TaskId      string
	SessionId   string
	ListenerId  string
	PipelineId  string
}

func (cond *EventCondition) Match(e *clientpb.Event) bool {
	// 使用空字段表示任意匹配
	if cond.Type != "" && cond.Type != e.Type {
		return false
	}
	if cond.Op != "" && cond.Op != e.Op {
		return false
	}
	if cond.MessageType != "" && cond.MessageType != e.Task.Type {
		return false
	}
	if cond.TaskId != "" && e.Task != nil && cond.TaskId != fmt.Sprintf("%s_%d", e.Task.SessionId, e.Task.TaskId) {
		return false
	}
	if cond.SessionId != "" && e.Session != nil && cond.SessionId != e.Session.SessionId {
		return false
	}
	if cond.ListenerId != "" && e.Job != nil && e.Job.Pipeline != nil && cond.ListenerId != e.Job.Pipeline.ListenerId {
		return false
	}
	if cond.PipelineId != "" && e.Job != nil && e.Job.Pipeline != nil && cond.PipelineId != e.Job.Pipeline.Name {
		return false
	}
	return true
}
