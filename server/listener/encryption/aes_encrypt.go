package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"fmt"
	"golang.org/x/crypto/pbkdf2"
	"io"
)

var (
	DefaultSalt = "crypto"
)

// NewAESWriter returns a new Writer that encrypts bytes to w.
func NewAESWriter(w io.Writer, key []byte) (*AESWriter, error) {
	key = pbkdf2.Key(key, []byte(DefaultSalt), 64, aes.BlockSize, sha1.New)

	// random iv
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	return &AESWriter{
		w: w,
		enc: &cipher.StreamWriter{
			S: cipher.NewCFBEncrypter(block, iv),
			W: w,
		},
		key: key,
		iv:  iv,
	}, nil
}

// Writer is an io.Writer that can write encrypted bytes.
// Now it only support aes-128-cfb.
type AESWriter struct {
	w      io.Writer
	enc    *cipher.StreamWriter
	key    []byte
	iv     []byte
	ivSend bool
	err    error
}

// Write satisfies the io.Writer interface.
func (w *AESWriter) Write(p []byte) (nRet int, errRet error) {
	if w.err != nil {
		return 0, w.err
	}

	// When write is first called, iv will be written to w.w
	if !w.ivSend {
		w.ivSend = true
		_, errRet = w.w.Write(w.iv)
		if errRet != nil {
			w.err = errRet
			return
		}
	}

	nRet, errRet = w.enc.Write(p)
	if errRet != nil {
		w.err = errRet
	}
	return
}

func Encode(s, key []byte) ([]byte, error) {
	key = pbkdf2.Key(key, []byte(DefaultSalt), 64, aes.BlockSize, sha1.New)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	ciphertext := make([]byte, aes.BlockSize+len(s))
	// random iv
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], s)
	return ciphertext, nil
}

// NewAESReader returns a new Reader that decrypts bytes from r
func NewAESReader(r io.Reader, key []byte) *AESReader {
	key = pbkdf2.Key(key, []byte(DefaultSalt), 64, aes.BlockSize, sha1.New)

	return &AESReader{
		r:   r,
		key: key,
	}
}

// Reader is an io.Reader that can read encrypted bytes.
// Now it only supports aes-128-cfb.
type AESReader struct {
	r   io.Reader
	dec *cipher.StreamReader
	key []byte
	iv  []byte
	err error
}

// Read satisfies the io.Reader interface.
func (r *AESReader) Read(p []byte) (nRet int, errRet error) {
	if r.err != nil {
		return 0, r.err
	}

	if r.dec == nil {
		iv := make([]byte, aes.BlockSize)
		if _, errRet = io.ReadFull(r.r, iv); errRet != nil {
			return
		}
		r.iv = iv

		block, err := aes.NewCipher(r.key)
		if err != nil {
			errRet = err
			return
		}
		r.dec = &cipher.StreamReader{
			S: cipher.NewCFBDecrypter(block, iv),
			R: r.r,
		}
	}

	nRet, errRet = r.dec.Read(p)
	if errRet != nil {
		r.err = errRet
	}
	return
}

// decode bytes by aes cfb
func Decode(s, key []byte) ([]byte, error) {
	key = pbkdf2.Key(key, []byte(DefaultSalt), 64, aes.BlockSize, sha1.New)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	if len(s) < aes.BlockSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	iv := s[:aes.BlockSize]
	s = s[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(s, s)
	return s, nil
}
