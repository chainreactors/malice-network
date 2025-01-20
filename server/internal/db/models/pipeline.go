package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// Pipeline
type Pipeline struct {
	ID                    uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt             time.Time `gorm:"->;<-:create;"`
	ListenerId            string    `gorm:"type:string;"`
	Name                  string    `gorm:"unique,type:string"`
	IP                    string    `gorm:"type:string;default:''"`
	Host                  string    `config:"host"`
	Port                  uint32    `config:"port"`
	Type                  string    `gorm:"type:string;"`
	Enable                bool      `gorm:"type:boolean;"`
	ParamsData            string    `gorm:"column:params"`
	*types.PipelineParams `gorm:"-"`
}

func (pipeline *Pipeline) ToProtobuf() *clientpb.Pipeline {
	switch pipeline.Type {
	case consts.TCPPipeline:
		return &clientpb.Pipeline{
			Name:       pipeline.Name,
			ListenerId: pipeline.ListenerId,
			Enable:     pipeline.Enable,
			Parser:     pipeline.Parser,
			Ip:         pipeline.IP,
			Type:       consts.TCPPipeline,
			Body: &clientpb.Pipeline_Tcp{
				Tcp: &clientpb.TCPPipeline{
					Name:       pipeline.Name,
					ListenerId: pipeline.ListenerId,
					Host:       pipeline.Host,
					Port:       uint32(pipeline.Port),
				},
			},
			Tls:        pipeline.Tls.ToProtobuf(),
			Encryption: pipeline.Encryption.ToProtobuf(),
		}
	case consts.BindPipeline:
		return &clientpb.Pipeline{
			Name:       pipeline.Name,
			ListenerId: pipeline.ListenerId,
			Enable:     pipeline.Enable,
			Parser:     pipeline.Parser,
			Ip:         pipeline.IP,
			Type:       consts.BindPipeline,
			Body: &clientpb.Pipeline_Bind{
				Bind: &clientpb.BindPipeline{
					Name:       pipeline.Name,
					ListenerId: pipeline.ListenerId,
				},
			},
			Tls:        pipeline.Tls.ToProtobuf(),
			Encryption: pipeline.Encryption.ToProtobuf(),
		}
	case consts.WebsitePipeline:
		return &clientpb.Pipeline{
			Name:       pipeline.Name,
			ListenerId: pipeline.ListenerId,
			Ip:         pipeline.IP,
			Enable:     pipeline.Enable,
			Parser:     pipeline.Parser,
			Type:       consts.WebsitePipeline,
			Body: &clientpb.Pipeline_Web{
				Web: &clientpb.Website{
					Name:       pipeline.Name,
					ListenerId: pipeline.ListenerId,
					Root:       pipeline.WebPath,
					Port:       uint32(pipeline.Port),
					Contents:   make(map[string]*clientpb.WebContent),
				},
			},
			Tls:        pipeline.Tls.ToProtobuf(),
			Encryption: pipeline.Encryption.ToProtobuf(),
		}
	case consts.RemPipeline:
		return &clientpb.Pipeline{
			Name:       pipeline.Name,
			ListenerId: pipeline.ListenerId,
			Enable:     pipeline.Enable,
			Type:       consts.RemPipeline,
			Ip:         pipeline.IP,
			Body: &clientpb.Pipeline_Rem{
				Rem: &clientpb.REM{
					Name:      pipeline.Name,
					Host:      pipeline.Host,
					Port:      pipeline.Port,
					Link:      pipeline.PipelineParams.Link,
					Subscribe: pipeline.PipelineParams.Subscribe,
					Console:   pipeline.Console,
					Agents:    pipeline.PipelineParams.Agents,
				},
			},
			Tls:        pipeline.Tls.ToProtobuf(),
			Encryption: pipeline.Encryption.ToProtobuf(),
		}
	default:
		return nil
	}
}
func (pipeline *Pipeline) Address() string {
	return fmt.Sprintf("%s:%d", pipeline.IP, pipeline.Port)
}

// BeforeCreate - GORM hook
func (pipeline *Pipeline) BeforeCreate(tx *gorm.DB) (err error) {
	pipeline.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	pipeline.CreatedAt = time.Now()
	return nil
}

// BeforeSave GORM 钩子 - 保存前将 Params 序列化
func (pipeline *Pipeline) BeforeSave(tx *gorm.DB) error {
	if pipeline.PipelineParams != nil {
		data, err := json.Marshal(pipeline.PipelineParams)
		if err != nil {
			return err
		}
		pipeline.ParamsData = string(data)
	}
	return nil
}

// AfterFind GORM 钩子 - 查询后反序列化
func (pipeline *Pipeline) AfterFind(tx *gorm.DB) error {
	if pipeline.ParamsData == "" {
		return nil
	}
	var params types.PipelineParams
	if err := json.Unmarshal([]byte(pipeline.ParamsData), &params); err != nil {
		return err
	}
	pipeline.PipelineParams = &params
	return nil
}

func FromPipelinePb(pipeline *clientpb.Pipeline) *Pipeline {
	switch body := pipeline.Body.(type) {
	case *clientpb.Pipeline_Tcp:
		return &Pipeline{
			ListenerId: pipeline.ListenerId,
			Name:       pipeline.Name,
			Enable:     pipeline.Enable,
			Host:       body.Tcp.Host,
			IP:         pipeline.Ip,
			Port:       body.Tcp.Port,
			Type:       consts.TCPPipeline,
			PipelineParams: &types.PipelineParams{
				Parser:     pipeline.Parser,
				Tls:        types.FromTls(pipeline.Tls),
				Encryption: types.FromEncryption(pipeline.Encryption),
			},
		}
	case *clientpb.Pipeline_Bind:
		return &Pipeline{
			ListenerId: pipeline.ListenerId,
			Name:       pipeline.Name,
			Enable:     pipeline.Enable,
			IP:         pipeline.Ip,
			Type:       consts.BindPipeline,
			PipelineParams: &types.PipelineParams{
				Parser:     pipeline.Parser,
				Tls:        types.FromTls(pipeline.Tls),
				Encryption: types.FromEncryption(pipeline.Encryption),
			},
		}
	case *clientpb.Pipeline_Rem:
		return &Pipeline{
			ListenerId: pipeline.ListenerId,
			Name:       pipeline.Name,
			Enable:     pipeline.Enable,
			Type:       consts.RemPipeline,
			Host:       body.Rem.Host,
			Port:       body.Rem.Port,
			IP:         pipeline.Ip,
			PipelineParams: &types.PipelineParams{
				Link:      body.Rem.Link,
				Subscribe: body.Rem.Subscribe,
				Agents:    body.Rem.Agents,
				Console:   body.Rem.Console,
			},
		}
	case *clientpb.Pipeline_Web:
		return &Pipeline{
			ListenerId: pipeline.ListenerId,
			Name:       pipeline.Name,
			Enable:     pipeline.Enable,
			IP:         pipeline.Ip,
			Port:       body.Web.Port,
			Type:       consts.WebsitePipeline,
			PipelineParams: &types.PipelineParams{
				WebPath: body.Web.Root,
				Tls:     types.FromTls(pipeline.Tls),
			},
		}

	default:
		return nil
	}
}
