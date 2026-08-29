package output

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/chainreactors/malice-network/helper/sekurlsa"
)

// ParseMinidumpFile extracts credentials from a restored LSASS minidump.
// Non-minidump files return (nil, nil). Template/build mismatches return
// whatever sessions were recovered plus the parser error; callers should
// log the error and still persist any credentials.
func ParseMinidumpFile(path string) ([]*CredentialContext, error) {
	res, err := sekurlsa.ParseFile(path)
	if res != nil {
		defer res.Wipe()
	}
	if err != nil && errors.Is(err, sekurlsa.ErrNotMinidump) {
		return nil, nil
	}
	return credentialContextsFromSekurlsa(res), err
}

func credentialContextsFromSekurlsa(res *sekurlsa.Result) []*CredentialContext {
	if res == nil {
		return nil
	}
	seen := make(map[string]bool)
	var out []*CredentialContext
	for _, session := range res.Sessions {
		for _, cred := range session.Credentials {
			for _, ctx := range credentialContextFromSekurlsa(session, cred) {
				key := fmt.Sprintf("%s:%s:%s:%s", ctx.CredentialType, ctx.Params["username"], ctx.Params["domain"], ctx.Params["password"])
				if seen[key] {
					continue
				}
				seen[key] = true
				out = append(out, ctx)
			}
		}
	}
	return out
}

func credentialContextFromSekurlsa(session sekurlsa.LogonSession, cred sekurlsa.Credential) []*CredentialContext {
	switch c := cred.(type) {
	case *sekurlsa.MSVCredential:
		if c == nil || !c.Found || allZero(c.NTHash[:]) {
			return nil
		}
		user, domain := firstNonEmpty(c.UserName, session.UserName), firstNonEmpty(c.LogonDomain, session.LogonDomain)
		if !isValidValue(user) {
			return nil
		}
		return []*CredentialContext{newMinidumpCredential(NtlmCredential, user, domain, strings.ToLower(hex.EncodeToString(c.NTHash[:])), c.AuthPackage())}
	case *sekurlsa.WdigestCredential:
		return plaintextMinidumpCred(session, c.UserName, c.LogonDomain, c.Password, c.Found, c.AuthPackage())
	case *sekurlsa.TSPkgCredential:
		return plaintextMinidumpCred(session, c.UserName, c.LogonDomain, c.Password, c.Found, c.AuthPackage())
	case *sekurlsa.CredManCredential:
		return plaintextMinidumpCred(session, c.UserName, c.LogonDomain, c.Password, c.Found, c.AuthPackage())
	case *sekurlsa.KerberosCredential:
		if c == nil {
			return nil
		}
		return plaintextMinidumpCred(session, c.UserName, c.LogonDomain, c.Password, isValidValue(c.Password), c.AuthPackage())
	case *sekurlsa.LiveSSPCredential:
		return plaintextMinidumpCred(session, c.UserName, c.LogonDomain, c.Password, c.Found, c.AuthPackage())
	default:
		return nil
	}
}

func plaintextMinidumpCred(session sekurlsa.LogonSession, user, domain, password string, found bool, pkg string) []*CredentialContext {
	if !found || !isValidValue(password) {
		return nil
	}
	user, domain = firstNonEmpty(user, session.UserName), firstNonEmpty(domain, session.LogonDomain)
	if !isValidValue(user) {
		return nil
	}
	return []*CredentialContext{newMinidumpCredential(UserPassCredential, user, domain, password, pkg)}
}

func newMinidumpCredential(kind, user, domain, secret, pkg string) *CredentialContext {
	target := user
	if domain != "" {
		target = domain + `\` + user
	}
	return &CredentialContext{
		Target:         target,
		CredentialType: kind,
		Params: map[string]string{
			"username": user,
			"domain":   domain,
			"password": secret,
			"source":   "nanodump",
			"package":  pkg,
		},
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func allZero(b []byte) bool {
	for _, v := range b {
		if v != 0 {
			return false
		}
	}
	return true
}
