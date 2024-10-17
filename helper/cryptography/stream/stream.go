package cryptostream

import (
	"bytes"
	"fmt"
	"github.com/chainreactors/malice-network/helper/consts"
	"io"
)

func NewCryptor(name string, key, secret []byte) (Cryptor, error) {
	switch name {
	case consts.CryptorXOR:
		return NewXorEncryptor(key, secret), nil
	case consts.CryptorAES:
		return NewAesCtrEncryptor([32]byte(PKCS7Pad(key, 32)), [16]byte(PKCS7Pad(secret, 16)))
	case consts.CryptorRAW:
		return NewXorEncryptor([]byte{0}, []byte{0}), nil
	default:
		return nil, fmt.Errorf("unknown cryptor: %s", name)
	}
}

type Cryptor interface {
	Encrypt(reader io.Reader, writer io.Writer) error
	Decrypt(reader io.Reader, writer io.Writer) error
	Reset() error
}

// PKCS7Pad pads the input data to the block size using PKCS#7 padding
func PKCS7Pad(data []byte, blockSize int) []byte {
	if len(data) > blockSize {
		return data[:blockSize]
	}
	padding := blockSize - (len(data) % blockSize)
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padText...)
}
