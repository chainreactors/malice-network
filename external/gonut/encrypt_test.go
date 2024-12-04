package gonut

import (
	"testing"
)

// donut/encrypt.c
var (
	// 128-bit master key
	key_tv = [16]uint8{0x56, 0x09, 0xe9, 0x68, 0x5f, 0x58, 0xe3, 0x29, 0x40, 0xec, 0xec, 0x98, 0xc5, 0x22, 0x98, 0x2f}

	// 128-bit plain text
	plain_tv = [16]uint8{0xb8, 0x23, 0x28, 0x26, 0xfd, 0x5e, 0x40, 0x5e, 0x69, 0xa3, 0x01, 0xa9, 0x78, 0xea, 0x7a, 0xd8}

	// 128-bit cipher text
	cipher_tv = [16]uint8{0xd5, 0x60, 0x8d, 0x4d, 0xa2, 0xbf, 0x34, 0x7b, 0xab, 0xf8, 0x77, 0x2f, 0xdf, 0xed, 0xde, 0x07}

	// 128-bit counter
	ctr_tv = [16]uint8{0xd0, 0x01, 0x36, 0x9b, 0xef, 0x6a, 0xa1, 0x05, 0x1d, 0x2d, 0x21, 0x98, 0x19, 0x8d, 0x88, 0x93}

	// 128-bit ciphertext for testing donut_encrypt
	donut_crypt_tv = [16]uint8{0xd0, 0x01, 0x36, 0x9b, 0xef, 0x6a, 0xa1, 0x05, 0x1d, 0x2d, 0x21, 0x98, 0x19, 0x8d, 0x8b, 0x13}
)

// TestDonutEncrypt
// donut/encrypt.c
// int main(void) { ... }
func TestDonutEncrypt(t *testing.T) {
	var key [16]uint8
	var tmp [16]uint8

	copy(key[:], key_tv[:])
	copy(tmp[:], ctr_tv[:])

	data := make([]uint8, 77)

	for i := 0; i < 128; i++ {
		// encrypt data
		data = DonutEncrypt(key, tmp[:], data)
		// update key with first 16 bytes of ciphertext
		for j := 0; j < 16; j++ {
			key[j] ^= data[j]
		}
	}

	t.Logf("%x", data)

	if donut_crypt_tv != tmp {
		t.Errorf("Donut Encrypt test: FAILED (%x!=%x)", tmp, donut_crypt_tv)
	}
	// data:
	// 93b060a6a4b3569c0b44e41da1cbdf45353508c113417cd700267f9c9945967c3bfe13ca12b7c
	// 8147aa3f90b63bca91acfd14a09a021274dfe6a7663015e8f8216bca1f06cdaf19cd56d53960c
}

// TestChasKey
// donut/encrypt.c
// int crypto_test(void) { ... }
func TestChasKey(t *testing.T) {
	var tmp1 [16]uint8
	copy(tmp1[:], plain_tv[:])

	result := ChasKey(key_tv, tmp1)

	if result != cipher_tv {
		t.Errorf("Chaskey test: FAILED (%x!=%x)", result, cipher_tv)
	}
}
