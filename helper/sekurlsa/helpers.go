package sekurlsa

import (
	"encoding/binary"
	"fmt"
)

// Shared building blocks for the per-provider walkers (Wdigest /
// TSPkg / DPAPI / Kerberos / CredMan / CloudAP / LiveSSP). MSV1_0
// is a hash-bucket array, not a flat list, and stays separate.
//
// Before v0.32.0 each walker carried its own ~30-line copy of the
// pattern-match-then-Flink-walk loop; the bug surface scaled
// linearly with the number of providers. Centralising here means a
// single fix touches every consumer.

// resolveListHead reads the mapped image of mod, pattern-scans for
// sig (with optional wildcards), and dereferences the rel32 at
// sig+offset. Returns the absolute VA the rel32 points at —
// typically the address of a doubly-linked list head OR an
// RTL_AVL_TABLE root.
//
// Callers wrap the returned error with their provider's name; this
// helper stays neutral so the same code serves Wdigest, TSPkg,
// CloudAP, etc.
func resolveListHead(r *reader, mod Module, sig []byte, wildcards []int, offset int32) (uint64, error) {
	body, err := r.ReadVA(mod.BaseOfImage, int(mod.SizeOfImage))
	if err != nil {
		return 0, fmt.Errorf("read %s body: %w", mod.Name, err)
	}
	return derefRel32(body, mod.BaseOfImage, sig, wildcards, offset, r)
}

// walkLinkedList traces a doubly-linked LIST_ENTRY chain rooted at
// listHead. Each node has Flink at offset 0; reading it gives the
// next node's address. The walk stops at:
//   - the maxNodes safety cap (defeats malformed dumps),
//   - a self-pointer (cur points back to itself),
//   - a Flink read failure or zero next pointer,
//   - the terminator condition cur == listHead (NT's circular
//     terminator).
//
// Per-node failures returned by visit are collected as warnings and
// surfaced to the caller. visit returns "" on success or a
// human-readable warning string.
func walkLinkedList(r *reader, listHead uint64, nodeSize uint32, maxNodes int, visit func(node []byte, nodeVA uint64) string) []string {
	flink, err := readPointer(r, listHead)
	if err != nil || flink == 0 || flink == listHead {
		return nil
	}
	var warnings []string
	walked := 0
	for cur := flink; cur != listHead && walked < maxNodes; walked++ {
		node, err := r.ReadVA(cur, int(nodeSize))
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("node @0x%X: %v", cur, err))
			break
		}
		if w := visit(node, cur); w != "" {
			warnings = append(warnings, w)
		}
		next, err := readPointer(r, cur)
		if err != nil || next == 0 || next == cur {
			break
		}
		cur = next
	}
	return warnings
}

// mergeByLUID grafts a map of credentials onto matching
// LogonSession entries by LUID. LUIDs absent from sessions surface
// as new sessions built via fabricator. Returns the (possibly
// extended) sessions slice.
//
// Generic on the concrete Credential type so each provider keeps
// its strongly-typed map without an interface-assertion at the call
// site. The fabricator constructs an orphan-session shell whose
// fields the caller wants to populate (UserName, LogonDomain) so
// MSV-only-fields like LogonType / SID stay zero.
func mergeByLUID[T Credential](sessions []LogonSession, m map[uint64]T, fabricator func(luid uint64, c T) LogonSession) []LogonSession {
	if len(m) == 0 {
		return sessions
	}
	seen := make(map[uint64]bool, len(sessions))
	for i := range sessions {
		if c, ok := m[sessions[i].LUID]; ok {
			sessions[i].Credentials = append(sessions[i].Credentials, c)
			seen[sessions[i].LUID] = true
		}
	}
	for luid, c := range m {
		if seen[luid] {
			continue
		}
		sessions = append(sessions, fabricator(luid, c))
	}
	return sessions
}

// readEncryptedPassword projects a 16-byte UNICODE_STRING from
// node[offset:offset+16] (Length + MaxLength + Padding + Buffer
// pointer), reads Length encrypted bytes from Buffer, and decrypts
// them with lsaKey. Returns the decoded UTF-16LE plaintext OR
// ("", nil) when the password is empty / no buffer / size out of
// range. Errors propagate up so callers can wrap them with provider
// context.
//
// Used by Wdigest / TSPkg / LiveSSP / CredMan / Kerberos —
// every walker that decodes a single Microsoft-encrypted password
// UNICODE_STRING.
func readEncryptedPassword(r *reader, field []byte, lsaKey *lsaKey) (string, error) {
	if len(field) < 16 {
		return "", nil
	}
	pwdLen := binary.LittleEndian.Uint16(field[0:2])
	pwdBufPtr := binary.LittleEndian.Uint64(field[8:16])
	if pwdLen == 0 || pwdBufPtr == 0 {
		return "", nil
	}
	ct, err := r.ReadVA(pwdBufPtr, int(pwdLen))
	if err != nil {
		return "", fmt.Errorf("cipher @0x%X: %w", pwdBufPtr, err)
	}
	pt, err := decryptLSA(ct, lsaKey)
	if err != nil {
		return "", fmt.Errorf("decrypt @0x%X: %w", pwdBufPtr, err)
	}
	return decodeUTF16LEBytes(pt), nil
}

// readUnicodeStringIfFits is a bounds-checked wrapper around
// readUnicodeString — returns "" if the requested 16-byte field
// would extend past nodeSize. Avoids slice-bounds panics on a
// malformed Layout. Lives here (not in credman.go) because every
// provider's optional UNICODE_STRING fields use it.
func readUnicodeStringIfFits(r *reader, node []byte, offset, nodeSize uint32) string {
	if offset == 0 || offset+16 > nodeSize {
		return ""
	}
	return readUnicodeString(r, node[offset:offset+16])
}
