package core

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/errs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/server/internal/db"
	"sync"
)

type contexts struct {
	sync.Map
}

var Contexts *contexts

type Context struct {
	ID       string
	Session  *Session
	Pipeline *clientpb.Pipeline
	Listener *Listener
	Task     *Task

	Type  string
	Value types.Context
}

func (ctx *Context) checkNil() {
	if ctx.Session == nil {
		ctx.Session = &Session{}
	}
	if ctx.Pipeline == nil {
		ctx.Pipeline = &clientpb.Pipeline{}
	}
	if ctx.Task == nil {
		ctx.Task = &Task{}
	}
	if ctx.Listener == nil {
		ctx.Listener = &Listener{}
	}
}

func (ctx *Context) ToProtobuf() *clientpb.Context {
	ctx.checkNil()
	resp := &clientpb.Context{
		Session:  ctx.Session.ToProtobuf(),
		Listener: ctx.Listener.ToProtobuf(),
		Pipeline: ctx.Pipeline,
		Task:     ctx.Task.ToProtobuf(),
		Type:     ctx.Type,
		Value:    ctx.Value.String(),
	}
	return resp
}

func NewContext(ctx *clientpb.Context) (*Context, error) {
	session, ok := Sessions.Get(ctx.Session.SessionId)
	if !ok {
		return nil, errs.ErrNotFoundSession
	}
	listener, err := Listeners.Get(ctx.Listener.Id)
	if err != nil {
		return nil, err
	}
	pipeline := listener.Pipelines[ctx.Pipeline.Name]
	task := session.Tasks.Get(ctx.Task.TaskId)
	var value types.Context
	if ctx.Type == consts.ScreenShotType {
		value, err = types.NewScreenShot([]byte(ctx.Value))
		if err != nil {
			return nil, err
		}
	} else if ctx.Type == consts.CredentialType {
		value, err = types.NewCredential([]byte(ctx.Value))
		if err != nil {
			return nil, err
		}
	} else if ctx.Type == consts.KeyLoggerType {
		value, err = types.NewKeyLogger([]byte(ctx.Value))
		if err != nil {
			return nil, err
		}
	}
	context := &Context{
		ID:       ctx.Id,
		Session:  session,
		Pipeline: pipeline,
		Listener: listener,
		Task:     task,
		Type:     ctx.Type,
		Value:    value,
	}

	context.checkNil()
	return context, nil
}

func NewContexts() *contexts {
	newContexts := &contexts{
		Map: sync.Map{},
	}
	Contexts = newContexts
	return newContexts
}

func (ctxs *contexts) filterContexts(filterFunc func(*Context) bool) *clientpb.Contexts {
	var resp *clientpb.Contexts
	ctxs.Map.Range(func(key, value interface{}) bool {
		if c, ok := value.(*Context); ok {
			if filterFunc(c) {
				resp.Contexts = append(resp.Contexts, c.ToProtobuf())
			}
		}
		return true
	})
	return resp
}

func (ctxs *contexts) ScreenShot() *clientpb.Contexts {
	return ctxs.filterContexts(func(c *Context) bool {
		return c.Type == consts.ScreenShotType
	})
}

func (ctxs *contexts) KeyLogger() *clientpb.Contexts {
	return ctxs.filterContexts(func(c *Context) bool {
		return c.Type == consts.KeyLoggerType
	})
}

func (ctxs *contexts) Credential() *clientpb.Contexts {
	return ctxs.filterContexts(func(c *Context) bool {
		return c.Type == consts.CredentialType
	})
}

func (ctxs *contexts) Add(c *Context) {
	ctxs.Store(c.ID, c)
}

func (ctxs *contexts) Remove(cID string) {
	val, ok := ctxs.Load(cID)
	if !ok {
		logs.Log.Errorf("Context not found: %s", cID)
		return
	}
	v := val.(*Context)
	ctxs.Delete(v.ID)

}

func (ctxs *contexts) Get(cID string) (*Context, bool) {
	val, ok := ctxs.Load(cID)
	if !ok {
		return nil, false
	}
	v := val.(*Context)
	return v, true
}

func (ctxs *contexts) All() []*Context {
	var contexts []*Context
	ctxs.Map.Range(func(key, value interface{}) bool {
		if c, ok := value.(*Context); ok {
			contexts = append(contexts, c)
		}
		return true
	})
	return contexts
}

func RecoverContext() error {
	dbContexts, err := db.GetAllContext()
	if err != nil {
		return err
	}
	for _, c := range dbContexts {
		context := c.ToProtobuf()
		newContext, err := NewContext(context)
		if err != nil {
			return err
		}
		Contexts.Add(newContext)
	}
	return nil
}
