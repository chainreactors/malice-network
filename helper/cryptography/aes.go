package cryptography

import (
	"crypto/aes"
	"crypto/rand"
	"encoding/hex"
	"errors"
)

var AESKey string

func InitAES(aesKey string) {
	AESKey = aesKey
}

// AESECBEncrypt encrypts data using AES ECB mode
// key: encryption key, must be 16, 24, or 32 bytes
// plaintext: data to encrypt
// returns encrypted data and possible error
func AESECBEncrypt(key, plaintext []byte) ([]byte, error) {
	// Validate key length
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, errors.New("key length must be 16, 24, or 32 bytes")
	}

	// Create AES cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// Calculate padding bytes needed
	blockSize := block.BlockSize()
	padding := blockSize - (len(plaintext) % blockSize)

	// Create padded data
	paddedData := make([]byte, len(plaintext)+padding)
	copy(paddedData, plaintext)

	// Apply PKCS7 padding
	for i := len(plaintext); i < len(paddedData); i++ {
		paddedData[i] = byte(padding)
	}

	// Perform ECB encryption
	encrypted := make([]byte, len(paddedData))
	for i := 0; i < len(paddedData); i += blockSize {
		block.Encrypt(encrypted[i:i+blockSize], paddedData[i:i+blockSize])
	}

	return encrypted, nil
}

// AESECBDecrypt decrypts data using AES ECB mode
// key: decryption key, must be 16, 24, or 32 bytes
// ciphertext: data to decrypt
// returns decrypted data and possible error
func AESECBDecrypt(key, ciphertext []byte) ([]byte, error) {
	// Validate key length
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, errors.New("key length must be 16, 24, or 32 bytes")
	}

	// Validate ciphertext length
	if len(ciphertext) == 0 || len(ciphertext)%aes.BlockSize != 0 {
		return nil, errors.New("ciphertext length must be multiple of AES block size")
	}

	// Create AES cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// Perform ECB decryption
	decrypted := make([]byte, len(ciphertext))
	for i := 0; i < len(ciphertext); i += aes.BlockSize {
		block.Decrypt(decrypted[i:i+aes.BlockSize], ciphertext[i:i+aes.BlockSize])
	}

	// Remove PKCS7 padding
	padding := int(decrypted[len(decrypted)-1])
	if padding > aes.BlockSize || padding == 0 {
		return nil, errors.New("invalid padding")
	}

	// Validate padding
	for i := len(decrypted) - padding; i < len(decrypted); i++ {
		if decrypted[i] != byte(padding) {
			return nil, errors.New("invalid padding")
		}
	}

	return decrypted[:len(decrypted)-padding], nil
}

// GenerateAESKey generates a random AES key of specified length
// keySize: key length (16, 24, or 32 bytes)
// returns generated key and possible error
func GenerateAESKey(keySize int) ([]byte, error) {
	if keySize != 16 && keySize != 24 && keySize != 32 {
		return nil, errors.New("key length must be 16, 24, or 32 bytes")
	}

	key := make([]byte, keySize)
	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}

	return key, nil
}

// AESECBEncryptNoPadding encrypts data using AES ECB mode without automatic padding
// key: encryption key, must be 16, 24, or 32 bytes
// plaintext: data to encrypt (must be multiple of AES block size)
// returns encrypted data and possible error
func AESECBEncryptNoPadding(key, plaintext []byte) ([]byte, error) {
	// Validate key length
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, errors.New("key length must be 16, 24, or 32 bytes")
	}

	// Validate plaintext length
	if len(plaintext) == 0 || len(plaintext)%aes.BlockSize != 0 {
		return nil, errors.New("plaintext length must be multiple of AES block size (16 bytes)")
	}

	// Create AES cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// Perform ECB encryption
	encrypted := make([]byte, len(plaintext))
	for i := 0; i < len(plaintext); i += aes.BlockSize {
		block.Encrypt(encrypted[i:i+aes.BlockSize], plaintext[i:i+aes.BlockSize])
	}

	return encrypted, nil
}

// AESECBDecryptNoPadding decrypts data using AES ECB mode without removing padding
// key: decryption key, must be 16, 24, or 32 bytes
// ciphertext: data to decrypt (must be multiple of AES block size)
// returns decrypted data and possible error
func AESECBDecryptNoPadding(key, ciphertext []byte) ([]byte, error) {
	// Validate key length
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, errors.New("key length must be 16, 24, or 32 bytes")
	}

	// Validate ciphertext length
	if len(ciphertext) == 0 || len(ciphertext)%aes.BlockSize != 0 {
		return nil, errors.New("ciphertext length must be multiple of AES block size")
	}

	// Create AES cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// Perform ECB decryption
	decrypted := make([]byte, len(ciphertext))
	for i := 0; i < len(ciphertext); i += aes.BlockSize {
		block.Decrypt(decrypted[i:i+aes.BlockSize], ciphertext[i:i+aes.BlockSize])
	}

	return decrypted, nil
}

// PadToBlockSize pads data to AES block size using PKCS7 padding
// data: data to pad
// returns padded data
func PadToBlockSize(data []byte) []byte {
	blockSize := aes.BlockSize
	padding := blockSize - (len(data) % blockSize)

	paddedData := make([]byte, len(data)+padding)
	copy(paddedData, data)

	// Apply PKCS7 padding
	for i := len(data); i < len(paddedData); i++ {
		paddedData[i] = byte(padding)
	}

	return paddedData
}

// UnpadFromBlockSize removes PKCS7 padding from data
// data: padded data
// returns unpadded data and possible error
func UnpadFromBlockSize(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("data is empty")
	}

	padding := int(data[len(data)-1])
	if padding > aes.BlockSize || padding == 0 {
		return nil, errors.New("invalid padding")
	}

	// Validate padding
	for i := len(data) - padding; i < len(data); i++ {
		if data[i] != byte(padding) {
			return nil, errors.New("invalid padding")
		}
	}

	return data[:len(data)-padding], nil
}

func BytesToHex(data []byte) string {
	return hex.EncodeToString(data)
}

// EncryptWithGlobalKey encrypts data using the global AES key
// plaintext: data to encrypt
// returns encrypted data and possible error
func EncryptWithGlobalKey(plaintext []byte) ([]byte, error) {
	if AESKey == "" {
		return nil, errors.New("global AES key not initialized, call InitAES first")
	}
	return AESECBEncrypt([]byte(AESKey), plaintext)
}

// DecryptWithGlobalKey decrypts data using the global AES key
// ciphertext: data to decrypt
// returns decrypted data and possible error
func DecryptWithGlobalKey(ciphertext []byte) ([]byte, error) {
	if AESKey == "" {
		return nil, errors.New("global AES key not initialized, call InitAES first")
	}
	return AESECBDecrypt([]byte(AESKey), ciphertext)
}

// HexToBytes converts hexadecimal string to byte slice
// hexStr: hexadecimal string to convert
// returns byte slice and possible error
func HexToBytes(hexStr string) ([]byte, error) {
	return hex.DecodeString(hexStr)
}
