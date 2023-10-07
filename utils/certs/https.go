package certs

import (
	"crypto/rand"
	"crypto/rsa"
)

const (
	// HTTPSCA - Directory containing operator certificates
	HTTPSCA = "https"
)

// HTTPSGenerateRSACertificate - Generate a server certificate signed with a given CA
func HTTPSGenerateRSACertificate(host string) ([]byte, []byte, error) {
	// TODO - log generating TLS certificate (RSA)
	//certsLog.Debugf("Generating TLS certificate (RSA) for '%s' ...", host)

	var privateKey interface{}
	var err error

	// Generate private key
	privateKey, err = rsa.GenerateKey(rand.Reader, rsaKeySize())
	if err != nil {
		// TODO - log failed to generate private key
		//certsLog.Fatalf("Failed to generate private key %s", err)
		return nil, nil, err
	}
	subject := randomSubject(host)
	cert, key := generateCertificate(HTTPSCA, (*subject), false, false, privateKey)
	err = saveCertificate(HTTPSCA, RSAKey, host, cert, key)
	return cert, key, err
}
