package models

import (
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
	"strconv"
	"time"
)

type Context struct {
	ID           uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt    time.Time `gorm:"->;<-:create;"`
	SessionID    string    `gorm:"type:string;index;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	PipelineName string    `gorm:"type:string;index;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	ListenerName string    `gorm:"type:string;index;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	TaskID       string    `gorm:"type:string;index;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Type         string
	Value        string
	value        types.Context `gorm:"-"`

	Session  *Session  `gorm:"foreignKey:SessionID;references:SessionID;"`
	Pipeline *Pipeline `gorm:"foreignKey:PipelineName;references:Name;"`
	Listener *Operator `gorm:"foreignKey:ListenerName;references:Name;"`
	Task     *Task     `gorm:"foreignKey:TaskID;references:ID;"`
}

func (c *Context) BeforeCreate(tx *gorm.DB) (err error) {
	c.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	c.CreatedAt = time.Now()
	return nil
}

func (c *Context) ToProtobuf() *clientpb.Context {
	if c.Session == nil {
		c.Session = &Session{}
	}
	if c.Pipeline == nil {
		c.Pipeline = &Pipeline{}
	}
	if c.Task == nil {
		c.Task = &Task{}
	}
	return &clientpb.Context{
		Id:       c.ID.String(),
		Session:  c.Session.ToProtobuf(),
		Pipeline: c.Pipeline.ToProtobuf(),
		Task:     c.Task.ToProtobuf(),
		Type:     c.Type,
		Value:    c.Value,
	}
}

func ToContextDB(ctx *clientpb.Context) *Context {
	context := &Context{
		ID:           uuid.FromStringOrNil(ctx.Id),
		SessionID:    ctx.Session.SessionId,
		PipelineName: ctx.Pipeline.Name,
		ListenerName: ctx.Listener.Id,
		TaskID:       ctx.Task.SessionId + "-" + strconv.Itoa(int(ctx.Task.TaskId)),
		Type:         ctx.Type,
		Value:        ctx.Value,
	}
	switch context.Type {
	case consts.ScreenShotType:
		shot, err := types.NewScreenShot([]byte(ctx.Value))
		if err != nil {
			return nil
		}
		context.value = shot
	case consts.CredentialType:
		credential, err := types.NewCredential([]byte(ctx.Value))
		if err != nil {
			return nil
		}
		context.value = credential
	case consts.KeyLoggerType:
		keyLogger, err := types.NewKeyLogger([]byte(ctx.Value))
		if err != nil {
			return nil
		}
		context.value = keyLogger
	}
	return context
}
