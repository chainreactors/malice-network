package models

import (
	"strconv"
	"time"

	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

type Context struct {
	ID         uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid"`
	CreatedAt  time.Time `gorm:"->;<-:create"`
	SessionID  string    `gorm:"type:string;index;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	PipelineID string    `gorm:"type:string;index;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	ListenerID string    `gorm:"type:string;index;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	TaskID     string    `gorm:"type:string;index;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Type       string
	Value      []byte
	value      types.Context `gorm:"-"`

	Session  *Session  `gorm:"foreignKey:SessionID;references:SessionID;"`
	Pipeline *Pipeline `gorm:"foreignKey:PipelineID;references:Name;"`
	Listener *Operator `gorm:"foreignKey:ListenerID;references:Name;"`
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

func (c *Context) Context() types.Context {
	return c.value
}

func (c *Context) ToProtobuf() *clientpb.Context {
	return &clientpb.Context{
		Id:       c.ID.String(),
		Session:  c.Session.ToProtobuf(),
		Pipeline: c.Pipeline.ToProtobuf(),
		Listener: c.Listener.ToListener(),
		Task:     c.Task.ToProtobuf(),
		Type:     c.Type,
		Value:    c.Value,
	}
}

func FromContextProtobuf(ctx *clientpb.Context) (*Context, error) {
	context := &Context{
		Type:  ctx.Type,
		Value: ctx.Value,
	}

	if ctx.Pipeline != nil {
		context.PipelineID = ctx.Pipeline.Name
	}
	if ctx.Listener != nil {
		context.ListenerID = ctx.Listener.Id
	}
	if ctx.Session != nil {
		context.SessionID = ctx.Session.SessionId
	}
	if ctx.Task != nil && ctx.Session != nil {
		context.TaskID = ctx.Task.SessionId + "-" + strconv.Itoa(int(ctx.Task.TaskId))
	}

	var err error
	context.value, err = types.ParseContext(context.Type, []byte(context.Value))
	if err != nil {
		return nil, err
	}
	return context, nil
}

//func (c *Context) ToFileProtobuf() *clientpb.File {
//	re := regexp.MustCompile(`-(\d+)$`)
//	match := re.FindStringSubmatch(c.TaskID)
//	if len(match) < 1 {
//		return &clientpb.File{}
//	}
//	file, err := c.toFileDescription()
//	if err != nil {
//		logs.Log.Errorf("Error parsing task file JSON: %v", err)
//		return &clientpb.File{}
//	}
//	return &clientpb.File{
//		TaskId: match[1],
//		Name:   file.Name,
//		Local:  file.Name,
//		//Checksum:  file.Checksum,
//		Remote:    file.TargetPath,
//		SessionId: c.SessionID,
//		Op:        c.Type,
//	}
//}
//
//func (f *Context) toFileDescription() (*types.FileDescription, error) {
//	var desc types.FileDescription
//	err := json.Unmarshal([]byte(f.Value), &desc)
//	if err != nil {
//		return nil, err
//	}
//	return &desc, nil
//}
