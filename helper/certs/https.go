package certs

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
)

const (
	// SERVERCA - Directory containing operator certificates
	SERVERCA = "root"
	ROOTPATH = ".malice/certs/"
)

// InitRSACertificate - Generate a server certificate signed with a given CA
func InitRSACertificate(host, user string, isCA, isClient bool) ([]byte, []byte, error) {
	certsLog.Debugf("Generating TLS certificate (RSA) for '%s' ...", host)

	var privateKey interface{}
	var err error

	// Generate private key
	privateKey, err = rsa.GenerateKey(rand.Reader, RsaKeySize())
	if err != nil {
		certsLog.Errorf("Failed to generate private key %v", err)
		return nil, nil, err
	}
	subject := randomSubject(host)
	cert, key := generateCertificate(SERVERCA, (*subject), isCA, isClient, privateKey)
	//err = saveCertificate(SERVERCA, RSAKey, host, cert, key)
	// 保存到文件
	if isCA {
		filename := fmt.Sprintf(ROOTPATH+"%s_%s", host, user)
		if certErr := SaveToPEMFile(filename+".crt", cert); certErr != nil {
			return nil, nil, certErr
		}
		if keyErr := SaveToPEMFile(filename+".key", key); keyErr != nil {
			return nil, nil, keyErr
		}
	}
	return cert, key, nil
}
