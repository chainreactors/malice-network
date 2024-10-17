package cryptostream

import (
	"bytes"
	"io"
)

// XorEncryptor struct holding key, iv, and separate counters for encryption and decryption
type XorEncryptor struct {
	key            []byte
	iv             []byte
	encryptCounter int // 用于加密的计数器
	decryptCounter int // 用于解密的计数器
}

// NewXorEncryptor creates a new instance of XorEncryptor
func NewXorEncryptor(key []byte, iv []byte) *XorEncryptor {
	return &XorEncryptor{
		key:            key,
		iv:             iv,
		encryptCounter: 0,
		decryptCounter: 0,
	}
}

// xorProcess applies XOR encryption/decryption with counter support
func (e *XorEncryptor) xorProcess(data []byte, counter *int) {
	keyLen := len(e.key)
	ivLen := len(e.iv)

	for i := range data {
		index := *counter + i // 从计数器位置开始
		keyByte := e.key[index%keyLen]
		ivByte := e.iv[index%ivLen]
		data[i] ^= keyByte ^ ivByte // XOR encryption/decryption
	}

	*counter += len(data)
}

// Encrypt reads data from reader, applies XOR encryption, and writes it to writer
func (e *XorEncryptor) Encrypt(reader io.Reader, writer io.Writer) error {
	buffer := &bytes.Buffer{}
	_, err := io.Copy(buffer, reader)
	if err != nil {
		return err
	}

	data := buffer.Bytes()
	e.xorProcess(data, &e.encryptCounter)

	_, err = writer.Write(data)
	return err
}

// Decrypt reads data from reader, applies XOR decryption, and writes it to writer
func (e *XorEncryptor) Decrypt(reader io.Reader, writer io.Writer) error {
	buffer := &bytes.Buffer{}
	_, err := io.Copy(buffer, reader)
	if err != nil {
		return err
	}

	data := buffer.Bytes()
	e.xorProcess(data, &e.decryptCounter)

	_, err = writer.Write(data)
	return err
}

// Reset resets both the encryption and decryption counters
func (e *XorEncryptor) Reset() error {
	e.encryptCounter = 0
	e.decryptCounter = 0
	return nil
}
