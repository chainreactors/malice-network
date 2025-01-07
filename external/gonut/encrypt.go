package gonut

import (
	"unsafe"
)

const (
	// CHASKEY block cipher
	// 128-bit block with 128-bit key

	CIPHER_BLK_LEN = (128 / 8)
	CIPHER_KEY_LEN = (128 / 8)
)

var (
	ENCRYPT = ChasKey
)

// Crypt
// donut/include/donut.h
// typedef struct _DONUT_CRYPT { ... }
type Crypt struct {
	MasterKey    [CIPHER_KEY_LEN]byte // master key
	CounterNonce [CIPHER_BLK_LEN]byte // counter + nonce
}

// ChasKey
// donut/encrypt.c
// static void chaskey(void *mk, void *p) { ... }
func ChasKey(mk [CIPHER_KEY_LEN]uint8, p [CIPHER_BLK_LEN]uint8) [CIPHER_BLK_LEN]uint8 {
	w := *(*[CIPHER_BLK_LEN / 4]uint32)(unsafe.Pointer(&p))
	k := *(*[CIPHER_BLK_LEN / 4]uint32)(unsafe.Pointer(&mk))

	// add 128-bit master key
	for i := 0; i < 4; i++ {
		w[i] ^= k[i]
	}
	// apply 16 rounds of permutation
	for i := 0; i < 16; i++ {
		w[0] += w[1]
		w[1] = ROTR32(w[1], 27) ^ w[0]
		w[2] += w[3]
		w[3] = ROTR32(w[3], 24) ^ w[2]
		w[2] += w[1]
		w[0] = ROTR32(w[0], 16) + w[3]
		w[3] = ROTR32(w[3], 19) ^ w[0]
		w[1] = ROTR32(w[1], 25) ^ w[2]
		w[2] = ROTR32(w[2], 16)
	}
	// add 128-bit master key
	for i := 0; i < 4; i++ {
		w[i] ^= k[i]
	}

	return *(*[CIPHER_BLK_LEN]uint8)(unsafe.Pointer(&w))
}

// DonutEncrypt encrypt/decrypt data in counter mode
// donut/encrypt.c
// void donut_encrypt(void *mk, void *ctr, void *data, uint32_t len) { ... }
func DonutEncrypt(mk [CIPHER_KEY_LEN]byte, ctr []byte, data []byte) []byte {
	length := len(data)
	position := 0
	result := make([]byte, length)

	var x [CIPHER_BLK_LEN]uint8

	for length != 0 {
		// copy counter+nonce to local buffer
		copy(x[:CIPHER_BLK_LEN], ctr[:CIPHER_BLK_LEN])

		// donut_encrypt x
		x = ENCRYPT(mk, x)

		// XOR plaintext with ciphertext
		r := length

		if length > CIPHER_BLK_LEN {
			r = CIPHER_BLK_LEN
		}

		for i := 0; i < r; i++ {
			result[i+position] = data[i+position] ^ x[i]
		}

		// update length + position
		length -= r
		position += r

		// update counter
		for i := CIPHER_BLK_LEN - 1; i >= 0; i-- {
			ctr[i]++
			if ctr[i] != 0 {
				break
			}
		}
	}

	return result
}
