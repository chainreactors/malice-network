package cryptostream

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"io"
)

// AesCtrEncryptor struct holding key, iv, and separate streams for encryption and decryption
type AesCtrEncryptor struct {
	key           [32]byte
	iv            [16]byte
	encryptStream cipher.Stream // 加密用的 stream
	decryptStream cipher.Stream // 解密用的 stream
}

// NewAesCtrEncryptor creates a new instance of AesCtrEncryptor
func NewAesCtrEncryptor(key [32]byte, iv [16]byte) (*AesCtrEncryptor, error) {
	// Initialize AES cipher block
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	// Create encryption and decryption streams
	encryptStream := cipher.NewCTR(block, iv[:])
	decryptStream := cipher.NewCTR(block, iv[:])

	return &AesCtrEncryptor{
		key:           key,
		iv:            iv,
		encryptStream: encryptStream,
		decryptStream: decryptStream,
	}, nil
}

// Encrypt encrypts data using AES-256-CTR mode
func (e *AesCtrEncryptor) Encrypt(reader io.Reader, writer io.Writer) error {
	buffer := &bytes.Buffer{}
	_, err := io.Copy(buffer, reader)
	if err != nil {
		return err
	}

	data := buffer.Bytes()

	// 使用独立的加密 stream
	e.encryptStream.XORKeyStream(data, data)

	_, err = writer.Write(data)
	return err
}

// Decrypt decrypts data using AES-256-CTR mode
func (e *AesCtrEncryptor) Decrypt(reader io.Reader, writer io.Writer) error {
	buffer := &bytes.Buffer{}
	_, err := io.Copy(buffer, reader)
	if err != nil {
		return err
	}

	data := buffer.Bytes()

	// 使用独立的解密 stream
	e.decryptStream.XORKeyStream(data, data)

	_, err = writer.Write(data)
	return err
}

// Reset reinitializes the encrypt and decrypt streams
func (e *AesCtrEncryptor) Reset() error {
	// Reset both encryption and decryption streams
	block, err := aes.NewCipher(e.key[:])
	if err != nil {
		return err
	}

	e.encryptStream = cipher.NewCTR(block, e.iv[:])
	e.decryptStream = cipher.NewCTR(block, e.iv[:])

	return nil
}
