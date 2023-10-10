package certs

import (
	"crypto/rand"
	"crypto/rsa"
)

const (
	// SERVERCA - Directory containing operator certificates
	SERVERCA = "root"
)

// InitRSACertificate - Generate a server certificate signed with a given CA
func InitRSACertificate(host, user string, isCA, isClient bool) error {
	certsLog.Debugf("Generating TLS certificate (RSA) for '%s' ...", host)

	var privateKey interface{}
	var err error

	// Generate private key
	privateKey, err = rsa.GenerateKey(rand.Reader, RsaKeySize())
	if err != nil {
		certsLog.Errorf("Failed to generate private key %v", err)
		return err
	}
	subject := randomSubject(host)
	cert, key := generateCertificate(SERVERCA, (*subject), isCA, isClient, privateKey)
	err = saveCertificate(SERVERCA, RSAKey, host, cert, key)
	// 保存到文件
	if certErr := saveToPEMFile(host+user+".crt", cert); certErr != nil {
		return certErr
	}

	if keyErr := saveToPEMFile(host+user+".key", key); keyErr != nil {
		return keyErr
	}

	return nil
}
