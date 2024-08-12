package certs

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/chainreactors/files"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/helper"
	"github.com/chainreactors/malice-network/helper/mtls"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"math/big"
	insecureRand "math/rand"
	"net"
	"os"
	"path"
	"time"
)

const (
	// ECCKey - Namespace for ECC keys
	ECCKey = "ecc"

	// RSAKey - Namespace for RSA keys
	RSAKey       = "rsa"
	RootName     = "Root"
	OperatorName = "server.operator"
	ListenerName = "default"
)

var (
	// ErrCertDoesNotExist - Returned if a GetCertificate() is called for a cert/cn that does not exist
	ErrCertDoesNotExist = errors.New("certificate does not exist")
)

// saveCertificate - Save the certificate and the key to the filesystem
func saveCertificate(caType int, keyType string, commonName string, cert []byte, key []byte) error {

	if keyType != ECCKey && keyType != RSAKey {
		return fmt.Errorf("invalid key type '%s'", keyType)
	}

	certsLog.Debugf("Saving certificate for cn = '%s'", commonName)

	certModel := &models.Certificate{
		CommonName:     commonName,
		CAType:         caType,
		KeyType:        keyType,
		CertificatePEM: string(cert),
		PrivateKeyPEM:  string(key),
	}
	err := db.DeleteCertificate(commonName)
	if err != nil {
		return err
	}
	createResult := db.SaveCertificate(certModel)

	return createResult
}

// SaveToPEMFile 将 PEM 格式数据保存到文件
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

// GetECCCertificate - Get an ECC certificate
func GetECCCertificate(caType int, commonName string) ([]byte, []byte, error) {
	return GetCertificate(caType, ECCKey, commonName)
}

// GetRSACertificate - Get an RSA certificate
func GetRSACertificate(caType int, commonName string) ([]byte, []byte, error) {
	return GetCertificate(caType, RSAKey, commonName)
}

// GetCertificate - Get the PEM encoded certificate & key for a host
func GetCertificate(caType int, keyType string, commonName string) ([]byte, []byte, error) {

	if keyType != ECCKey && keyType != RSAKey {
		return nil, nil, fmt.Errorf("Invalid key type '%s'", keyType)
	}

	certsLog.Infof("Getting certificate ca type = %s, cn = '%s'", caType, commonName)

	certModel := models.Certificate{}
	dbSession := db.Session()
	result := dbSession.Where(&models.Certificate{
		CAType:     caType,
		KeyType:    keyType,
		CommonName: commonName,
	}).First(&certModel)
	if errors.Is(result.Error, db.ErrRecordNotFound) {
		return nil, nil, ErrCertDoesNotExist
	}
	if result.Error != nil {
		return nil, nil, result.Error
	}

	return []byte(certModel.CertificatePEM), []byte(certModel.PrivateKeyPEM), nil
}

// RemoveCertificate - Remove a certificate from the cert store
func RemoveCertificate(caType int, keyType string, commonName string) error {
	var err error
	if keyType != RSAKey {
		return fmt.Errorf("invalid key type '%s'", keyType)
	}
	dbSession := db.Session()
	if caType == ListenerCA {
		err = dbSession.Where(&models.Listener{
			Name: commonName,
		}).Delete(&models.Listener{}).Error
		commonName = "listener." + commonName
	} else {
		err = dbSession.Where(&models.Operator{
			Name: commonName,
		}).Delete(&models.Operator{}).Error
	}
	err = dbSession.Where(&models.Certificate{
		CAType:     caType,
		KeyType:    keyType,
		CommonName: commonName,
	}).Delete(&models.Certificate{}).Error
	if err != nil {
		return err
	}

	return err
}

// --------------------------------
//  Generic Certificate Functions
// --------------------------------

// GenerateECCCertificate - Generate a TLS certificate with the given parameters
// We choose some reasonable defaults like Curve, Key Size, ValidFor, etc.
// Returns two strings `cert` and `key` (PEM Encoded).
func GenerateECCCertificate(caType int, commonName string, isCA bool, isClient bool) ([]byte, []byte) {

	certsLog.Infof("Generating TLS certificate (ECC) for '%s' ...", commonName)

	var privateKey interface{}
	var err error

	// Generate private key
	curves := []elliptic.Curve{elliptic.P521(), elliptic.P384(), elliptic.P256()}
	curve := curves[randomInt(len(curves))]
	privateKey, err = ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		certsLog.Errorf("Failed to generate private key: %v", err)
	}
	subject := &pkix.Name{
		CommonName: commonName,
	}
	return generateCertificate(caType, subject, isCA, isClient, privateKey)
}

func GenerateListenerCertificate(config *configs.TlsConfig) ([]byte, []byte, error) {
	if files.IsExist(config.CertFile) && files.IsExist(config.KeyFile) {
		cert, err := os.ReadFile(config.CertFile)
		if err != nil {
			return nil, nil, err
		}
		key, err := os.ReadFile(config.KeyFile)
		if err != nil {
			return nil, nil, err
		}
		return cert, key, nil
	} else {
		cert, key := GenerateRSACertificate(ImplantCA, "", true, false, config.ToPkix())
		err := os.WriteFile(path.Join(configs.ListenerPath, config.Name+"_crt.pem"), cert, 0644)
		if err != nil {
			return nil, nil, err
		}
		err = os.WriteFile(path.Join(configs.ListenerPath, config.Name+"_key.pem"), key, 0644)
		if err != nil {
			return nil, nil, err
		}
		logs.Log.Importantf("generate implant ca , save crt to %s", path.Join(configs.ListenerPath, config.Name+"_crt.pem"))
		return cert, key, nil
	}
}

// GenerateRSACertificate - Generates an RSA Certificate
func GenerateRSACertificate(caType int, commonName string, isCA bool, isClient bool,
	subject *pkix.Name) ([]byte, []byte) {
	certsLog.Debugf("Generating TLS certificate (RSA) for '%s' ...", commonName)

	//var subject *pkix.Name
	// Generate private key
	privateKey, _ := rsa.GenerateKey(rand.Reader, RsaKeySize())

	// Generate random listener subject if listener config is null
	if caType == ListenerCA && subject == nil {
		subject = randomSubject(commonName)
	} else {
		subject = randomSubject(commonName)
	}
	return generateCertificate(caType, subject, isCA, isClient, privateKey)
}

func generateCertificate(caType int, subject *pkix.Name, isCA bool, isClient bool, privateKey interface{}) ([]byte, []byte) {
	// Valid times, subtract random days from .Now()
	notBefore := time.Now()
	days := randomInt(365) * -1 // Within -1 year
	notBefore = notBefore.AddDate(0, 0, days)
	notAfter := notBefore.Add(randomValidFor())
	certsLog.Debugf("Valid from %v to %v", notBefore, notAfter)

	// Serial number
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, _ := rand.Int(rand.Reader, serialNumberLimit)
	certsLog.Debugf("Serial Number: %d", serialNumber)

	var keyUsage x509.KeyUsage = x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature
	var extKeyUsage []x509.ExtKeyUsage

	if isCA {
		certsLog.Debug("Authority certificate")
		keyUsage = x509.KeyUsageCertSign | x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature
		extKeyUsage = []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth,
		}
	} else if isClient {
		certsLog.Debug("Client authentication certificate")
		extKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	} else {
		certsLog.Debug("Server authentication certificate")
		extKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	}
	certsLog.Debugf("ExtKeyUsage = %v", extKeyUsage)

	// Certificate template
	template := x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               *subject,
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              keyUsage,
		ExtKeyUsage:           extKeyUsage,
		BasicConstraintsValid: isCA,
	}

	if !isClient {
		// Host or IP address
		if ip := net.ParseIP(subject.CommonName); ip != nil {
			certsLog.Debugf("Certificate authenticates IP address: %v", ip)
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			certsLog.Debugf("Certificate authenticates host: %v", subject.CommonName)
			template.DNSNames = append(template.DNSNames, subject.CommonName)
		}
	} else {
		certsLog.Debugf("Client certificate authenticates CN: %v", subject.CommonName)
	}

	// Sign certificate or self-sign if CA
	var certErr error
	var derBytes []byte
	if isCA {
		certsLog.Debugf("Certificate is an AUTHORITY")
		template.IsCA = true
		template.KeyUsage |= x509.KeyUsageCertSign
		derBytes, certErr = x509.CreateCertificate(rand.Reader, &template, &template, publicKey(privateKey), privateKey)
	} else {
		caCert, caKey, err := GetCertificateAuthority() // Sign the new certificate with our CA
		if err != nil {
			certsLog.Errorf("Invalid ca type (%d): %v", caType, err)
		}
		derBytes, certErr = x509.CreateCertificate(rand.Reader, &template, caCert, publicKey(privateKey), caKey)
	}
	if certErr != nil {
		// We maybe don't want this to be fatal, but it should basically never happen afaik
		certsLog.Errorf("Failed to create certificate: %v", certErr)
	}

	// Encode certificate and key
	certOut := bytes.NewBuffer([]byte{})
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	keyOut := bytes.NewBuffer([]byte{})
	pem.Encode(keyOut, pemBlockForKey(privateKey))

	return certOut.Bytes(), keyOut.Bytes()
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
			certsLog.Errorf("Unable to marshal ECDSA private key: %v", err)
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

// removeOldCerts - Remove old certificates from the filesystem
func removeOldCerts(cfgPath string) error {
	if _, err := os.Stat(cfgPath); err == nil {
		if err := os.Remove(cfgPath); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	err := db.DeleteAllCertificates()
	if err != nil {
		return err
	}
	certConfigPath := configs.CertsPath
	if _, err := os.Stat(certConfigPath); os.IsNotExist(err) {
		return nil
	}
	files, err := os.ReadDir(certConfigPath)
	if err != nil {
		return err
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if !helper.FileExists(path.Join(certConfigPath, file.Name())) {
			continue
		}
		if err := os.Remove(path.Join(certConfigPath, file.Name())); err != nil {
			return err
		}
	}

	return nil
}

func CheckCertIsExist(certPath, keyPath, commonName string, caType int) ([]byte, []byte, error) {
	var existingCert models.Certificate
	var certBytes, keyBytes []byte
	var err error
	if caType == ListenerCA {
		configDir, _ := os.Getwd()
		configPath := path.Join(configDir, fmt.Sprintf("%s.yaml", commonName))
		listener, err := mtls.ReadConfig(configPath)
		if err != nil {
			return nil, nil, err
		}
		certBytes = []byte(listener.Certificate)
		keyBytes = []byte(listener.PrivateKey)
	} else if caType == OperatorCA {
		certBytes, err = os.ReadFile(certPath)
		if err != nil {
			return nil, nil, err
		}
		keyBytes, err = os.ReadFile(keyPath)
		if err != nil {
			return nil, nil, err
		}
	}
	dbSession := db.Session()
	result := dbSession.Where("common_name = ?", commonName).First(&existingCert).Error
	if result != nil {
		err := saveCertificate(caType, RSAKey, commonName, certBytes, keyBytes)
		if err != nil {
			return nil, nil, err
		}
	}
	return certBytes, keyBytes, nil
}

// GetOperatorServerMTLSConfig - Get the TLS config for the operator server
func GetOperatorServerMTLSConfig(host string) *tls.Config {
	caCert, _, err := GetCertificateAuthority()
	if err != nil {
		logs.Log.Errorf("Failed to load CA %s", err)
		return nil
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(caCert)
	certPEM, keyPEM, err := ServerGenerateCertificate(host, false, "")
	if err != nil {
		logs.Log.Errorf("Failed to load certificate %s", err)
	}
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		logs.Log.Errorf("Error loading server certificate: %v", err)
	}

	tlsConfig := &tls.Config{
		RootCAs:      caCertPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caCertPool,
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
	}

	return tlsConfig
}
