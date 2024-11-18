package cryptostream

import (
	"bytes"
	"net"
)

func NewCryptoConn(conn net.Conn, cryptor Cryptor) *CryptoConn {
	return &CryptoConn{
		Conn:    conn,
		Cryptor: cryptor,
	}
}

type CryptoConn struct {
	net.Conn

	Cryptor
}

func (sc *CryptoConn) Write(data []byte) (int, error) {
	encryptedData, err := sc.encrypt(data)
	if err != nil {
		return 0, err
	}

	return sc.Conn.Write(encryptedData)
}

// Read 方法从底层连接读取数据并解密
func (sc *CryptoConn) Read(data []byte) (int, error) {
	encryptedData := make([]byte, 1024)
	n, err := sc.Conn.Read(encryptedData)
	if err != nil {
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
