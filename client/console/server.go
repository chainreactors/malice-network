package console

import (
	"context"
	"errors"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/tui"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"google.golang.org/grpc"
	"io"
	"sync"
	"time"
)

type Listener struct {
	*clientpb.Listener
}

type Client struct {
	*clientpb.Client
}

func InitServerStatus(conn *grpc.ClientConn) (*ServerStatus, error) {
	var err error
	s := &ServerStatus{
		Rpc:       clientrpc.NewMaliceRPCClient(conn),
		Sessions:  make(map[string]*clientpb.Session),
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
		s.Clients = append(s.Clients, &Client{client})
	}

	listeners, err := s.Rpc.GetListeners(context.Background(), &clientpb.Empty{})
	if err != nil {
		return nil, err
	}
	for _, listener := range listeners.GetListeners() {
		s.Listeners = append(s.Listeners, &Listener{listener})
	}

	err = s.UpdateSession()
	if err != nil {
		return nil, err
	}

	go s.EventHandler()

	return s, nil
}

type ServerStatus struct {
	Rpc       clientrpc.MaliceRPCClient
	Info      *clientpb.Basic
	Clients   []*Client
	Listeners []*Listener
	Sessions  map[string]*clientpb.Session
	Callbacks *sync.Map
	Alive     bool
}

func (s *ServerStatus) UpdateSession() error {
	sessions, err := s.Rpc.GetSessions(context.Background(), &clientpb.Empty{})
	if err != nil {
		return err
	}

	if len(sessions.GetSessions()) == 0 {
		return nil
	}

	for _, session := range sessions.GetSessions() {
		s.Sessions[session.SessionId] = session
	}
	return nil
}

func (s *ServerStatus) UpdateTasks(session *clientpb.Session) error {
	tasks, err := s.Rpc.GetTasks(context.Background(), session)
	if err != nil {
		return err
	}

	if len(tasks.GetTasks()) == 0 {
		return nil
	}
	s.Sessions[session.SessionId].Tasks = []*clientpb.Task{}
	for _, task := range tasks.GetTasks() {
		s.Sessions[session.SessionId].Tasks = append(s.Sessions[session.SessionId].Tasks, task)
	}
	return nil
}

func (s *ServerStatus) CancelCallback(taskId uint32) {
	s.Callbacks.Delete(taskId)
}

func (s *ServerStatus) AddCallback(taskId uint32, callback TaskCallback) {
	s.Callbacks.Store(taskId, callback)
}

func (s *ServerStatus) triggerTaskCallback(event *clientpb.Event) {
	task := event.GetTask()
	if task == nil {
		Log.Errorf(ErrNotFoundTask.Error())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if callback, ok := s.Callbacks.Load(task.TaskId); ok {
		content, err := s.Rpc.GetTaskContent(ctx, &clientpb.Task{
			TaskId:    task.TaskId,
			SessionId: task.SessionId,
		})
		if err != nil {
			Log.Errorf(err.Error())
			return
		}
		callback.(TaskCallback)(content)
		s.Callbacks.Delete(task.TaskId)
	}
}

func (s *ServerStatus) triggerTaskDone(event *clientpb.Event) {
	task := event.GetTask()
	if task == nil {
		Log.Errorf(ErrNotFoundTask.Error())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if callback, ok := s.Callbacks.Load(task.TaskId); ok {
		content, err := s.Rpc.GetTaskContent(ctx, &clientpb.Task{
			TaskId:    task.TaskId,
			SessionId: task.SessionId,
		})
		err = s.handleMaleficError(content)
		if err != nil {
			return
		}
		err = s.handleTaskError(content.GetStatus())
		callback.(TaskCallback)(content)
		if err != nil {
			return
		}
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
			Log.Infof("%s has joined the game", event.Client.Name)
		case consts.EventLeft:
			Log.Infof("%s left the game", event.Client.Name)
		case consts.EventBroadcast:
			tui.Clear()
			Log.Infof("%s broadcasted: %s  %s", event.Source, string(event.Data), event.Err)
		case consts.EventNotify:
			tui.Clear()
			Log.Importantf("%s notified: %s %s", event.Source, string(event.Data), event.Err)
		case consts.EventTaskCallback:
			Log.Debugf("task callback")
			s.triggerTaskCallback(event)
		case consts.EventTaskDone:
			s.triggerTaskDone(event)
			tui.Clear()
			Log.Debugf("task done")
		}

		//con.triggerReactions(event)
	}
}

func (s *ServerStatus) handleMaleficError(content *implantpb.Spite) error {
	if content.Error == 0 {
		return nil
	} else {
		switch content.Error {
		case consts.MaleficErrorPanic:
			Log.Errorf("Module Panic")
		case consts.MaleficErrorUnpackError:
			Log.Errorf("Module unpack error")
		case consts.MaleficErrorMissbody:
			Log.Errorf("Module miss body")
		case consts.MaleficErrorModuleError:
			Log.Errorf("Module error")
		case consts.MaleficErrorModeleNotFound:
			Log.Errorf("Module not found")
		case consts.MaleficErrorTaskError:
			Log.Errorf("Task error")
		case consts.MaleficErrorTaskNotFound:
			Log.Errorf("Task not found")
		case consts.MaleficErrorTaskOperatorNotFound:
			Log.Errorf("Task operator not found")
		}
		return errors.New("malefic error")
	}
}

func (s *ServerStatus) handleTaskError(status *implantpb.Status) error {
	if status.Status == 0 {
		return nil
	} else {
		switch status.Status {
		case consts.TaskErrorOperatorError:
			Log.Errorf("Task error: %s", status.Error)
		case consts.TaskErrorNotExpectBody:
			Log.Errorf("Task error: %s", status.Error)
		case consts.TaskErrorFieldRequired:
			Log.Errorf("Task error: %s", status.Error)
		case consts.TaskErrorFieldLengthMismatch:
			Log.Errorf("Task error: %s", status.Error)
		case consts.TaskErrorFieldInvalid:
			Log.Errorf("Task error: %s", status.Error)
		case consts.TaskError:
			Log.Errorf("Task error: %s", status.Error)
		}
		return errors.New("task error")
	}
}
