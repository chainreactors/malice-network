package types

import "github.com/chainreactors/malice-network/helper/proto/client/clientpb"

func FromTls(tls *clientpb.TLS) *TlsConfig {
	return &TlsConfig{
		Cert:   tls.Cert,
		Key:    tls.Key,
		Enable: tls.Enable,
	}
}

func FromEncryption(encryption *clientpb.Encryption) *EncryptionConfig {
	if encryption == nil {
		return nil
	}
	return &EncryptionConfig{
		Enable: encryption.Enable,
		Type:   encryption.Type,
		Key:    encryption.Key,
	}
}

type TlsConfig struct {
	Enable bool   `json:"enable"`
	Cert   string `json:"cert"`
	Key    string `json:"key"`
}

func (tls *TlsConfig) ToProtobuf() *clientpb.TLS {
	if tls == nil {
		return &clientpb.TLS{
			Enable: false,
		}
	}
	return &clientpb.TLS{
		Enable: tls.Enable,
		Cert:   tls.Cert,
		Key:    tls.Key,
	}
}

type EncryptionConfig struct {
	Enable bool   `json:"enable"`
	Type   string `json:"type"`
	Key    string `json:"key"`
}

func (encryption *EncryptionConfig) ToProtobuf() *clientpb.Encryption {
	if encryption == nil {
		return &clientpb.Encryption{
			Enable: false,
		}
	}
	return &clientpb.Encryption{
		Enable: encryption.Enable,
		Type:   encryption.Type,
		Key:    encryption.Key,
	}
}

type PipelineParams struct {
	Parser     string                        `json:"parser,omitempty"`
	WebPath    string                        `json:"path,omitempty"`
	Link       string                        `json:"link,omitempty"`
	Console    string                        `json:"console,omitempty"`
	Subscribe  string                        `json:"subscribe,omitempty"`
	Agents     map[string]*clientpb.REMAgent `json:"agents,omitempty"`
	Encryption *EncryptionConfig             `json:"encryption,omitempty"`
	Tls        *TlsConfig                    `json:"tls,omitempty"`
}
