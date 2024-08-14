package cryptography

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rc4"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
)

// RC4 encryption - Cryptographically insecure!
// Added for stage-listener shellcode obfuscation
// Dont use for anything else!
func RC4EncryptUnsafe(data []byte, key []byte) []byte {
	cipher, err := rc4.NewCipher(key)
	if err != nil {
		return make([]byte, 0)
	}
	cipherText := make([]byte, len(data))
	cipher.XORKeyStream(cipherText, data)
	return cipherText
}

// PreludeEncrypt the results
func PreludeEncrypt(data []byte, key []byte, iv []byte) []byte {
	plainText, err := pad(data, aes.BlockSize)
	if err != nil {
		return make([]byte, 0)
	}
	block, _ := aes.NewCipher(key)
	cipherText := make([]byte, aes.BlockSize+len(plainText))
	// Create a random IV if none was provided
	// len(nil) returns 0
	if len(iv) == 0 {
		iv = cipherText[:aes.BlockSize]
		if _, err := io.ReadFull(rand.Reader, iv); err != nil {
			return make([]byte, 0)
		}
	} else {
		// make sure we copy the IV
		copy(cipherText[:aes.BlockSize], iv)
	}
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(cipherText[aes.BlockSize:], plainText)
	return cipherText
}

// PreludeDecrypt a command
func PreludeDecrypt(data []byte, key []byte) []byte {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil
	}
	iv := data[:aes.BlockSize]
	data = data[aes.BlockSize:]
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(data, data)
	data, _ = unpad(data, aes.BlockSize)
	return data
}

func pad(buf []byte, size int) ([]byte, error) {
	bufLen := len(buf)
	padLen := size - bufLen%size
	padded := make([]byte, bufLen+padLen)
	copy(padded, buf)
	for i := 0; i < padLen; i++ {
		padded[bufLen+i] = byte(padLen)
	}
	return padded, nil
}

func unpad(padded []byte, size int) ([]byte, error) {
	if len(padded)%size != 0 {
		return nil, errors.New("pkcs7: Padded value wasn't in correct size")
	}
	bufLen := len(padded) - int(padded[len(padded)-1])
	buf := make([]byte, bufLen)
	copy(buf, padded[:bufLen])
	return buf, nil
}

func ProcessPEM(filepath string) (string, error) {
	pemData, err := os.ReadFile(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to read PEM file: %v", err)
	}
	block, _ := pem.Decode(pemData)
	if block == nil {
		return "", fmt.Errorf("failed to parse PEM file")
	}

	var pemString string

	switch block.Type {
	case "CERTIFICATE":
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return "", fmt.Errorf("failed to parse certificate: %v", err)
		}
		pemString = string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}))
	case "RSA PRIVATE KEY":
		key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return "", fmt.Errorf("failed to parse RSA private key: %v", err)
		}
		pemString = string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}))
	case "EC PRIVATE KEY":
		key, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return "", fmt.Errorf("failed to parse EC private key: %v", err)
		}
		ecBytes, err := x509.MarshalECPrivateKey(key)
		if err != nil {
			return "", fmt.Errorf("failed to marshal EC private key: %v", err)
		}
		pemString = string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: ecBytes}))
	case "PRIVATE KEY":
		_, err = x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return "", fmt.Errorf("failed to parse private key: %v", err)
		}
		pemString = string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: block.Bytes}))
	default:
		return "", fmt.Errorf("unknown PEM type: %s", block.Type)
	}

	return pemString, nil
}
