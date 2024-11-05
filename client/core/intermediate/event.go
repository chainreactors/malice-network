package intermediate

import (
	"fmt"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
)

var (
	EventMap = map[string]Event{
		"beacon_checkin": Event{Type: consts.EventSession, Op: consts.CtrlSessionCheckin},
		"beacon_initial": Event{Type: consts.EventSession, Op: consts.CtrlSessionRegister},
	}
)

func NewEvent(e *clientpb.Event) Event {
	event := Event{
		Type: e.Type,
		Op:   e.Op,
	}

	if e.Task != nil {
		event.TaskId = fmt.Sprintf("%s_%d", e.Task.SessionId, e.Task.TaskId)
		event.MessageType = e.Task.Type
	}
	if e.Session != nil {
		event.SessionId = e.Session.SessionId
	}
	if e.Job != nil {
		event.ListenerId = e.Job.Pipeline.ListenerId
		event.PipelineId = e.Job.Pipeline.Name
	}
	return event
}

type Event struct {
	Type        string
	Op          string
	MessageType string
	TaskId      string
	SessionId   string
	ListenerId  string
	PipelineId  string
}

type OnEventFunc func(*clientpb.Event) (bool, error)
