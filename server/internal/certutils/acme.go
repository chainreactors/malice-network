package certutils

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"golang.org/x/crypto/acme/autocert"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	// ACMEDirName - Name of dir to store ACME certs
	ACMEDirName = "acme"
)

// GetACMEDir - Dir to store ACME certs
func GetACMEDir() string {
	dir, err := os.Getwd()
	if err != nil {
		logs.Log.Errorf("Failed to get work dir %s", err)
		return ""
	}
	acmePath := filepath.Join(dir, ACMEDirName)
	if _, err := os.Stat(acmePath); os.IsNotExist(err) {
		logs.Log.Infof("[mkdir] %s", acmePath)
		os.MkdirAll(acmePath, 0700)
	}
	return acmePath
}

// GetACMEManager - Get an ACME cert/tls config with the certs
func GetACMEManager(domain string) *autocert.Manager {
	acmeDir := GetACMEDir()
	return &autocert.Manager{
		Cache:  autocert.DirCache(acmeDir),
		Prompt: autocert.AcceptTOS,
	}
}

func isCertValid(certPath string) bool {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return false
	}
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return false
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return false
	}
	return cert.NotAfter.After(time.Now())
}

func GetAutoCertTls(config *clientpb.TLS) (*types.TlsConfig, error) {
	config.Domain = filepath.Base(config.Domain)
	certPath := filepath.Join(GetACMEDir(), config.Domain)
	keyPath := filepath.Join(GetACMEDir(), config.Domain+".key")

	if fileutils.Exist(certPath) && fileutils.Exist(keyPath) && isCertValid(certPath) {
		certPEM, _ := os.ReadFile(certPath)
		keyPEM, _ := os.ReadFile(keyPath)
		config.Cert = &clientpb.Cert{
			Cert: string(certPEM),
			Key:  string(keyPEM),
		}
		return &types.TlsConfig{
			Cert: &types.CertConfig{
				Cert: string(certPEM),
				Key:  string(keyPEM),
			},
			Domain:   config.Domain,
			AutoCert: config.AutoCert,
			Enable:   config.Enable,
		}, nil
	}

	logs.Log.Infof("Attempting to fetch let's encrypt certificate for '%s' ...", config.Domain)
	acmeManager := GetACMEManager(config.Domain)
	acmeHTTPServer := &http.Server{Addr: ":80", Handler: acmeManager.HTTPHandler(nil)}

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := acmeHTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logs.Log.Warnf("ACME HTTP server error: %v", err)
		}
	}()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := acmeHTTPServer.Shutdown(shutdownCtx); err != nil {
		logs.Log.Warnf("Failed to shutdown acmeHTTPServer: %v", err)
	}
	hello := &tls.ClientHelloInfo{ServerName: config.Domain}
	tlsCert, err := acmeManager.GetCertificate(hello)
	if err != nil {
		return nil, err
	}

	var certPEMBytes []byte
	for _, cert := range tlsCert.Certificate {
		certPEMBytes = append(certPEMBytes, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert})...)
	}

	keyBytes, err := x509.MarshalPKCS8PrivateKey(tlsCert.PrivateKey)
	if err != nil {
		return nil, err
	}
	keyPEMBytes := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes})

	return &types.TlsConfig{
		Cert: &types.CertConfig{
			Cert: string(certPEMBytes),
			Key:  string(keyPEMBytes),
		},
		Domain:   config.Domain,
		AutoCert: config.AutoCert,
		Enable:   config.Enable,
	}, nil
}
