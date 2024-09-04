package models

import (
	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
	"time"
)

// Pipeline
type Pipeline struct {
	ID         uuid.UUID        `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt  time.Time        `gorm:"->;<-:create;"`
	ListenerID string           `gorm:"type:string;"`
	Name       string           `gorm:"type:string"`
	WebPath    string           `gorm:"type:string;default:''"`
	Host       string           `config:"host"`
	Port       uint16           `config:"port"`
	Type       string           `gorm:"type:string;"`
	Tls        TlsConfig        `gorm:"embedded;embeddedPrefix:tls_"`
	Encryption EncryptionConfig `gorm:"embedded;embeddedPrefix:encryption_"`
}

type TlsConfig struct {
	Enable   bool   `gorm:"column:enable"`
	Name     string `gorm:"column:name"`
	CN       string `gorm:"column:cn"`
	O        string `gorm:"column:o"`
	C        string `gorm:"column:c"`
	L        string `gorm:"column:l"`
	OU       string `gorm:"column:ou"`
	ST       string `gorm:"column:st"`
	Validity string `gorm:"column:validity"`
	Cert     string `gorm:"column:cert"`
	Key      string `gorm:"column:key"`
}

type EncryptionConfig struct {
	Enable bool   `gorm:"column:enable"`
	Type   string `gorm:"column:type"`
	Key    string `gorm:"column:key"`
}

// BeforeCreate - GORM hook
func (l *Pipeline) BeforeCreate(tx *gorm.DB) (err error) {
	l.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	l.CreatedAt = time.Now()
	return nil
}

func ProtoBufToDB(pipeline *lispb.Pipeline) Pipeline {
	switch body := pipeline.Body.(type) {
	case *lispb.Pipeline_Tcp:
		return Pipeline{
			ListenerID: body.Tcp.ListenerId,
			Name:       body.Tcp.Name,
			Host:       body.Tcp.Host,
			Port:       uint16(body.Tcp.Port),
			Type:       "tcp",
			Tls:        ToTlsDB(pipeline.Tls),
		}
	case *lispb.Pipeline_Web:
		return Pipeline{
			ListenerID: body.Web.ListenerId,
			Name:       body.Web.Name,
			WebPath:    body.Web.RootPath,
			Port:       uint16(body.Web.Port),
			Type:       "web",
			Tls:        ToTlsDB(pipeline.Tls),
		}
	default:
		return Pipeline{}
	}
}

func ToTlsDB(tls *lispb.TLS) TlsConfig {
	return TlsConfig{
		Cert: tls.Cert,
		Key:  tls.Key,
	}
}

//func ToEncryptionDB(encryption *lispb.Encryption) EncryptionConfig {
//	return EncryptionConfig{
//		Type: encryption.Type,
//		Key:  encryption.Key,
//	}
//}

func ToProtobuf(pipeline *Pipeline) *lispb.Pipeline {
	switch pipeline.Type {
	case "tcp":
		return &lispb.Pipeline{
			Body: &lispb.Pipeline_Tcp{
				Tcp: &lispb.TCPPipeline{
					Name: pipeline.Name,
					Host: pipeline.Host,
					Port: uint32(pipeline.Port),
				},
			},
			Tls: ToTlsProtobuf(&pipeline.Tls),
		}
	case "web":
		return &lispb.Pipeline{
			Body: &lispb.Pipeline_Web{
				Web: &lispb.Website{
					Name:     pipeline.Name,
					RootPath: pipeline.WebPath,
					Port:     uint32(pipeline.Port),
				},
			},
			Tls: ToTlsProtobuf(&pipeline.Tls),
		}
	default:
		return nil
	}
}

func ToTlsProtobuf(tls *TlsConfig) *lispb.TLS {
	return &lispb.TLS{
		Cert: tls.Cert,
		Key:  tls.Key,
	}
}
