package cryptostream

import (
	"bytes"
	"io"
	"net"
)

func NewCryptoConn(conn net.Conn, cryptor Cryptor) *CryptoConn {
	return &CryptoConn{
		Conn:            conn,
		ReadWriteCloser: conn,
		Cryptor:         cryptor,
	}
}

func NewCryptoRWC(rwc io.ReadWriteCloser, cryptor Cryptor) *CryptoConn {
	return &CryptoConn{
		ReadWriteCloser: rwc,
		Cryptor:         cryptor,
	}
}

type CryptoConn struct {
	net.Conn
	io.ReadWriteCloser
	Cryptor
	readBuf []byte
}

func (sc *CryptoConn) Write(data []byte) (int, error) {
	encryptedData, err := sc.encrypt(data)
	if err != nil {
		return 0, err
	}

	return sc.ReadWriteCloser.Write(encryptedData)
}

func (sc *CryptoConn) Read(data []byte) (int, error) {
	// 1. If there is cached decrypted data from a previous over-read, serve it first
	if len(sc.readBuf) > 0 {
		n := copy(data, sc.readBuf)
		sc.readBuf = sc.readBuf[n:]
		return n, nil
	}

	// 2. Read new encrypted data from underlying connection and decrypt
	encryptedData := make([]byte, 1024)
	n, err := sc.ReadWriteCloser.Read(encryptedData)
	if n == 0 {
		return 0, err
	}

	decryptedData, err := sc.decrypt(encryptedData[:n])
	if err != nil {
		return 0, err
	}

	// 3. Return only the amount the caller requested, cache any overflow
	copied := copy(data, decryptedData)
	if copied < len(decryptedData) {
		sc.readBuf = append(sc.readBuf[:0], decryptedData[copied:]...)
	}
	return copied, nil
}

func (sc *CryptoConn) Close() error {
	return sc.ReadWriteCloser.Close()
}

func (sc *CryptoConn) encrypt(data []byte) ([]byte, error) {
	reader := bytes.NewReader(data)
	writer := &bytes.Buffer{}

	err := sc.Cryptor.Encrypt(reader, writer)
	if err != nil {
		return nil, err
	}

	return writer.Bytes(), nil
}

func (sc *CryptoConn) RemoteAddr() net.Addr {
	if sc.Conn != nil {
		return sc.Conn.RemoteAddr()
	} else if sc.ReadWriteCloser != nil {
		remote, ok := sc.ReadWriteCloser.(interface {
			RemoteAddr() net.Addr
		})
		if ok {
			return remote.RemoteAddr()
		}
	}
	return nil
}

// 解密数据
func (sc *CryptoConn) decrypt(data []byte) ([]byte, error) {
	reader := bytes.NewReader(data)
	writer := &bytes.Buffer{}

	// 使用加密器解密数据
	err := sc.Cryptor.Decrypt(reader, writer)
	if err != nil {
		return nil, err
	}
	return writer.Bytes(), nil
}
