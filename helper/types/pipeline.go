package types

import (
	"crypto/x509/pkix"
	"encoding/json"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/utils"
	"math/rand"
)

func FromTls(tls *clientpb.TLS) *TlsConfig {
	if tls == nil {
		return &TlsConfig{
			Enable: false,
		}
	}
	return &TlsConfig{
		Cert:   FromCert(tls.Cert),
		CA:     FromCert(tls.Ca),
		Enable: tls.Enable,
		Acme:   tls.Acme,
		Domain: tls.Domain,
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
	Enable  bool        `json:"enable"`
	Acme    bool        `json:"acme"`
	Cert    *CertConfig `json:"cert"`
	CA      *CertConfig `json:"ca"`
	Domain  string      `json:"domain"`
	Subject *pkix.Name  `json:"subject"`
}

func (tls *TlsConfig) Empty() bool {
	return tls == nil || tls.Cert == nil
}

func (tls *TlsConfig) ToSubjectProtobuf() *clientpb.CertificateSubject {
	if tls.Subject == nil {
		return nil
	}
	return &clientpb.CertificateSubject{
		Cn: tls.Subject.CommonName,
		O:  utils.FirstOrEmpty(tls.Subject.Organization),
		C:  utils.FirstOrEmpty(tls.Subject.Country),
		L:  utils.FirstOrEmpty(tls.Subject.Locality),
		Ou: utils.FirstOrEmpty(tls.Subject.OrganizationalUnit),
		St: utils.FirstOrEmpty(tls.Subject.Province),
	}
}

func (tls *TlsConfig) ToProtobuf() *clientpb.TLS {
	if tls == nil {
		return &clientpb.TLS{
			Enable: false,
		}
	}
	return &clientpb.TLS{
		Enable:      tls.Enable,
		Cert:        tls.Cert.ToProtobuf(),
		Ca:          tls.CA.ToProtobuf(),
		Domain:      tls.Domain,
		Acme:        tls.Acme,
		CertSubject: tls.ToSubjectProtobuf(),
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
	if len(e) == 0 {
		return nil
	}
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

func (params *PipelineParams) String() string {
	content, err := json.Marshal(params)
	if err != nil {
		return ""
	}
	return string(content)
}

func UnmarshalPipelineParams(params string) (*PipelineParams, error) {
	if len(params) == 0 {
		return &PipelineParams{}, nil
	}
	var p *PipelineParams
	err := json.Unmarshal([]byte(params), &p)
	if err != nil {
		return p, err
	}
	return p, nil
}
