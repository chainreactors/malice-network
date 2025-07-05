package types

import (
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"math/rand"
)

func FromTls(tls *clientpb.TLS) *TlsConfig {
	return &TlsConfig{
		Cert:     FromCert(tls.Cert),
		CA:       FromCert(tls.Ca),
		Enable:   tls.Enable,
		AutoCert: tls.AutoCert,
	}
}

func FromCert(cert *clientpb.Cert) *CertConfig {
	if cert == nil {
		return nil
	}
	return &CertConfig{
		Cert: cert.Cert,
		Key:  cert.Key,
	}
}

func FromEncryption(encryption *clientpb.Encryption) *EncryptionConfig {
	if encryption == nil {
		return nil
	}
	return &EncryptionConfig{
		Type: encryption.Type,
		Key:  encryption.Key,
	}
}

type CertConfig struct {
	Enable bool   `json:"enable" yaml:"enable" config:"enable"`
	Cert   string `json:"cert" yaml:"cert" config:"cert"`
	Key    string `json:"key" yaml:"key" config:"key"`
}

func (cert *CertConfig) ToProtobuf() *clientpb.Cert {
	if cert == nil {
		return nil
	}
	return &clientpb.Cert{
		Cert: cert.Cert,
		Key:  cert.Key,
	}
}

type TlsConfig struct {
	AutoCert bool        `json:"auto_cert"`
	Enable   bool        `json:"enable"`
	Cert     *CertConfig `json:"cert"`
	CA       *CertConfig `json:"ca"`
	Domain   string      `json:"domain"`
}

func (tls *TlsConfig) Empty() bool {
	return tls == nil || tls.Cert == nil
}

func (tls *TlsConfig) ToProtobuf() *clientpb.TLS {
	if tls == nil {
		return &clientpb.TLS{
			Enable: false,
		}
	}
	return &clientpb.TLS{
		Enable: tls.Enable,
		Cert:   tls.Cert.ToProtobuf(),
		Ca:     tls.CA.ToProtobuf(),
	}
}

type EncryptionsConfig []*EncryptionConfig

func (e EncryptionsConfig) ToProtobuf() []*clientpb.Encryption {
	var encryptions []*clientpb.Encryption
	for _, e := range e {
		encryptions = append(encryptions, e.ToProtobuf())
	}
	return encryptions
}

func (e EncryptionsConfig) Choice() *EncryptionConfig {
	return e[rand.Intn(len(e))]
}

func FromEncryptions(es []*clientpb.Encryption) EncryptionsConfig {
	var encryptions EncryptionsConfig
	for _, e := range es {
		encryptions = append(encryptions, &EncryptionConfig{
			Type: e.Type,
			Key:  e.Key,
		})
	}
	return encryptions
}

type EncryptionConfig struct {
	Type string `json:"type" config:"type"`
	Key  string `json:"key" config:"key"`
}

func (encryption *EncryptionConfig) ToProtobuf() *clientpb.Encryption {
	if encryption == nil {
		return &clientpb.Encryption{}
	}
	return &clientpb.Encryption{
		Type: encryption.Type,
		Key:  encryption.Key,
	}
}

type PipelineParams struct {
	Parser     string                        `json:"parser,omitempty"`
	WebPath    string                        `json:"path,omitempty"`
	Link       string                        `json:"link,omitempty"`
	Console    string                        `json:"console,omitempty"`
	Subscribe  string                        `json:"subscribe,omitempty"`
	Agents     map[string]*clientpb.REMAgent `json:"agents,omitempty"`
	Encryption EncryptionsConfig             `json:"encryption,omitempty"`
	Tls        *TlsConfig                    `json:"tls,omitempty"`
	// HTTP pipeline specific params
	Headers    map[string][]string `json:"headers,omitempty"`
	ErrorPage  string              `json:"error_page,omitempty" gorm:"-"`
	BodyPrefix string              `json:"body_prefix,omitempty"`
	BodySuffix string              `json:"body_suffix,omitempty"`
}
