package certutils

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"golang.org/x/crypto/acme/autocert"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	// ACMEDirName - Name of dir to store ACME certs
	ACMEDirName = "acme"
)

// ACMEServerManager manages global ACME HTTP server
type ACMEServerManager struct {
	server  *http.Server
	manager *autocert.Manager // Unified ACME manager
	running bool
	mutex   sync.RWMutex
	domains map[string]bool // Track registered domains
}

var (
	globalACMEServerManager *ACMEServerManager
	acmeManagerOnce         sync.Once
)

// GetACMEServerManager gets global ACME server manager instance
func GetACMEServerManager() *ACMEServerManager {
	acmeManagerOnce.Do(func() {
		globalACMEServerManager = &ACMEServerManager{
			domains: make(map[string]bool),
		}
	})
	return globalACMEServerManager
}

// Start starts ACME HTTP server
func (asm *ACMEServerManager) Start() error {
	asm.mutex.Lock()
	defer asm.mutex.Unlock()

	if asm.running {
		return nil // Server is already running
	}

	// Create unified ACME manager
	acmeDir := GetACMEDir()
	asm.manager = &autocert.Manager{
		Cache:  autocert.DirCache(acmeDir),
		Prompt: autocert.AcceptTOS,
		// Initially don't set HostPolicy, update dynamically later via UpdateHostPolicy
	}

	asm.server = &http.Server{
		Addr:         ":80",
		Handler:      asm.manager.HTTPHandler(nil),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		logs.Log.Infof("Starting ACME HTTP server on :80")
		if err := asm.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logs.Log.Errorf("ACME HTTP server error: %v", err)
			asm.mutex.Lock()
			asm.running = false
			asm.mutex.Unlock()
		}
	}()

	asm.running = true

	// Wait for server to start
	time.Sleep(1 * time.Second)

	logs.Log.Infof("ACME HTTP server started successfully")
	return nil
}

// Stop stops ACME HTTP server
func (asm *ACMEServerManager) Stop() error {
	asm.mutex.Lock()
	defer asm.mutex.Unlock()

	if !asm.running || asm.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := asm.server.Shutdown(ctx); err != nil {
		logs.Log.Warnf("Failed to shutdown ACME HTTP server: %v", err)
		return err
	}

	logs.Log.Infof("ACME HTTP server stopped successfully")
	asm.running = false
	asm.server = nil
	asm.manager = nil
	asm.domains = make(map[string]bool)

	return nil
}

// IsRunning checks if server is running
func (asm *ACMEServerManager) IsRunning() bool {
	asm.mutex.RLock()
	defer asm.mutex.RUnlock()
	return asm.running
}

// GetManager gets unified ACME manager
func (asm *ACMEServerManager) GetManager() *autocert.Manager {
	asm.mutex.RLock()
	defer asm.mutex.RUnlock()
	return asm.manager
}

// UpdateHostPolicy updates HostPolicy to support new domains
func (asm *ACMEServerManager) UpdateHostPolicy() {
	asm.mutex.RLock()
	domains := make([]string, 0, len(asm.domains))
	for domain := range asm.domains {
		domains = append(domains, domain)
	}
	asm.mutex.RUnlock()

	if asm.manager != nil && len(domains) > 0 {
		asm.manager.HostPolicy = autocert.HostWhitelist(domains...)
		logs.Log.Infof("Updated ACME HostPolicy with domains: %v", domains)
	}
}

// RegisterDomain registers domain and updates HostPolicy
func (asm *ACMEServerManager) RegisterDomain(domain string) {
	asm.mutex.Lock()
	asm.domains[domain] = true
	asm.mutex.Unlock()

	logs.Log.Infof("Registered domain '%s' for ACME", domain)

	// Update HostPolicy
	asm.UpdateHostPolicy()
}

// GetRegisteredDomains gets list of registered domains
func (asm *ACMEServerManager) GetRegisteredDomains() []string {
	asm.mutex.RLock()
	defer asm.mutex.RUnlock()

	domains := make([]string, 0, len(asm.domains))
	for domain := range asm.domains {
		domains = append(domains, domain)
	}
	return domains
}

// Restart restarts server
func (asm *ACMEServerManager) Restart() error {
	if err := asm.Stop(); err != nil {
		return fmt.Errorf("failed to stop ACME server: %v", err)
	}

	time.Sleep(2 * time.Second) // Wait for port to be released

	if err := asm.Start(); err != nil {
		return fmt.Errorf("failed to start ACME server: %v", err)
	}

	return nil
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

func GetAcmeTls(config *clientpb.TLS) (*types.TlsConfig, error) {
	config.Domain = filepath.Base(config.Domain)
	certPath := filepath.Join(GetACMEDir(), config.Domain+".crt")
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
			Domain: config.Domain,
			Acme:   config.Acme,
			Enable: config.Enable,
		}, nil
	}

	logs.Log.Infof("Attempting to fetch let's encrypt certificate for '%s' ...", config.Domain)

	// Get ACME server manager
	serverManager := GetACMEServerManager()

	// Start ACME HTTP server (if not already started)
	if !serverManager.IsRunning() {
		if err := serverManager.Start(); err != nil {
			return nil, err
		}
	}

	// Register domain (this will automatically update HostPolicy)
	serverManager.RegisterDomain(config.Domain)
	logs.Log.Debugf("Getting unified ACME manager")

	// Get unified ACME manager
	acmeManager := serverManager.GetManager()
	if acmeManager == nil {
		return nil, fmt.Errorf("ACME manager not available")
	}

	logs.Log.Debugf("Successfully got unified ACME manager")
	// Get certificate
	hello := &tls.ClientHelloInfo{ServerName: config.Domain}
	tlsCert, err := acmeManager.GetCertificate(hello)
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

	// Stop ACME HTTP server after certificate is successfully obtained
	err = serverManager.Stop()
	if err != nil {
		return nil, err
	}

	return &types.TlsConfig{
		Cert: &types.CertConfig{
			Cert: string(certPEMBytes),
			Key:  string(keyPEMBytes),
		},
		Domain: config.Domain,
		Acme:   config.Acme,
		Enable: config.Enable,
	}, nil
}
