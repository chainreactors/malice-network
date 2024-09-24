package certs

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"github.com/chainreactors/logs"
	"math/big"
	insecureRand "math/rand"
	"net"
	"os"
	"time"
)

// SaveToPEMFile save to PEM file
func SaveToPEMFile(filename string, pemData []byte) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(pemData)
	if err != nil {
		return err
	}

	return nil
}

func publicKey(priv interface{}) interface{} {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}

func pemBlockForKey(priv interface{}) *pem.Block {
	switch key := priv.(type) {
	case *rsa.PrivateKey:
		data := x509.MarshalPKCS1PrivateKey(key)
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: data}
	case *ecdsa.PrivateKey:
		data, err := x509.MarshalECPrivateKey(key)
		if err != nil {
			logs.Log.Errorf("Unable to marshal ECDSA private key: %v", err)
		}
		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: data}
	default:
		return nil
	}
}

func randomInt(max int) int {
	buf := make([]byte, 4)
	rand.Read(buf)
	i := binary.LittleEndian.Uint32(buf)
	return int(i) % max
}

func randomValidFor() time.Duration {
	validFor := 3 * (365 * 24 * time.Hour)
	switch insecureRand.Intn(2) {
	case 0:
		validFor = 2 * (365 * 24 * time.Hour)
	case 1:
		validFor = 3 * (365 * 24 * time.Hour)
	}
	return validFor
}

func RsaKeySize() int {
	rsaKeySizes := []int{4096, 2048}
	return rsaKeySizes[randomInt(len(rsaKeySizes))]
}

func GenerateCACert(commonName string) ([]byte, []byte, error) {
	subject := RandomSubject(commonName)
	privateKey, _ := rsa.GenerateKey(rand.Reader, RsaKeySize())
	notBefore := time.Now()
	days := randomInt(365) * -1
	notBefore = notBefore.AddDate(0, 0, days)
	notAfter := notBefore.Add(randomValidFor())

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, _ := rand.Int(rand.Reader, serialNumberLimit)

	template := x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               *subject,
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, publicKey(privateKey), privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create certificate: %v", err)
	}

	certOut := bytes.NewBuffer([]byte{})
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	keyOut := bytes.NewBuffer([]byte{})
	pem.Encode(keyOut, pemBlockForKey(privateKey))

	return certOut.Bytes(), keyOut.Bytes(), nil
}

func GenerateChildCert(commonName string, isClient bool, caCert *x509.Certificate, caKey *rsa.PrivateKey) ([]byte, []byte, error) {
	var template x509.Certificate
	privateKey, _ := rsa.GenerateKey(rand.Reader, RsaKeySize())
	subject := RandomSubject(commonName)
	notBefore := time.Now()
	days := randomInt(365) * -1
	notBefore = notBefore.AddDate(0, 0, days)
	notAfter := notBefore.Add(randomValidFor())

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, _ := rand.Int(rand.Reader, serialNumberLimit)
	if isClient {
		template = x509.Certificate{
			SerialNumber:          serialNumber,
			Subject:               *subject,
			NotBefore:             notBefore,
			NotAfter:              notAfter,
			KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
			ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			BasicConstraintsValid: false,
		}
	} else {
		template = x509.Certificate{
			SerialNumber:          serialNumber,
			Subject:               *subject,
			NotBefore:             notBefore,
			NotAfter:              notAfter,
			KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
			ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			BasicConstraintsValid: false,
		}

		if ip := net.ParseIP(subject.CommonName); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, subject.CommonName)
		}
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, caCert, publicKey(privateKey), caKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create certificate: %v", err)
	}

	certOut := bytes.NewBuffer([]byte{})
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	keyOut := bytes.NewBuffer([]byte{})
	pem.Encode(keyOut, pemBlockForKey(privateKey))

	return certOut.Bytes(), keyOut.Bytes(), nil
}
