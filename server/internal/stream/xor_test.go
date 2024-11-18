package cryptostream

import (
	"bytes"
	"fmt"
	"testing"
)

func TestXorEncryptor_EncryptDecrypt(t *testing.T) {
	key := []byte{0x0} // 示例 key
	iv := []byte{0x0}  // 示例 iv
	encryptor := NewXorEncryptor(key, iv)

	plaintext := []byte("Hello, XOR encryption!") // 原始明文数据
	reader := bytes.NewReader(plaintext)
	writer := &bytes.Buffer{}

	// 测试加密
	err := encryptor.Encrypt(reader, writer)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	ciphertext := writer.Bytes()
	fmt.Println(ciphertext)
	// 测试解密
	decReader := bytes.NewReader(ciphertext)
	decWriter := &bytes.Buffer{}

	err = encryptor.Decrypt(decReader, decWriter)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	decryptedText := decWriter.Bytes()

	// 验证解密后的数据是否与原始明文一致
	if !bytes.Equal(plaintext, decryptedText) {
		t.Fatalf("Decrypted text does not match original, got: %s", decryptedText)
	}
}
