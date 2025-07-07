package core

import (
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/stream"
	"io"
)

type Pipeline interface {
	ID() string
	Start() error
	Close() error
	ToProtobuf() *clientpb.Pipeline
}

type Pipelines map[string]Pipeline

func (ps Pipelines) Add(p Pipeline) {
	ps[p.ID()] = p
}

func (ps Pipelines) Get(id string) Pipeline {
	return ps[id]
}

func (ps Pipelines) ToProtobuf() *clientpb.Pipelines {
	var pls = &clientpb.Pipelines{
		Pipelines: make([]*clientpb.Pipeline, 0),
	}
	for _, p := range ps {
		pls.Pipelines = append(pls.Pipelines, p.ToProtobuf())
	}
	return pls
}

func FromPipeline(pipeline *clientpb.Pipeline) *PipelineConfig {
	return &PipelineConfig{
		ListenerID: pipeline.ListenerId,
		Parser:     pipeline.Parser,
		TLSConfig:  types.FromTls(pipeline.Tls),
		Encryption: types.FromEncryptions(pipeline.GetEncryption()),
	}
}

type PipelineConfig struct {
	ListenerID string
	Parser     string
	TLSConfig  *types.TlsConfig
	Encryption types.EncryptionsConfig
}

func (p *PipelineConfig) WrapConn(conn io.ReadWriteCloser) (*cryptostream.Conn, error) {
	crys, err := configs.NewCrypto(p.Encryption.ToProtobuf())
	if err != nil {
		return nil, err
	}
	return cryptostream.WrapPeekConn(conn, crys, p.Parser)
}

//
//func (p *PipelineConfig) ToFile() *clientpb.Pipeline {
//	return &clientpb.Pipeline{
//		Tls: &clientpb.TLS{
//			TLSConfig:   p.TlsConfig.TLSConfig,
//			Key:    p.TlsConfig.Key,
//			Enable: p.TlsConfig.Enable,
//		},
//		Encryption: &clientpb.Encryption{
//			Enable: p.Encryption.Enable,
//			Type:   p.Encryption.Type,
//			Key:    p.Encryption.Key,
//		},
//	}
//}
