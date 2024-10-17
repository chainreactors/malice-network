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
	// 使用 Cryptor 加密数据
	encryptedData, err := sc.encrypt(data)
	if err != nil {
		return 0, err
	}

	// 将加密后的数据发送到底层连接
	return sc.Conn.Write(encryptedData)
}

// Read 方法从底层连接读取数据并解密
func (sc *CryptoConn) Read(data []byte) (int, error) {
	// 读取加密数据
	encryptedData := make([]byte, 1024) // 假设读取缓冲区的大小
	n, err := sc.Conn.Read(encryptedData)
	if err != nil {
		return 0, err
	}

	// 解密读取到的数据
	decryptedData, err := sc.decrypt(encryptedData[:n])
	if err != nil {
		return 0, err
	}

	// 将解密后的数据拷贝到传入的缓冲区
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
