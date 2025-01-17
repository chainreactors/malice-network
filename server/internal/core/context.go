package core

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"sync"
)

type ContextsResp []*clientpb.Context

type Context struct {
	sync.Map
}

var Contexts *Context

func NewContext() *Context {
	newContexts := &Context{
		Map: sync.Map{},
	}
	Contexts = newContexts
	return newContexts
}

func (ctx *Context) WithSession(sid string) ContextsResp {
	var contexts ContextsResp

	ctx.Map.Range(func(key, value interface{}) bool {
		if c, ok := value.(*models.Context); ok {
			c.SessionID = sid
			contexts = append(contexts, c.ToProtobuf())
		}
		return true
	})
	return contexts
}

func (ctx *Context) WithPipeline(pid string) ContextsResp {
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

func (ctx *Context) WithListener(lName string) ContextsResp {
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

func (ctx *Context) WithTask(tName string) ContextsResp {
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

func (ctx *Context) ScreenShot() ContextsResp {
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

func (ctx *Context) KeyLogger() ContextsResp {
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

func (ctx *Context) Credential() ContextsResp {
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

func (ctx *Context) Add(c *models.Context) {
	ctx.Store(c.ID.String(), c)
}

func (ctx *Context) Remove(cID string) {
	val, ok := ctx.Load(cID)
	if !ok {
		logs.Log.Errorf("Context not found: %s", cID)
		return
	}
	v := val.(*models.Context)
	ctx.Delete(v.ID)

}

func (ctx *Context) Get(cID string) (*models.Context, bool) {
	val, ok := ctx.Load(cID)
	if !ok {
		return nil, false
	}
	v := val.(*models.Context)
	return v, true
}

func (ctx *Context) All() []*models.Context {
	var contexts []*models.Context
	ctx.Map.Range(func(key, value interface{}) bool {
		if c, ok := value.(*models.Context); ok {
			contexts = append(contexts, c)
		}
		return true
	})
	return contexts
}

func (ctx *Context) Recover() error {
	contexts, err := db.GetAllContext()
	if err != nil {
		return err
	}
	for _, c := range contexts {
		ctx.Add(c)
	}
	return nil
}
