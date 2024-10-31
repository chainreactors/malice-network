package listener

import (
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/configs"
)

func FromProtobuf(pipeline *clientpb.Pipeline) *PipelineConfig {
	return &PipelineConfig{
		TlsConfig: &configs.CertConfig{
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
	TlsConfig  *configs.CertConfig
	Encryption *configs.EncryptionConfig
}

func (p *PipelineConfig) ToProtobuf() *clientpb.Pipeline {
	return &clientpb.Pipeline{
		Tls: &clientpb.TLS{
			Cert:   p.TlsConfig.Cert,
			Key:    p.TlsConfig.Key,
			Enable: p.TlsConfig.Enable,
		},
		Encryption: &clientpb.Encryption{
			Enable: p.Encryption.Enable,
			Type:   p.Encryption.Type,
			Key:    p.Encryption.Key,
		},
	}
}
