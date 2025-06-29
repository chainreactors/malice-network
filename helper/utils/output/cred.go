package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/parsers"
	"regexp"
	"strings"
)

var (
	UPCredential    = "UP"
	TOKENCredential = "token"
	CERTCredential  = "cert"
)

func ParseZombie(content []byte) ([]*CredentialContext, error) {
	var res []*CredentialContext
	for _, b := range bytes.Split(content, []byte{'\n'}) {
		var r *parsers.ZombieResult
		if len(b) == 0 {
			continue
		}
		err := json.Unmarshal(b, &r)
		if err != nil {
			return nil, err
		}
		res = append(res, &CredentialContext{
			Target:         r.URI(),
			CredentialType: UPCredential,
			Params: map[string]string{
				"username": r.Username,
				"password": r.Password,
			},
		})
	}
	return res, nil
}

// ParseMimikatz parses mimikatz sekurlsa::logonpasswords output
func ParseMimikatz(content []byte) ([]*CredentialContext, error) {
	var res []*CredentialContext

	// Convert to string and split into lines
	output := string(content)
	lines := strings.Split(output, "\n")

	// Regular expressions for parsing
	authIdRegex := regexp.MustCompile(`Authentication Id\s*:\s*(.+)`)
	sessionRegex := regexp.MustCompile(`Session\s*:\s*(.+)`)
	userNameRegex := regexp.MustCompile(`User Name\s*:\s*(.+)`)
	domainRegex := regexp.MustCompile(`Domain\s*:\s*(.+)`)
	logonTimeRegex := regexp.MustCompile(`Logon Time\s*:\s*(.+)`)

	// Credential field regexes (with leading whitespace and asterisk)
	credUsernameRegex := regexp.MustCompile(`\s*\*\s*Username\s*:\s*(.+)`)
	credDomainRegex := regexp.MustCompile(`\s*\*\s*Domain\s*:\s*(.+)`)
	credPasswordRegex := regexp.MustCompile(`\s*\*\s*Password\s*:\s*(.+)`)
	credNTLMRegex := regexp.MustCompile(`\s*\*\s*NTLM\s*:\s*(.+)`)

	// Map for deduplication: key = "username:password", value = bool
	seen := make(map[string]bool)

	var currentAuth *authBlock

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check for new authentication block
		if matches := authIdRegex.FindStringSubmatch(line); matches != nil {
			// Save previous auth block if it has valid credentials
			if currentAuth != nil && currentAuth.hasValidCredentials() {
				if cred := currentAuth.toCredentialContext(); cred != nil {
					// Create deduplication key
					dedupKey := fmt.Sprintf("%s:%s", cred.Params["username"], cred.Params["password"])
					if !seen[dedupKey] {
						seen[dedupKey] = true
						res = append(res, cred)
					}
				}
			}

			// Start new auth block
			currentAuth = &authBlock{
				authId: strings.TrimSpace(matches[1]),
			}
			continue
		}

		if currentAuth == nil {
			continue
		}

		// Parse session info
		if matches := sessionRegex.FindStringSubmatch(line); matches != nil {
			currentAuth.session = strings.TrimSpace(matches[1])
		} else if matches := userNameRegex.FindStringSubmatch(line); matches != nil {
			username := strings.TrimSpace(matches[1])
			if isValidValue(username) {
				currentAuth.username = username
			}
		} else if matches := domainRegex.FindStringSubmatch(line); matches != nil {
			domain := strings.TrimSpace(matches[1])
			if isValidValue(domain) {
				currentAuth.domain = domain
			}
		} else if matches := logonTimeRegex.FindStringSubmatch(line); matches != nil {
			currentAuth.logonTime = strings.TrimSpace(matches[1])
		}

		// Parse credential fields (from protocol sections)
		if matches := credUsernameRegex.FindStringSubmatch(line); matches != nil {
			username := strings.TrimSpace(matches[1])
			if isValidValue(username) {
				currentAuth.credUsername = username
			}
		} else if matches := credDomainRegex.FindStringSubmatch(line); matches != nil {
			domain := strings.TrimSpace(matches[1])
			if isValidValue(domain) {
				currentAuth.credDomain = domain
			}
		} else if matches := credPasswordRegex.FindStringSubmatch(line); matches != nil {
			password := strings.TrimSpace(matches[1])
			if isValidValue(password) {
				currentAuth.password = password
			}
		} else if matches := credNTLMRegex.FindStringSubmatch(line); matches != nil {
			ntlm := strings.TrimSpace(matches[1])
			if isValidValue(ntlm) {
				currentAuth.ntlm = ntlm
			}
		}
	}

	// Don't forget the last auth block
	if currentAuth != nil && currentAuth.hasValidCredentials() {
		if cred := currentAuth.toCredentialContext(); cred != nil {
			// Create deduplication key
			dedupKey := fmt.Sprintf("%s:%s", cred.Params["username"], cred.Params["password"])
			if !seen[dedupKey] {
				seen[dedupKey] = true
				res = append(res, cred)
			}
		}
	}

	return res, nil
}

// authBlock represents a single authentication session block from mimikatz output
type authBlock struct {
	authId       string
	session      string
	username     string
	domain       string
	logonTime    string
	credUsername string
	credDomain   string
	password     string
	ntlm         string
}

// isValidValue checks if a credential value is valid (not null, empty, or placeholder)
func isValidValue(value string) bool {
	if value == "" || value == "(null)" || value == "null" {
		return false
	}
	return true
}

// hasValidCredentials checks if the auth block has extractable credentials
func (a *authBlock) hasValidCredentials() bool {
	// Must have username and domain (either from session info or credential fields)
	username := a.credUsername
	if username == "" {
		username = a.username
	}

	domain := a.credDomain
	if domain == "" {
		domain = a.domain
	}

	// Must have either password or NTLM hash
	hasPassword := isValidValue(a.password) || isValidValue(a.ntlm)

	return isValidValue(username) && isValidValue(domain) && hasPassword
}

// toCredentialContext converts authBlock to CredentialContext
func (a *authBlock) toCredentialContext() *CredentialContext {
	if !a.hasValidCredentials() {
		return nil
	}

	// Prefer credential fields over session fields
	username := a.credUsername
	if username == "" {
		username = a.username
	}

	domain := a.credDomain
	if domain == "" {
		domain = a.domain
	}

	// Prefer password over NTLM, but both are treated as "password"
	password := a.password
	if password == "" {
		password = a.ntlm
	}

	target := fmt.Sprintf("%s\\%s", domain, username)

	params := map[string]string{
		"username": username,
		"password": password,
		"domain":   domain,
	}

	// Add optional fields if available
	if a.authId != "" {
		params["auth_id"] = a.authId
	}
	if a.session != "" {
		params["session"] = a.session
	}
	if a.logonTime != "" {
		params["logon_time"] = a.logonTime
	}

	// Distinguish between password and NTLM hash in params
	if a.password != "" && a.ntlm != "" {
		params["ntlm"] = a.ntlm
	} else if a.ntlm != "" && a.password == "" {
		params["credential_type"] = "ntlm"
	}

	return &CredentialContext{
		Target:         target,
		CredentialType: UPCredential,
		Params:         params,
	}
}

func NewCredential(content []byte) (*CredentialContext, error) {
	credential := &CredentialContext{}
	err := json.Unmarshal(content, credential)
	if err != nil {
		return nil, err
	}
	return credential, nil
}

type CredentialContext struct {
	CredentialType string            `json:"type"`
	Target         string            `json:"target"`
	Params         map[string]string `json:"params"`
}

func (c *CredentialContext) Type() string {
	return consts.ContextCredential
}

func (c *CredentialContext) Marshal() []byte {
	marshal, err := json.Marshal(c)
	if err != nil {
		return nil
	}
	return marshal
}

func (c *CredentialContext) String() string {
	return fmt.Sprintf("%s: %s %s\n", c.CredentialType, c.Target, MapJoin(c.Params))
}

func MapJoin(m map[string]string) string {
	var s string
	for k, v := range m {
		s += fmt.Sprintf("%s: %s ", k, v)
	}
	return s
}
