package certs

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"os"
)

const (
	// SERVERCA - Directory containing operator certificates
	SERVERCA = "root"
	// CLIENTCA - Directory containing client certificates
	CLIENTCA = "client"
)

// InitRSACertificate - Generate a server certificate signed with a given CA
func InitRSACertificate(host, user string, isCA, isClient bool) ([]byte, []byte, error) {
	certsLog.Debugf("Generating TLS certificate (RSA) for '%s' ...", host)

	var privateKey interface{}
	var err error
	var caType string

	// Generate private key
	privateKey, err = rsa.GenerateKey(rand.Reader, RsaKeySize())
	if err != nil {
		certsLog.Errorf("Failed to generate private key %v", err)
		return nil, nil, err
	}
	subject := randomSubject(host)
	switch isClient {
	case true:
		caType = CLIENTCA
	case false:
		caType = SERVERCA
	}
	cert, key := generateCertificate(caType, (*subject), isCA, isClient, privateKey)
	err = saveCertificate(caType, RSAKey, fmt.Sprintf("%s.%s", host, user), cert, key)
	// 保存到文件
	if isCA {
		filename := fmt.Sprintf(configs.CertsPath+"/%s_%s", host, user)
		if certErr := os.WriteFile(filename+"_crt.pem", cert, 0o777); certErr != nil {
			return nil, nil, certErr
		}
		if keyErr := os.WriteFile(filename+"_key.pem", key, 0o777); keyErr != nil {
			return nil, nil, keyErr
		}
	}
	return cert, key, nil
}
