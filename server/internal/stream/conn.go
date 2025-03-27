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
}

func (sc *CryptoConn) Write(data []byte) (int, error) {
	encryptedData, err := sc.encrypt(data)
	if err != nil {
		return 0, err
	}

	return sc.ReadWriteCloser.Write(encryptedData)
}

// Read 方法从底层连接读取数据并解密
func (sc *CryptoConn) Read(data []byte) (int, error) {
	encryptedData := make([]byte, 1024)
	n, err := sc.ReadWriteCloser.Read(encryptedData)
	if n == 0 {
		return 0, err
	}

	// 解密读取到的数据
	decryptedData, err := sc.decrypt(encryptedData[:n])
	if err != nil {
		return 0, err
	}

	copy(data, decryptedData)
	return len(decryptedData), nil
}

func (sc *CryptoConn) Close() error {
	return sc.ReadWriteCloser.Close()
}

// 加密数据
func (sc *CryptoConn) encrypt(data []byte) ([]byte, error) {
	reader := bytes.NewReader(data)
	writer := &bytes.Buffer{}

	// 使用加密器加密数据
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
