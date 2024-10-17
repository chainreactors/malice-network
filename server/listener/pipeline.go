package listener

import (
	"github.com/chainreactors/malice-network/server/internal/configs"
)

type Pipeline struct {
	TlsConfig  *configs.CertConfig
	Encryption *configs.EncryptionConfig
}
