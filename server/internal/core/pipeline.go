package core

import (
	"github.com/chainreactors/malice-network/helper/cryptography/stream"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/utils/peek"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"net"
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

func FromProtobuf(pipeline *clientpb.Pipeline) *PipelineConfig {
	return &PipelineConfig{
		ListenerID: pipeline.ListenerId,
		Parser:     pipeline.Parser,
		Tls: &configs.CertConfig{
			Cert:   pipeline.GetTls().Cert,
			Key:    pipeline.GetTls().Key,
			Enable: pipeline.GetTls().Enable,
		},
		Encryption: &configs.EncryptionConfig{
			Enable: pipeline.GetEncryption().Enable,
			Type:   pipeline.GetEncryption().Type,
			Key:    pipeline.GetEncryption().Key,
		},
	}
}

type PipelineConfig struct {
	ListenerID string
	Parser     string
	Tls        *configs.CertConfig
	Encryption *configs.EncryptionConfig
}

func (p *PipelineConfig) WrapConn(conn net.Conn) (*peek.Conn, error) {
	cry, err := p.Encryption.NewCrypto()
	if err != nil {
		return nil, err
	}
	conn = cryptostream.NewCryptoConn(conn, cry)
	return peek.WrapPeekConn(conn), nil
}

//
//func (p *PipelineConfig) ToProtobuf() *clientpb.Pipeline {
//	return &clientpb.Pipeline{
//		Tls: &clientpb.TLS{
//			Cert:   p.TlsConfig.Cert,
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
