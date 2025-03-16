package models

import (
	"github.com/chainreactors/malice-network/helper/utils/output"
	"strconv"
	"time"

	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

type Context struct {
	ID         uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid"`
	CreatedAt  time.Time `gorm:"->;<-:create"`
	SessionID  string    `gorm:"type:string;index;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	PipelineID string    `gorm:"type:string;index;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	TaskID     string    `gorm:"type:string;index;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Type       string
	Nonce      string
	Value      []byte
	Context    output.Context `gorm:"-"`

	Session  *Session  `gorm:"foreignKey:SessionID;references:SessionID;"`
	Pipeline *Pipeline `gorm:"foreignKey:PipelineID;references:Name;"`
	Task     *Task     `gorm:"foreignKey:TaskID;references:ID;"`
}

func (c *Context) BeforeCreate(tx *gorm.DB) (err error) {
	if c.ID == uuid.Nil {
		c.ID, err = uuid.NewV4()
		if err != nil {
			return err
		}
	}
	c.CreatedAt = time.Now()
	return nil
}

func (c *Context) AfterFind(tx *gorm.DB) (err error) {
	c.Context, err = output.ParseContext(c.Type, c.Value)
	return err
}

func (c *Context) ToProtobuf() *clientpb.Context {
	return &clientpb.Context{
		Id:       c.ID.String(),
		Session:  c.Session.ToProtobuf(),
		Pipeline: c.Pipeline.ToProtobuf(),
		Task:     c.Task.ToProtobuf(),
		Type:     c.Type,
		Value:    c.Value,
	}
}

func FromContextProtobuf(ctx *clientpb.Context) (*Context, error) {
	context := &Context{
		Type:  ctx.Type,
		Value: ctx.Value,
		Nonce: ctx.Nonce,
	}

	if ctx.Pipeline != nil {
		context.PipelineID = ctx.Pipeline.Name
	}

	if ctx.Session != nil {
		context.SessionID = ctx.Session.SessionId
	}
	if ctx.Task != nil && ctx.Session != nil {
		context.TaskID = ctx.Task.SessionId + "-" + strconv.Itoa(int(ctx.Task.TaskId))
	}

	var err error
	context.Context, err = output.ParseContext(context.Type, context.Value)
	if err != nil {
		return nil, err
	}
	return context, nil
}
