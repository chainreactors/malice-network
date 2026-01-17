package cryptography

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"filippo.io/age"
	"fmt"
	"github.com/chainreactors/malice-network/helper/cryptography/minisign"
	"github.com/chainreactors/malice-network/helper/encoders"
	"golang.org/x/crypto/chacha20poly1305"
	"io"
	"math/big"
	"sync"
)

const (
	serverAgeKeyPairKey      = "server.age"
	serverMinisignPrivateKey = "server.minisign"

	sha256Size = 32 // size in bytes of a sha256 hash
)

var (
	// ErrInvalidKeyLength - Invalid key length
	ErrInvalidKeyLength = errors.New("invalid length")

	// ErrReplayAttack - Replay attack
	ErrReplayAttack = errors.New("replay attack detected")

	// ErrDecryptFailed
	ErrDecryptFailed = errors.New("decryption failed")

	// This will be prepended to any age encrypted message, however
	// since we already know what it is, and who the recipient is,
	// and we can ensure there will only ever be a single recipient,
	// we can just ignore add/remove it at runtime to safe space.
	agePrefix = []byte("age-encryption.org/v1\n-> X25519 ")

	cachedAgeServerKeyPairOnce sync.Once
	cachedAgeServerKeyPair     *AgeKeyPair
	cachedAgeServerKeyPairErr  error

	cachedMinisignServerKeyOnce sync.Once
	cachedMinisignServerKey     *minisign.PrivateKey
	cachedMinisignServerKeyErr  error

	ageKeyExReplay sync.Map
)

// deriveKeyFrom - Derives a key from input data using SHA256
func deriveKeyFrom(data []byte) [chacha20poly1305.KeySize]byte {
	digest := sha256.Sum256(data)
	var key [chacha20poly1305.KeySize]byte
	copy(key[:], digest[:chacha20poly1305.KeySize])
	return key
}

// RandomSymmetricKey - Generate random ID of randomIDSize bytes
func RandomSymmetricKey() [chacha20poly1305.KeySize]byte {
	randBuf := make([]byte, 64)
	_, err := rand.Read(randBuf)
	if err != nil {
		panic(err)
	}
	return deriveKeyFrom(randBuf)
}

// KeyFromBytes - Convert to fixed length buffer
func KeyFromBytes(data []byte) ([chacha20poly1305.KeySize]byte, error) {
	var key [chacha20poly1305.KeySize]byte
	if len(data) != chacha20poly1305.KeySize {
		// We cannot return nil due to the fixed length buffer type ...
		// and it seems like a really bad idea to return a zero key in case
		// the error is not checked by the caller, so instead we return a
		// random key, which should break everything if the error is not checked.
		return RandomSymmetricKey(), ErrInvalidKeyLength
	}
	copy(key[:], data)
	return key, nil
}

// AgeKeyPair - Holds the public/private key pair
type AgeKeyPair struct {
	Public  string `json:"public"`
	Private string `json:"private"`
}

// PublicKey - Return the parsed public key
func (e *AgeKeyPair) PublicKey() *age.X25519Recipient {
	recipient, _ := age.ParseX25519Recipient(e.Public)
	return recipient
}

// PrivateBase64 - Base64 encoded private key
func (e *AgeKeyPair) PrivateKey() string {
	return e.Private
}

// RandomAgeKeyPair - Generate a random Curve 25519 key pair
func RandomAgeKeyPair() (*AgeKeyPair, error) {
	k, err := age.GenerateX25519Identity()
	if err != nil {
		return nil, err
	}
	return &AgeKeyPair{
		Public:  k.Recipient().String(),
		Private: k.String(),
	}, nil
}

// AgeEncrypt - Encrypt using Nacl Box
func AgeEncrypt(recipientPublicKey string, plaintext []byte) ([]byte, error) {
	recipient, err := age.ParseX25519Recipient(recipientPublicKey)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer([]byte{})
	stream, err := age.Encrypt(buf, recipient)
	if err != nil {
		return nil, err
	}
	if _, err := stream.Write(plaintext); err != nil {
		return nil, err
	}
	if err := stream.Close(); err != nil {
		return nil, err
	}
	return bytes.TrimPrefix(buf.Bytes(), agePrefix), nil
}

// AgeDecrypt - Decrypt using Curve 25519 + ChaCha20Poly1305
// 支持解密包含 grease recipients 的 Age 加密数据（兼容 Rust 版本）
func AgeDecrypt(recipientPrivateKey string, ciphertext []byte) ([]byte, error) {
	identity, err := age.ParseX25519Identity(recipientPrivateKey)
	if err != nil {
		return nil, err
	}

	// Accept both full age ciphertext and ciphertext with the standard prefix trimmed.
	if !bytes.HasPrefix(ciphertext, []byte("age-encryption.org/v1")) {
		prefixed := make([]byte, 0, len(agePrefix)+len(ciphertext))
		prefixed = append(prefixed, agePrefix...)
		prefixed = append(prefixed, ciphertext...)
		ciphertext = prefixed
	}

	// Age library automatically handles grease recipients
	buf := bytes.NewBuffer(ciphertext)
	stream, err := age.Decrypt(buf, identity)
	if err != nil {
		// If decryption fails, add debug info
		return nil, fmt.Errorf("age decrypt failed (ciphertext size: %d bytes): %w", len(ciphertext), err)
	}

	plaintext, err := io.ReadAll(stream)
	if err != nil {
		return nil, fmt.Errorf("failed to read decrypted stream: %w", err)
	}

	return plaintext, nil
}

// AgeKeyPairFromImplant - Decrypt the session key from an implant
func AgeKeyExFromImplant(serverPrivateKey string, implantPrivateKey string, ciphertext []byte) ([]byte, error) {
	ciphertextDigest := sha256.Sum256(ciphertext)
	b64Digest := base64.RawStdEncoding.EncodeToString(ciphertextDigest[:])

	// Decrypt the message
	plaintext, err := AgeDecrypt(serverPrivateKey, ciphertext)
	if err != nil {
		return nil, err
	}

	// Check there's enough data for an HMAC check
	if len(plaintext) <= sha256Size {
		return nil, ErrDecryptFailed
	}

	// Recompute the HMAC to verify the message
	privateKeyDigest := sha256.Sum256([]byte(implantPrivateKey))
	mac := hmac.New(sha256.New, privateKeyDigest[:])
	mac.Write(plaintext[sha256Size:])

	// Constant-time comparison of the HMACs
	if !hmac.Equal(mac.Sum(nil), plaintext[:sha256Size]) {
		return nil, ErrDecryptFailed
	}

	if _, ok := ageKeyExReplay.LoadOrStore(b64Digest, true); ok {
		return nil, ErrReplayAttack
	}
	return plaintext[sha256Size:], nil
}

// Encrypt - Encrypt using chacha20poly1305
// https://pkg.go.dev/golang.org/x/crypto/chacha20poly1305
func Encrypt(key [chacha20poly1305.KeySize]byte, plaintext []byte) ([]byte, error) {
	aead, err := chacha20poly1305.New(key[:])
	if err != nil {
		return nil, err
	}
	compressed, _ := encoders.GzipBuf(plaintext)
	plaintext = bytes.NewBuffer(compressed).Bytes()
	nonce := make([]byte, aead.NonceSize(), aead.NonceSize()+len(plaintext)+aead.Overhead())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return aead.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt - Decrypt using chacha20poly1305
// https://pkg.go.dev/golang.org/x/crypto/chacha20poly1305
func Decrypt(key [chacha20poly1305.KeySize]byte, ciphertext []byte) ([]byte, error) {
	aead, err := chacha20poly1305.New(key[:])
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < aead.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}

	// Split nonce and ciphertext.
	nonce, ciphertext := ciphertext[:aead.NonceSize()], ciphertext[aead.NonceSize():]

	// Decrypt the message and check it wasn't tampered with.
	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return encoders.GunzipBuf(plaintext), nil
}

// NewCipherContext - Wrapper around creating a cipher context from a key
func NewCipherContext(key [chacha20poly1305.KeySize]byte) *CipherContext {
	return &CipherContext{
		Key:    key,
		replay: &sync.Map{},
	}
}

// CipherContext - Tracks a series of messages encrypted under the same key
// and detects/prevents replay attacks.
type CipherContext struct {
	Key    [chacha20poly1305.KeySize]byte
	replay *sync.Map
}

// Decrypt - Decrypt a message with the contextual key and check for replay attacks
func (c *CipherContext) Decrypt(ciphertext []byte) ([]byte, error) {
	plaintext, err := Decrypt(c.Key, ciphertext)
	if err != nil {
		return nil, err
	}
	if 0 < len(ciphertext) {
		digest := sha256.Sum256(ciphertext)
		b64Digest := base64.RawStdEncoding.EncodeToString(digest[:])
		if _, ok := c.replay.LoadOrStore(b64Digest, true); ok {
			return nil, ErrReplayAttack
		}
	}
	return plaintext, nil
}

// Encrypt - Encrypt a message with the contextual key
func (c *CipherContext) Encrypt(plaintext []byte) ([]byte, error) {
	ciphertext, err := Encrypt(c.Key, plaintext)
	if err != nil {
		return nil, err
	}
	if 0 < len(ciphertext) {
		digest := sha256.Sum256(ciphertext)
		b64Digest := base64.RawStdEncoding.EncodeToString(digest[:])
		c.replay.Store(b64Digest, true)
	}
	rawSig := serverSignRawBuf(ciphertext)
	return append(rawSig, ciphertext...), nil
}

// serverSignRawBuf - Sign a buffer with the server's minisign private key
func serverSignRawBuf(buf []byte) []byte {
	privateKey := MinisignServerPrivateKey()
	rawSig := minisign.SignRawBuf(*privateKey, buf)
	return rawSig[:]
}

// AgeServerKeyPair - Get teh server's ECC key pair
func AgeServerKeyPair() *AgeKeyPair {
	// TODO: load from persistent storage; fall back to an in-memory keypair.
	cachedAgeServerKeyPairOnce.Do(func() {
		cachedAgeServerKeyPair, cachedAgeServerKeyPairErr = RandomAgeKeyPair()
	})
	if cachedAgeServerKeyPairErr != nil {
		panic(cachedAgeServerKeyPairErr)
	}
	return cachedAgeServerKeyPair
}

// MinisignServerPublicKey - Get the server's minisign public key string
func MinisignServerPublicKey() string {
	publicKey := MinisignServerPrivateKey().Public().(minisign.PublicKey)
	publicKeyText, err := publicKey.MarshalText()
	if err != nil {
		panic(err)
	}
	return string(publicKeyText)
}

// MinisignServerSign - Sign a message with the server's minisign private key
func MinisignServerSign(message []byte) string {
	privateKey := MinisignServerPrivateKey()
	return string(minisign.Sign(*privateKey, message))
}

// MinisignServerPrivateKey - Get the server's minisign key pair
func MinisignServerPrivateKey() *minisign.PrivateKey {
	// TODO: load from persistent storage; fall back to an in-memory keypair.
	cachedMinisignServerKeyOnce.Do(func() {
		_, privateKey, err := minisign.GenerateKey(rand.Reader)
		if err != nil {
			cachedMinisignServerKeyErr = err
			return
		}
		cachedMinisignServerKey = &privateKey
	})
	if cachedMinisignServerKeyErr != nil {
		panic(cachedMinisignServerKeyErr)
	}
	return cachedMinisignServerKey
}

func RandomInRange(min, max uint32) uint32 {
	rangeSize := max - min + 1
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(rangeSize)))
	return uint32(n.Int64()) + min
}

var letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandomString(length int) string {
	result := make([]byte, length)
	lettersLen := big.NewInt(int64(len(letters)))

	for i := range result {
		n, err := rand.Int(rand.Reader, lettersLen)
		if err != nil {
			return ""
		}
		result[i] = letters[n.Int64()]
	}

	return string(result)
}

func RandomBytes(size int) []byte {
	buf := make([]byte, size)
	rand.Read(buf)
	return buf
}
