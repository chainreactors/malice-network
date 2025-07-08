package certutils

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"golang.org/x/crypto/acme/autocert"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	// ACMEDirName - Name of dir to store ACME certs
	ACMEDirName  = "acme"
	ACMERootPath = "/.well-known/acme-challenge/"
)

type ACMEManager struct {
	manager *autocert.Manager
	domains map[string]bool
	mutex   sync.RWMutex
}

var (
	globalACMEManager *ACMEManager
	acmeOnce          sync.Once
)

func GetACMEManager() *ACMEManager {
	acmeOnce.Do(func() {
		globalACMEManager = &ACMEManager{
			domains: make(map[string]bool),
		}
		acmeDir := GetACMEDir()
		cacheDir := filepath.Join(acmeDir, "cache")
		globalACMEManager.manager = &autocert.Manager{
			Cache:      autocert.DirCache(cacheDir),
			Prompt:     autocert.AcceptTOS,
			HostPolicy: globalACMEManager.hostPolicy,
		}
	})
	return globalACMEManager
}

func (a *ACMEManager) RegisterDomain(domain string) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.domains[domain] = true
	logs.Log.Infof("Registered domain '%s' for ACME", domain)
}

func (a *ACMEManager) hostPolicy(ctx context.Context, host string) error {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	if a.domains[host] {
		return nil
	}
	return fmt.Errorf("host %s not allowed for ACME", host)
}

func (a *ACMEManager) GetManager() *autocert.Manager {
	return a.manager
}

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

func GetAcmeTls(config *clientpb.TLS) (*clientpb.TLS, error) {
	config.Domain = filepath.Base(config.Domain)
	acmeMgr := GetACMEManager()

	// mkdir domain dir
	domainDir := filepath.Join(GetACMEDir(), config.Domain)
	if _, err := os.Stat(domainDir); os.IsNotExist(err) {
		err = os.MkdirAll(domainDir, 0700)
		if err != nil {
			logs.Log.Errorf("Failed to create domain dir: %v", err)
		}
	}
	certPath := filepath.Join(domainDir, config.Domain+".crt")
	keyPath := filepath.Join(domainDir, config.Domain+".key")

	if fileutils.Exist(certPath) && fileutils.Exist(keyPath) && isCertValid(certPath) {
		certPEM, _ := os.ReadFile(certPath)
		keyPEM, _ := os.ReadFile(keyPath)
		return &clientpb.TLS{
			Cert: &clientpb.Cert{
				Cert: string(certPEM),
				Key:  string(keyPEM),
			},
			Domain: config.Domain,
			Acme:   config.Acme,
			Enable: config.Enable,
		}, nil
	}

	logs.Log.Infof("Attempting to fetch let's encrypt certificate for '%s' ...", config.Domain)

	hello := &tls.ClientHelloInfo{
		ServerName: config.Domain,
		//SupportedProtos: []string{acme.ALPNProto},
	}
	tlsCert, err := acmeMgr.manager.GetCertificate(hello)
	if err != nil {
		logs.Log.Errorf("Failed to get certificate for domain '%s': %v", config.Domain, err)
		return nil, err
	}

	logs.Log.Infof("Successfully obtained certificate for domain '%s'", config.Domain)

	var certPEMBytes []byte
	for _, cert := range tlsCert.Certificate {
		certPEMBytes = append(certPEMBytes, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert})...)
	}

	keyBytes, err := x509.MarshalPKCS8PrivateKey(tlsCert.PrivateKey)
	if err != nil {
		return nil, err
	}
	keyPEMBytes := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes})

	// Save certificate to local filesystem
	if saveErr := os.WriteFile(certPath, certPEMBytes, 0600); saveErr != nil {
		logs.Log.Warnf("Failed to save certificate to %s: %v", certPath, saveErr)
	}

	if saveErr := os.WriteFile(keyPath, keyPEMBytes, 0600); saveErr != nil {
		logs.Log.Warnf("Failed to save private key to %s: %v", keyPath, saveErr)
	}
	return &clientpb.TLS{
		Cert: &clientpb.Cert{
			Cert: string(certPEMBytes),
			Key:  string(keyPEMBytes),
		},
		Domain: config.Domain,
		Acme:   config.Acme,
		Enable: config.Enable,
	}, nil

}
