package core

import (
	"errors"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/commonpb"
	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"google.golang.org/grpc"
	"sync"
	"time"
)

var (
	// Sessions - Manages implant connections
	Sessions = &sessions{
		sessions: &sync.Map{},
	}

	// ErrUnknownMessageType - Returned if the implant did not understand the message for
	//                         example when the command is not supported on the platform
	ErrUnknownMessageType = errors.New("unknown message type")

	// ErrImplantSendTimeout - The implant did not respond prior to timeout deadline
	ErrImplantSendTimeout = errors.New("implant timeout")
)

func NewSession(req *lispb.RegisterSession) *Session {
	return &Session{
		ID:         req.SessionId,
		ListenerId: req.ListenerId,
		RemoteAddr: req.RemoteAddr,
		Os:         req.RegisterData.Os,
		Process:    req.RegisterData.Process,
		Timer:      req.RegisterData.Timer,
		TaskMap:    &sync.Map{},
	}
}

// Session - Represents a connection to an implant
type Session struct {
	ListenerId  string
	ID          string
	Name        string
	RemoteAddr  string
	Os          *commonpb.Os
	Process     *commonpb.Process
	Timer       *commonpb.Timer
	Filename    string
	ActiveC2    string
	ProxyURL    string
	PollTimeout int64
	Extensions  []string
	ConfigID    string
	PeerID      int64
	Locale      string
	TaskCount   uint32
	TaskMap     *sync.Map
}

func (s *Session) ToProtobuf() *clientpb.Session {
	return &clientpb.Session{
		SessionId: s.ID,
		Name:      s.Name,
		Os:        s.Os,
		Process:   s.Process,
		Timer:     s.Timer,
	}
}

func (s *Session) UpdateLastCheckin() {
	s.Timer.LastCheckin = uint64(time.Now().Unix())
}

// Request
func (s *Session) Request(msg *lispb.SpiteSession, stream grpc.ServerStream, timeout time.Duration) (uint32, error) {
	resp := make(chan *commonpb.Spite)
	taskId := s.GenTaskId()
	s.StoreTask(taskId, resp)

	if timeout == 0 {
		timeout = 60 * time.Second
	}
	var err error
	done := make(chan struct{})
	go func() {
		err = stream.SendMsg(msg)
		if err != nil {
			return
		}
		close(done)
	}()
	select {
	case <-done:
		if err != nil {
			return taskId, err
		}
		return taskId, nil
	case <-time.After(timeout):
		return taskId, ErrImplantSendTimeout
	}
}

func (s *Session) RequestAndWait(msg *lispb.SpiteSession, stream grpc.ServerStream, timeout time.Duration) (*commonpb.Spite, error) {
	taskId, err := s.Request(msg, stream, timeout)
	if err != nil {
		return nil, err
	}
	ch, _ := s.GetTask(taskId)
	defer func() {
		close(ch)
		s.DeleteTask(taskId)
	}()
	resp := <-ch
	// todo @hyc save to database
	return resp, nil
}

func (s *Session) RequestWithStream(msg *lispb.SpiteSession, stream grpc.ServerStream, timeout time.Duration) (chan *commonpb.Spite, error) {
	taskId, err := s.Request(msg, stream, timeout)
	if err != nil {
		return nil, err
	}
	task, _ := s.GetTask(taskId)
	ch := make(chan *commonpb.Spite)
	go func() {
		defer func() {
			s.DeleteTask(taskId)
			close(ch)
		}()

		for resp := range task {
			if resp.End {
				close(task)
			} else {
				// todo @hyc save to database
				ch <- resp
			}
		}
	}()
	return ch, nil
}

func (s *Session) GenTaskId() uint32 {
	s.TaskCount++
	return s.TaskCount
}

func (s *Session) StoreTask(taskId uint32, msg chan *commonpb.Spite) {
	s.TaskMap.Store(taskId, msg)
}

func (s *Session) GetTask(taskId uint32) (chan *commonpb.Spite, bool) {
	msg, ok := s.TaskMap.Load(taskId)
	if !ok {
		return nil, false
	}
	return msg.(chan *commonpb.Spite), true
}

func (s *Session) DeleteTask(taskId uint32) {
	s.TaskMap.Delete(taskId)
}

// sessions - Manages the slivers, provides atomic access
type sessions struct {
	sessions *sync.Map // map[uint32]*Session
}

// All - Return a list of all sessions
func (s *sessions) All() []*Session {
	all := []*Session{}
	s.sessions.Range(func(key, value interface{}) bool {
		all = append(all, value.(*Session))
		return true
	})
	return all
}

// Get - Get a session by ID
func (s *sessions) Get(sessionID string) *Session {
	if val, ok := s.sessions.Load(sessionID); ok {
		return val.(*Session)
	}
	return nil
}

// Add - Add a sliver to the hive (atomically)
func (s *sessions) Add(session *Session) *Session {
	s.sessions.Store(session.ID, session)
	//EventBroker.Publish(Event{
	//	EventType: consts.SessionOpenedEvent,
	//	Session:   session,
	//})
	return session
}

// Remove - Remove a sliver from the hive (atomically)
func (s *sessions) Remove(sessionID string) {
	val, ok := s.sessions.Load(sessionID)
	if !ok {
		return
	}
	parentSession := val.(*Session)
	//children := findAllChildrenByPeerID(parentSession.PeerID)
	s.sessions.Delete(parentSession.ID)
	//coreLog.Debugf("Removing %d children of session %d (%v)", len(children), parentSession.ID, children)
	//for _, child := range children {
	//	childSession, ok := s.sessions.LoadAndDelete(child.SessionID)
	//	if ok {
	//		PivotSessions.Delete(childSession.(*Session).Connection.ID)
	//		EventBroker.Publish(Event{
	//			EventType: consts.SessionClosedEvent,
	//			Session:   childSession.(*Session),
	//		})
	//	}
	//}

	//Remove the parent session
	//EventBroker.Publish(Event{
	//	EventType: consts.SessionClosedEvent,
	//	Session:   parentSession,
	//})
}
