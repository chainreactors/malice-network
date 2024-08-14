package rpc

import (
	"context"
	"errors"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/server/internal/core"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func newGenericRequest(ctx context.Context, msg proto.Message, opts ...int) (*GenericRequest, error) {
	req := &GenericRequest{
		Message: msg,
	}
	if session, err := getSession(ctx); err == nil {
		req.Session = session
	} else {
		return nil, err
	}

	if opts == nil {
		req.Task = req.NewTask(1)
	} else {
		req.Task = req.NewTask(opts[0])
	}
	return req, nil
}

type GenericRequest struct {
	proto.Message
	Task    *core.Task
	Session *core.Session
}

func (r *GenericRequest) NewTask(total int) *core.Task {
	return r.Session.NewTask(string(proto.MessageName(r.Message).Name()), total)
}

func (r *GenericRequest) NewSpite(msg proto.Message) (*implantpb.Spite, error) {
	spite := &implantpb.Spite{
		Timeout: uint64(consts.MinTimeout.Seconds()),
		TaskId:  r.Task.Id,
		Async:   true,
	}
	var err error
	spite, err = types.BuildSpite(spite, msg)
	if err != nil {
		return nil, err
	}
	return spite, nil
}

func (r *GenericRequest) SetCallback(callback func()) {
	r.Task.Callback = callback
}

func (r *GenericRequest) HandlerAsyncResponse(ch chan *implantpb.Spite, typ types.MsgName, callbacks ...func(spite *implantpb.Spite)) {
	resp := <-ch

	err := AssertStatusAndResponse(resp, typ)
	if err != nil {
		logs.Log.Debug(err)
		r.Task.Panic(buildErrorEvent(r.Task, err), resp)
		return
	}
	r.SetCallback(func() {
		r.Session.AddMessage(resp, r.Task.Cur)
		if callbacks != nil {
			for _, callback := range callbacks {
				callback(resp)
			}
		}
	})
	r.Task.Done(core.Event{
		EventType: consts.EventTaskDone,
		Task:      r.Task,
	})
}

func AssertRequestName(req *implantpb.Request, expect types.MsgName) error {
	if req.Name != string(expect) {
		return ErrAssertFailure
	}
	return nil
}

func AssertStatus(spite *implantpb.Spite) error {
	if stat := spite.GetStatus(); stat == nil {
		return ErrMissingRequestField
	} else if stat.Status != 0 {
		return status.Error(codes.InvalidArgument, stat.Error)
	}
	return nil
}

func AssertResponse(spite *implantpb.Spite, expect types.MsgName) error {
	body := spite.GetBody()
	if body == nil && expect != types.MsgNil {
		return ErrNilResponseBody
	}

	if expect != types.MessageType(spite) {
		return ErrAssertFailure
	}
	return nil
}

func AssertStatusAndResponse(spite *implantpb.Spite, expect types.MsgName) error {
	if err := AssertStatus(spite); err != nil {
		return err
	}
	return AssertResponse(spite, expect)
}

func buildErrorEvent(task *core.Task, err error) core.Event {
	if errors.Is(err, ErrNilStatus) {
		return core.Event{
			EventType: consts.EventTaskDone,
			Task:      task,
			Err:       ErrNilStatus.Error(),
		}
	} else if errors.Is(err, ErrAssertFailure) {
		return core.Event{
			EventType: consts.EventTaskDone,
			Task:      task,
			Err:       ErrAssertFailure.Error(),
		}
	} else if errors.Is(err, ErrNilResponseBody) {
		return core.Event{
			EventType: consts.EventTaskDone,
			Task:      task,
			Err:       ErrNilResponseBody.Error(),
		}
	} else if errors.Is(err, ErrMissingRequestField) {
		return core.Event{
			EventType: consts.EventTaskDone,
			Task:      task,
			Err:       ErrMissingRequestField.Error(),
		}
	} else {
		return core.Event{
			EventType: consts.EventTaskDone,
			Task:      task,
			Err:       err.Error(),
		}
	}
}
