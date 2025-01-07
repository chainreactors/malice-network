package gonut

import "unsafe"

// donut/hash.h
const (
	MARU_MAX_STR  = 64
	MARU_BLK_LEN  = 16
	MARU_HASH_LEN = 8
	MARU_IV_LEN   = MARU_HASH_LEN
)

var (
	MARU_CRYPT = Speck
)

// ROTR32
// donut/hash.h
// #define ROTR32(v,n)(((v)>>(n))|((v)<<(32-(n))))
func ROTR32(v uint32, n uint32) uint32 {
	return ((v) >> (n)) | ((v) << (32 - (n)))
}

// Speck SPECK-64/128
// donut/hash.c
// static uint64_t speck(void *mk, uint64_t p) { ... }
func Speck(mk [MARU_BLK_LEN]byte, p uint64) uint64 {
	var k [4]uint32

	var x [8]uint8
	w := (*[2]uint32)(unsafe.Pointer(&x))
	q := (*uint64)(unsafe.Pointer(&x))

	// copy 64-bit plaintext to local buffer
	*q = p

	// copy 128-bit master key to local buffer
	for i := 0; i < 4; i++ {
		k[i] = (*[MARU_BLK_LEN / 4]uint32)(unsafe.Pointer(&mk))[i]
	}

	for i := uint32(0); i < 27; i++ {
		// encrypt 64-bit plaintext
		w[0] = (ROTR32(w[0], 8) + w[1]) ^ k[0]
		w[1] = ROTR32(w[1], 29) ^ w[0]

		// create next 32-bit subkey
		t := k[3]
		k[3] = (ROTR32(k[1], 8) + k[0]) ^ i
		k[0] = ROTR32(k[0], 29) ^ k[3]

		k[1] = k[2]
		k[2] = t
	}

	// return 64-bit ciphertext
	return *q
}

// Maru
// donut/hash.c
// uint64_t maru(const void *input, uint64_t iv) { ... }
func Maru(input []byte, iv uint64) uint64 {
	var m [MARU_BLK_LEN]uint8
	b := (*[MARU_BLK_LEN]uint8)(unsafe.Pointer(&m))
	w := (*[MARU_BLK_LEN / 4]uint32)(unsafe.Pointer(&m))

	// set H to initial value
	h := iv

	for idx, length, end := 0, 0, false; !end; {
		// end of string or max len?
		if length == len(input) || length == MARU_MAX_STR {
			// zero remainder of M
			copy(b[idx:], make([]uint8, MARU_BLK_LEN-idx))
			// store the end bit
			b[idx] = 0x80
			// have we space in M for api length?
			if idx >= MARU_BLK_LEN-4 {
				// no, update H with E
				h ^= MARU_CRYPT(m, h)
				// zero M
				m = [MARU_BLK_LEN]byte{0}
			}
			// store total length in bits
			w[MARU_BLK_LEN/4-1] = uint32(length * 8)
			idx = MARU_BLK_LEN
			end = true
		} else {
			// store character from api string
			b[idx] = input[length]
			idx++
			length++
		}

		if idx == MARU_BLK_LEN {
			// update H with E
			h ^= MARU_CRYPT(m, h)
			// reset idx
			idx = 0
		}
	}

	return h
}
