package models

import (
	"github.com/chainreactors/malice-network/helper/certs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
	"time"
)

// Certificate - 通用证书数据库模型，支持自签名、Let's Encrypt、手动导入等多种类型
// 每个证书一套独立的CA，防止关联
// Type: selfsigned, letsencrypt, imported
// Acme: 适用于 Let's Encrypt 自动申请
// Remark: 备注

type Certificate struct {
	ID        uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt time.Time `gorm:"->;<-:create;"`
	Name      string    `gorm:"unique"`
	Type      string    // 证书类型: selfsigned, letsencrypt, imported
	Domain    string    // 证书绑定域名
	CertPEM   string    // 证书内容 PEM
	KeyPEM    string    // 私钥内容 PEM
	CACertPEM string    // CA 证书内容 PEM
	CAKeyPEM  string    // CA 私钥内容 PEM
}

// BeforeCreate - GORM hook 自动设置ID和创建时间
func (c *Certificate) BeforeCreate(tx *gorm.DB) (err error) {
	c.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	c.CreatedAt = time.Now()
	return nil
}

func firstOrEmpty(arr []string) string {
	if len(arr) > 0 {
		return arr[0]
	}
	return ""
}

func (c *Certificate) ToProtobuf() *clientpb.TLS {
	subject, err := certs.ExtractCertificateSubject(c.CertPEM)
	if err != nil {
		return nil
	}
	var domain string
	if c.Type == certs.Acme {
		domain = c.Name
	}
	return &clientpb.TLS{
		Enable: true,
		Cert: &clientpb.Cert{
			Name: c.Name,
			Type: c.Type,
			Cert: c.CertPEM,
			Key:  c.KeyPEM,
		},
		Ca: &clientpb.Cert{
			Cert: c.CACertPEM,
			Key:  c.CAKeyPEM,
		},
		CertSubject: &clientpb.CertificateSubject{
			Cn: subject.CommonName,
			O:  firstOrEmpty(subject.Organization),
			C:  firstOrEmpty(subject.Country),
			L:  firstOrEmpty(subject.Locality),
			Ou: firstOrEmpty(subject.OrganizationalUnit),
			St: firstOrEmpty(subject.StreetAddress),
		},
		Domain: domain,
	}
}
