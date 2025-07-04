package types

import "github.com/chainreactors/malice-network/helper/proto/client/clientpb"

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
		Enable: encryption.Enable,
		Type:   encryption.Type,
		Key:    encryption.Key,
	}
}

type CertConfig struct {
	Cert string `json:"cert" yaml:"cert"`
	Key  string `json:"key" yaml:"key"`
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
	// HTTP pipeline specific params
	Headers    map[string][]string `json:"headers,omitempty"`
	ErrorPage  string              `json:"error_page,omitempty" gorm:"-"`
	BodyPrefix string              `json:"body_prefix,omitempty"`
	BodySuffix string              `json:"body_suffix,omitempty"`
}
