package cryptostream

import (
	"bytes"
	"fmt"
	"github.com/chainreactors/malice-network/helper/consts"
	"io"
	"strings"
)

func NewCryptor(name string, key, secret []byte) (Cryptor, error) {
	switch strings.ToUpper(name) {
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

func PKCS7Pad(data []byte, blockSize int) []byte {
	if len(data) >= blockSize {
		return data[:blockSize]
	}
	padding := blockSize - len(data)
	padText := bytes.Repeat([]byte{0}, padding)
	return append(data, padText...)
}

func Decrypt(c Cryptor, en []byte) ([]byte, error) {
	reader := bytes.NewReader(en)
	writer := &bytes.Buffer{}

	err := c.Decrypt(reader, writer)
	if err != nil {
		return nil, err
	}
	return writer.Bytes(), nil
}

func Encrypt(c Cryptor, en []byte) ([]byte, error) {
	reader := bytes.NewReader(en)
	writer := &bytes.Buffer{}

	err := c.Encrypt(reader, writer)
	if err != nil {
		return nil, err
	}
	return writer.Bytes(), nil
}
