package repl

import (
	"context"
	"errors"
	"fmt"
	"github.com/alecthomas/chroma/v2/quick"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/handler"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/chainreactors/tui"
	"google.golang.org/grpc"
	"io"
	"os"
	"sync"
	"time"
)

func InitServerStatus(conn *grpc.ClientConn) (*ServerStatus, error) {
	var err error
	s := &ServerStatus{
		Rpc:       clientrpc.NewMaliceRPCClient(conn),
		Sessions:  make(map[string]*Session),
		Alive:     true,
		Callbacks: &sync.Map{},
	}

	s.Info, err = s.Rpc.GetBasic(context.Background(), &clientpb.Empty{})
	if err != nil {
		return nil, err
	}

	clients, err := s.Rpc.GetClients(context.Background(), &clientpb.Empty{})
	if err != nil {
		return nil, err
	}
	for _, client := range clients.GetClients() {
		s.Clients = append(s.Clients, client)
	}

	listeners, err := s.Rpc.GetListeners(context.Background(), &clientpb.Empty{})
	if err != nil {
		return nil, err
	}
	for _, listener := range listeners.GetListeners() {
		s.Listeners = append(s.Listeners, listener)
	}

	err = s.UpdateSessions(true)
	if err != nil {
		return nil, err
	}

	go s.EventHandler()

	return s, nil
}

type ServerStatus struct {
	Rpc       clientrpc.MaliceRPCClient
	Info      *clientpb.Basic
	Clients   []*clientpb.Client
	Listeners []*clientpb.Listener
	Sessions  map[string]*Session
	sessions  []*clientpb.Session
	Callbacks *sync.Map
	Alive     bool
}

func (c *Console) SessionLog(sid string) *logs.Logger {
	if ob, ok := c.Observers[sid]; ok {
		return ob.Log
	} else if c.ActiveTarget.GetInteractive() != nil {
		return c.ActiveTarget.activeObserver.Log
	} else {
		return MuteLog
	}
}

func (s *ServerStatus) UpdateSessions(all bool) error {
	var sessions *clientpb.Sessions
	var err error
	if s == nil {
		return errors.New("You need login first")
	}
	if all {
		sessions, err = s.Rpc.GetSessions(context.Background(), &clientpb.Empty{})
	} else {
		sessions, err = s.Rpc.GetAlivedSessions(context.Background(), &clientpb.Empty{})
	}
	if err != nil {
		return err
	}
	s.sessions = sessions.Sessions
	newSessions := make(map[string]*Session)

	for _, session := range sessions.GetSessions() {
		newSessions[session.SessionId] = NewSession(session)
		if session.Note != "" {
			newSessions[session.Note] = NewSession(session)
		}
	}

	s.Sessions = newSessions
	return nil
}

func (s *ServerStatus) UpdateSession(sid string) (*clientpb.Session, error) {
	session, err := s.Rpc.GetSession(context.Background(), &clientpb.SessionRequest{SessionId: sid})
	if err != nil {
		return nil, err
	}

	s.Sessions[session.SessionId] = NewSession(session)
	if session.Note != "" {
		s.Sessions[session.Note] = NewSession(session)
	}
	return nil, nil
}

func (s *ServerStatus) AlivedSessions() []*clientpb.Session {
	var alivedSessions []*clientpb.Session
	for _, session := range s.sessions {
		if session.IsAlive {
			alivedSessions = append(alivedSessions, session)
		}
	}
	return alivedSessions
}

func (s *ServerStatus) UpdateTasks(session *Session) error {
	if session == nil {
		return errors.New("session is nil")
	}
	tasks, err := s.Rpc.GetTasks(context.Background(), session.Session)
	if err != nil {
		return err
	}

	session.Tasks = &clientpb.Tasks{Tasks: tasks.Tasks}
	return nil
}

func (s *ServerStatus) CancelCallback(task *clientpb.Task) {
	s.Callbacks.Delete(fmt.Sprintf("%s_%d", task.SessionId, task.TaskId))
}

func (s *ServerStatus) AddCallback(task *clientpb.Task, callback TaskCallback) {
	s.Callbacks.Store(fmt.Sprintf("%s_%d", task.SessionId, task.TaskId), callback)
}

func (s *ServerStatus) triggerTaskCallback(event *clientpb.Event) {
	task := event.GetTask()
	if task == nil {
		Log.Errorf(ErrNotFoundTask.Error())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if callback, ok := s.Callbacks.Load(fmt.Sprintf("%s_%d", task.SessionId, task.TaskId)); ok {
		spite, err := s.Rpc.GetTaskContent(ctx, event.Task)
		if err != nil {
			Log.Errorf(err.Error())
			return
		}
		callback.(TaskCallback)(spite.Spite)
	}
}

func (s *ServerStatus) triggerTaskFinish(event *clientpb.Event) {
	task := event.GetTask()
	if task == nil {
		Log.Errorf(ErrNotFoundTask.Error())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if callback, ok := s.Callbacks.Load(fmt.Sprintf("%s_%d", task.SessionId, task.TaskId)); ok {
		content, err := s.Rpc.GetTaskContent(ctx, &clientpb.Task{
			TaskId:    task.TaskId,
			SessionId: task.SessionId,
			Need:      -1,
		})
		if err != nil {
			Log.Errorf(err.Error())
			return
		}
		quick.Highlight(os.Stdout, fmt.Sprintf("session: %s task: %d index: %d\n", task.SessionId, task.TaskId, task.Cur), "go", "terminal", "monokai")

		err = handler.HandleMaleficError(content.Spite)
		if err != nil {
			Log.Errorf(err.Error())
			return
		}

		callback.(TaskCallback)(content.Spite)
		s.Callbacks.Delete(fmt.Sprintf("%s_%d", task.SessionId, task.TaskId))
	}
}

func (s *ServerStatus) EventHandler() {
	eventStream, err := s.Rpc.Events(context.Background(), &clientpb.Empty{})
	if err != nil {
		logs.Log.Warnf("Error getting event stream: %v", err)
		return
	}
	for {
		event, err := eventStream.Recv()
		if err == io.EOF || event == nil {
			return
		}

		// Trigger event based on type
		switch event.Type {

		case consts.EventJoin:
			tui.Clear()
			Log.Infof("%s has joined the game", event.Client.Name)
		case consts.EventLeft:
			tui.Clear()
			Log.Infof("%s left the game", event.Client.Name)
		case consts.EventBroadcast:
			tui.Clear()
			Log.Infof("%s broadcasted: %s  %s", event.Source, string(event.Data), event.Err)
		case consts.EventSession:
			tui.Clear()
			Log.Importantf("%s session: %s ", event.Session.SessionId, event.Message)
		case consts.EventNotify:
			tui.Clear()
			Log.Importantf("%s notified: %s %s", event.Source, string(event.Data), event.Err)
		case consts.EventTaskCallback:
			tui.Clear()
			s.triggerTaskCallback(event)
		case consts.EventTaskFinish:
			tui.Clear()
			s.triggerTaskFinish(event)
		case consts.EventPipeline:
			tui.Clear()
			if event.GetErr() != "" {
				Log.Errorf("Pipeline error: %s", event.GetErr())
				return
			}
			Log.Importantf("Pipeline: %s", event.Message)
		case consts.EventWebsite:
			tui.Clear()
			if event.GetErr() != "" {
				Log.Errorf("Website error: %s", event.GetErr())
				return
			}
			Log.Importantf("Website: %s", event.Message)
		}
		//con.triggerReactions(event)
	}
}
