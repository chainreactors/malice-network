package certutils

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/providers/dns/alidns"
	"github.com/go-acme/lego/v4/providers/dns/cloudflare"
	"github.com/go-acme/lego/v4/providers/dns/dnspod"
	"github.com/go-acme/lego/v4/providers/dns/route53"
	"github.com/go-acme/lego/v4/registration"
)

const (
	acmeAccountKeyFile  = "acme_account_key.pem"
	acmeAccountInfoFile = "acme_account.json"
)

var SupportedProviders = []string{"cloudflare", "alidns", "dnspod", "route53"}

// AcmeUser implements registration.User interface for lego
type AcmeUser struct {
	Email        string                 `json:"email"`
	Registration *registration.Resource `json:"registration"`
	key          crypto.PrivateKey
}

func (u *AcmeUser) GetEmail() string {
	return u.Email
}

func (u *AcmeUser) GetRegistration() *registration.Resource {
	return u.Registration
}

func (u *AcmeUser) GetPrivateKey() crypto.PrivateKey {
	return u.key
}

// loadOrCreateAccount loads an existing ACME account or creates a new one
func loadOrCreateAccount(email string) (*AcmeUser, error) {
	certsDir := configs.GetCertDir()
	keyPath := filepath.Join(certsDir, acmeAccountKeyFile)
	infoPath := filepath.Join(certsDir, acmeAccountInfoFile)

	user := &AcmeUser{}

	// Try to load existing account key
	keyPEM, err := os.ReadFile(keyPath)
	if err == nil {
		block, _ := pem.Decode(keyPEM)
		if block != nil {
			privKey, parseErr := x509.ParseECPrivateKey(block.Bytes)
			if parseErr == nil {
				user.key = privKey

				// Try to load account info
				infoData, infoErr := os.ReadFile(infoPath)
				if infoErr == nil {
					json.Unmarshal(infoData, user)
				}

				// Update email if different
				if email != "" && user.Email != email {
					user.Email = email
				}

				if user.Registration != nil {
					return user, nil
				}
			}
		}
	}

	// Generate new key if needed
	if user.key == nil {
		privKey, genErr := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if genErr != nil {
			return nil, fmt.Errorf("failed to generate ACME account key: %w", genErr)
		}
		user.key = privKey

		// Save key
		keyBytes, marshalErr := x509.MarshalECPrivateKey(privKey)
		if marshalErr != nil {
			return nil, fmt.Errorf("failed to marshal ACME account key: %w", marshalErr)
		}
		keyPemBlock := pem.EncodeToMemory(&pem.Block{
			Type:  "EC PRIVATE KEY",
			Bytes: keyBytes,
		})
		if writeErr := os.WriteFile(keyPath, keyPemBlock, 0600); writeErr != nil {
			return nil, fmt.Errorf("failed to save ACME account key: %w", writeErr)
		}
	}

	if email != "" {
		user.Email = email
	}

	return user, nil
}

// saveAccount persists the ACME account info to disk
func saveAccount(user *AcmeUser) error {
	certsDir := configs.GetCertDir()
	infoPath := filepath.Join(certsDir, acmeAccountInfoFile)

	data, err := json.MarshalIndent(user, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal account info: %w", err)
	}
	return os.WriteFile(infoPath, data, 0600)
}

// NewDNSProvider creates a DNS challenge provider based on provider name and credentials
func NewDNSProvider(providerName string, credentials map[string]string) (challenge.Provider, error) {
	switch providerName {
	case "cloudflare":
		cfg := cloudflare.NewDefaultConfig()
		if token, ok := credentials["api_token"]; ok && token != "" {
			cfg.AuthToken = token
		} else {
			cfg.AuthEmail = credentials["api_email"]
			cfg.AuthKey = credentials["api_key"]
		}
		return cloudflare.NewDNSProviderConfig(cfg)

	case "alidns":
		cfg := alidns.NewDefaultConfig()
		cfg.APIKey = credentials["access_key"]
		cfg.SecretKey = credentials["secret_key"]
		if region, ok := credentials["region"]; ok && region != "" {
			cfg.RegionID = region
		}
		return alidns.NewDNSProviderConfig(cfg)

	case "dnspod":
		cfg := dnspod.NewDefaultConfig()
		cfg.LoginToken = credentials["api_id"] + "," + credentials["api_token"]
		return dnspod.NewDNSProviderConfig(cfg)

	case "route53":
		cfg := route53.NewDefaultConfig()
		cfg.AccessKeyID = credentials["access_key_id"]
		cfg.SecretAccessKey = credentials["secret_access_key"]
		if region, ok := credentials["region"]; ok && region != "" {
			cfg.Region = region
		}
		return route53.NewDNSProviderConfig(cfg)

	default:
		return nil, fmt.Errorf("unsupported DNS provider: %s (supported: %v)", providerName, SupportedProviders)
	}
}

// ObtainCert obtains a certificate for the given domain using DNS-01 challenge via lego.
// Parameters from the request take precedence over server config defaults.
func ObtainCert(domain, providerName, email, caURL string, credentials map[string]string) (certPEM, keyPEM []byte, err error) {
	// Merge with server config defaults
	acmeCfg := configs.GetAcmeConfig()
	if acmeCfg != nil {
		if providerName == "" {
			providerName = acmeCfg.Provider
		}
		if email == "" {
			email = acmeCfg.Email
		}
		if caURL == "" {
			caURL = acmeCfg.CAUrl
		}
		if len(credentials) == 0 && len(acmeCfg.Credentials) > 0 {
			credentials = acmeCfg.Credentials
		}
	}

	if caURL == "" {
		caURL = lego.LEDirectoryProduction
	}

	if providerName == "" {
		return nil, nil, fmt.Errorf("DNS provider is required, set via --provider or server acme config")
	}

	// Load or create ACME account
	user, err := loadOrCreateAccount(email)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load/create ACME account: %w", err)
	}

	// Create lego config
	config := lego.NewConfig(user)
	config.CADirURL = caURL
	config.UserAgent = "malice-network/1.0"

	// Create lego client
	client, err := lego.NewClient(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create ACME client: %w", err)
	}

	// Create and set DNS provider
	dnsProvider, err := NewDNSProvider(providerName, credentials)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create DNS provider: %w", err)
	}

	err = client.Challenge.SetDNS01Provider(dnsProvider)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to set DNS-01 provider: %w", err)
	}

	// Register account if needed
	if user.Registration == nil {
		logs.Log.Infof("Registering ACME account with email: %s", user.Email)
		reg, regErr := client.Registration.Register(registration.RegisterOptions{
			TermsOfServiceAgreed: true,
		})
		if regErr != nil {
			return nil, nil, fmt.Errorf("failed to register ACME account: %w", regErr)
		}
		user.Registration = reg
		if saveErr := saveAccount(user); saveErr != nil {
			logs.Log.Warnf("Failed to save ACME account info: %v", saveErr)
		}
	}

	// Obtain certificate
	logs.Log.Infof("Requesting certificate for domain: %s (provider: %s, CA: %s)", domain, providerName, caURL)
	request := certificate.ObtainRequest{
		Domains: []string{domain},
		Bundle:  true,
	}

	certificates, err := client.Certificate.Obtain(request)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to obtain certificate for %s: %w", domain, err)
	}

	logs.Log.Infof("Successfully obtained certificate for domain: %s", domain)
	return certificates.Certificate, certificates.PrivateKey, nil
}
