package certutils

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"github.com/chainreactors/IoM-go/mtls"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/certs"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"path"
)

var certsLog = logs.Log

// CertFingerprint computes SHA-256 hex fingerprint from PEM-encoded certificate.
func CertFingerprint(certPEM []byte) (string, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return "", fmt.Errorf("failed to decode certificate PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse certificate: %w", err)
	}
	hash := sha256.Sum256(cert.Raw)
	return hex.EncodeToString(hash[:]), nil
}

// --------------------------------
//  Generic Certificate Functions
// --------------------------------

// GetOperatorServerMTLSConfig - Get the TLS config for the operator server
func GetOperatorServerMTLSConfig(host string) *tls.Config {
	caCert, _, err := GetCertificateAuthority()
	if err != nil {
		logs.Log.Errorf("Failed to load CA %s", err)
		return nil
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(caCert)
	certPEM, keyPEM, err := GenerateServerCert(host)
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

func GenerateRootCert() error {
	rootCertPath := path.Join(configs.CertsPath, certs.RootCert)
	rootKeyPath := path.Join(configs.CertsPath, certs.RootKey)
	if fileutils.Exist(rootCertPath) && fileutils.Exist(rootKeyPath) {
		return nil
	}
	cert, key, err := certs.GenerateCACert(certs.RootName, nil)
	if err != nil {
		return err
	}
	err = certs.SaveToPEMFile(rootCertPath, cert)
	if err != nil {
		return err
	}
	err = certs.SaveToPEMFile(rootKeyPath, key)
	if err != nil {
		return err
	}
	if err != nil {
		return err
	}
	return nil
}

func GenerateServerCert(name string) ([]byte, []byte, error) {
	certPath := path.Join(configs.CertsPath, certs.ServerCert)
	certKeyPath := path.Join(configs.CertsPath, certs.ServerKey)
	ca, caKey, err := GetCertificateAuthority()
	if err != nil {
		return nil, nil, err
	}
	cert, key, err := certs.GenerateChildCert(name, false, ca, caKey)
	if err != nil {
		return nil, nil, err
	}
	err = certs.SaveToPEMFile(certPath, cert)
	if err != nil {
		return nil, nil, err
	}
	err = certs.SaveToPEMFile(certKeyPath, key)
	if err != nil {
		return nil, nil, err
	}
	return cert, key, nil
}

func GenerateClientCert(host, name string, port int) (*mtls.ClientConfig, string, error) {
	ca, caKey, err := GetCertificateAuthority()
	caCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca.Raw})
	if err != nil {
		return nil, "", err
	}
	cert, key, err := certs.GenerateChildCert(name, true, ca, caKey)
	if err != nil {
		return nil, "", err
	}
	fp, err := CertFingerprint(cert)
	if err != nil {
		return nil, "", err
	}
	return &mtls.ClientConfig{
		Operator:      name,
		Host:          host,
		Port:          port,
		Type:          mtls.Client,
		CACertificate: string(caCert),
		PrivateKey:    string(key),
		Certificate:   string(cert),
	}, fp, nil
}

func GenerateListenerCert(host, name string, port int) (*mtls.ClientConfig, string, error) {
	var cert, key []byte
	ca, caKey, err := GetCertificateAuthority()
	caCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca.Raw})
	if err != nil {
		return nil, "", err
	}
	cert, key, err = certs.GenerateChildCert(host, true, ca, caKey)
	if err != nil {
		return nil, "", err
	}
	fp, err := CertFingerprint(cert)
	if err != nil {
		return nil, "", err
	}
	return &mtls.ClientConfig{
		Operator:      name,
		Host:          host,
		Port:          port,
		Type:          mtls.Listener,
		CACertificate: string(caCert),
		PrivateKey:    string(key),
		Certificate:   string(cert),
	}, fp, nil
}

func GenerateSelfTLS(name string, certsSubject *clientpb.CertificateSubject) (*clientpb.TLS, error) {
	subject := CertificateSubjectToPkixName(certsSubject)
	caCertByte, caKeyByte, err := certs.GenerateCACert(name, subject)
	if err != nil {
		return nil, err
	}
	caCert, caKey, err := ParseCertificateAuthority(caCertByte, caKeyByte)
	if err != nil {
		return nil, err
	}
	certByte, keyByte, err := certs.GenerateChildCert(name, false, caCert, caKey)
	if err != nil {
		return nil, err
	}
	return &clientpb.TLS{
		Cert: &clientpb.Cert{
			Cert: string(certByte),
			Key:  string(keyByte),
		},
		Ca: &clientpb.Cert{
			Cert: string(caCertByte),
			Key:  string(caKeyByte),
		},
		Enable: true,
		Acme:   false,
	}, nil
}

func CertificateSubjectToPkixName(subject *clientpb.CertificateSubject) *pkix.Name {
	if subject == nil {
		return nil
	}

	var organizations []string
	if subject.O != "" {
		organizations = append(organizations, subject.O)
	}

	var organizationalUnits []string
	if subject.Ou != "" {
		organizationalUnits = append(organizationalUnits, subject.Ou)
	}

	var countries []string
	if subject.C != "" {
		countries = append(countries, subject.C)
	}

	var provinces []string
	if subject.St != "" {
		provinces = append(provinces, subject.St)
	}

	var localities []string
	if subject.L != "" {
		localities = append(localities, subject.L)
	}

	return &pkix.Name{
		CommonName:         subject.Cn,
		Organization:       organizations,
		OrganizationalUnit: organizationalUnits,
		Country:            countries,
		Province:           provinces,
		Locality:           localities,
	}
}
