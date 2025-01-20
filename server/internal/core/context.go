package core

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/errs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"sync"
)

type ContextsResp []*clientpb.Context

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

func (c *Context) ToProtobuf() *clientpb.Context {
	resp := &clientpb.Context{
		Session:  c.Session.ToProtobuf(),
		Listener: c.Listener.ToProtobuf(),
		Pipeline: c.Pipeline,
		Task:     c.Task.ToProtobuf(),
		Type:     c.Type,
		Value:    c.Value.String(),
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
	return context, nil
}

func NewContexts() *contexts {
	newContexts := &contexts{
		Map: sync.Map{},
	}
	Contexts = newContexts
	return newContexts
}

func (ctx *contexts) WithSession(sid string) ContextsResp {
	var contexts ContextsResp

	ctx.Map.Range(func(key, value interface{}) bool {
		if c, ok := value.(*Context); ok {
			if c.Session.ID == sid {
				contexts = append(contexts, c.ToProtobuf())
			}
		}
		return true
	})
	return contexts
}

func (ctx *contexts) WithPipeline(pid string) ContextsResp {
	var contexts ContextsResp

	ctx.Map.Range(func(key, value interface{}) bool {
		if c, ok := value.(*models.Context); ok {
			c.PipelineName = pid
			contexts = append(contexts, c.ToProtobuf())
		}
		return true
	})
	return contexts
}

func (ctx *contexts) WithListener(lName string) ContextsResp {
	var contexts ContextsResp

	ctx.Map.Range(func(key, value interface{}) bool {
		if c, ok := value.(*models.Context); ok {
			c.ListenerName = lName
			contexts = append(contexts, c.ToProtobuf())
		}
		return true
	})
	return contexts

}

func (ctx *contexts) WithTask(tName string) ContextsResp {
	var contexts ContextsResp

	ctx.Map.Range(func(key, value interface{}) bool {
		if c, ok := value.(*models.Context); ok {
			c.TaskID = tName
			contexts = append(contexts, c.ToProtobuf())
		}
		return true
	})
	return contexts

}

func (ctx *contexts) ScreenShot() ContextsResp {
	var contexts ContextsResp

	ctx.Map.Range(func(key, value interface{}) bool {
		if c, ok := value.(*models.Context); ok {
			if c.Type == consts.ScreenShotType {
				contexts = append(contexts, c.ToProtobuf())
			}
		}
		return true
	})
	return contexts
}

func (ctx *contexts) KeyLogger() ContextsResp {
	var contexts ContextsResp

	ctx.Map.Range(func(key, value interface{}) bool {
		if c, ok := value.(*models.Context); ok {
			if c.Type == consts.KeyLoggerType {
				contexts = append(contexts, c.ToProtobuf())
			}
		}
		return true
	})
	return contexts
}

func (ctx *contexts) Credential() ContextsResp {
	var contexts ContextsResp

	ctx.Map.Range(func(key, value interface{}) bool {
		if c, ok := value.(*models.Context); ok {
			if c.Type == consts.CredentialType {
				contexts = append(contexts, c.ToProtobuf())
			}
		}
		return true
	})
	return contexts
}

func (ctx *contexts) Add(c *Context) {
	ctx.Store(c.ID, c)
}

func (ctx *contexts) Remove(cID string) {
	val, ok := ctx.Load(cID)
	if !ok {
		logs.Log.Errorf("Context not found: %s", cID)
		return
	}
	v := val.(*Context)
	ctx.Delete(v.ID)

}

func (ctx *contexts) Get(cID string) (*Context, bool) {
	val, ok := ctx.Load(cID)
	if !ok {
		return nil, false
	}
	v := val.(*Context)
	return v, true
}

func (ctx *contexts) All() []*Context {
	var contexts []*Context
	ctx.Map.Range(func(key, value interface{}) bool {
		if c, ok := value.(*Context); ok {
			contexts = append(contexts, c)
		}
		return true
	})
	return contexts
}

func RecoverContext() error {
	contexts, err := db.GetAllContext()
	if err != nil {
		return err
	}
	for _, c := range contexts {
		context := c.ToProtobuf()
		newContext, err := NewContext(context)
		if err != nil {
			return err
		}
		Contexts.Add(newContext)
	}
	return nil
}
