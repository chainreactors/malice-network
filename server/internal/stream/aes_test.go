package cryptostream

import (
	"bytes"
	"fmt"
	"testing"
)

func TestAesCtrEncryptor_EncryptDecrypt(t *testing.T) {
	key := [32]byte{
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
		0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
		0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f,
	}
	iv := [16]byte{
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
	}

	encryptor, _ := NewAesCtrEncryptor(key, iv)

	originalData := []byte("This is a secret message!")
	reader := bytes.NewReader(originalData)
	writer := &bytes.Buffer{}

	// Test encryption
	err := encryptor.Encrypt(reader, writer)
	if err != nil {
		t.Fatalf("failed to encrypt: %v", err)
	}

	encryptedData := writer.Bytes()
	fmt.Println(encryptedData)
	// Test decryption
	decReader := bytes.NewReader(encryptedData)
	decWriter := &bytes.Buffer{}

	err = encryptor.Decrypt(decReader, decWriter)
	if err != nil {
		t.Fatalf("failed to decrypt: %v", err)
	}

	decryptedData := decWriter.Bytes()

	// Verify the decrypted data matches the original data
	if !bytes.Equal(originalData, decryptedData) {
		t.Fatalf("decrypted data does not match original, got: %s", decryptedData)
	}
}
