package parser

import (
	"errors"
	"fmt"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"io"
	"net"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/errs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/server/internal/parser/malefic"
	"github.com/chainreactors/malice-network/server/internal/parser/pulse"
)

// DetectProtocol 检测协议类型
func DetectProtocol(data []byte) (*MessageParser, error) {
	// Malefic 协议以 0xd1 开头
	if data[0] == malefic.DefaultStartDelimiter {
		return NewParser(consts.ImplantMalefic)
	}

	// Pulse 协议以 0x41 开头
	if data[0] == pulse.DefaultStartDelimiter {
		return NewParser(consts.ImplantPulse)
	}

	return nil, errors.New("unknown protocol")
}

// PacketParser packet parser, like malefic, beacon ...
type PacketParser interface {
	//PeekHeader(conn io.ReadWriteCloser) (uint32, uint32, error)
	ReadHeader(conn io.ReadWriteCloser) (uint32, uint32, error)
	Parse([]byte) (*implantpb.Spites, error)
	Marshal(*implantpb.Spites, uint32) ([]byte, error)
}

func NewParser(name string) (*MessageParser, error) {
	switch name {
	case consts.ImplantMalefic:
		return &MessageParser{Implant: name, PacketParser: malefic.NewMaleficParser()}, nil
	case consts.ImplantPulse:
		return &MessageParser{Implant: name, PacketParser: pulse.NewPulseParser()}, nil
	default:
		return nil, errs.ErrInvalidImplant
	}
}

type MessageParser struct {
	Implant string
	PacketParser
}

func (parser *MessageParser) ReadMessage(conn io.ReadWriteCloser, length uint32) (*implantpb.Spites, error) {
	buf := make([]byte, length)
	_, err := io.ReadFull(conn, buf)
	if err != nil {
		return nil, err
	}
	return parser.Parse(buf)
}

func (parser *MessageParser) ReadPacket(conn io.ReadWriteCloser) (uint32, *implantpb.Spites, error) {
	sessionId, length, err := parser.ReadHeader(conn)
	if err != nil {
		return 0, nil, err
	}

	buf := make([]byte, length)
	_, err = io.ReadFull(conn, buf)
	if err != nil {
		return 0, nil, err
	}

	msg, err := parser.Parse(buf)
	return sessionId, msg, nil
}

func (parser *MessageParser) WritePacket(conn net.Conn, msg *implantpb.Spites, sid uint32) error {
	bs, err := parser.Marshal(msg, sid)
	if err != nil {
		return err
	}

	n, err := conn.Write(bs)
	if err != nil {
		return err
	}
	if n != len(bs) {
		return fmt.Errorf("send error, expect send %d, real send %d", len(bs), n)
	}
	if len(bs) <= 1000 {
		logs.Log.Debugf("write packet to %s , %d bytes", conn.RemoteAddr(), len(bs))
	} else {
		logs.Log.Debugf("write packet to %s , %d bytes", conn.RemoteAddr(), len(bs))
	}

	return nil
}

// SecureWritePacket - 使用密钥对对整个Spites进行Age加密后写入
func (parser *MessageParser) SecureWritePacket(conn net.Conn, msg *implantpb.Spites, sid uint32, keyPair *clientpb.KeyPair) error {
	// 1. 将Spites序列化为protobuf字节
	bs, err := parser.Marshal(msg, sid)
	if err != nil {
		return err
	}

	// 2. 如果有密钥对，使用Age加密序列化后的字节
	if keyPair != nil && keyPair.PublicKey != "" && keyPair.PrivateKey != "" {
		encryptedBs, encErr := cryptography.AgeEncrypt(keyPair.PublicKey, bs)
		if encErr != nil {
			logs.Log.Errorf("failed to encrypt with age keyPair %s: %v", keyPair.KeyId, encErr)
			// 加密失败时发送明文（兼容性）
		} else {
			bs = encryptedBs
			logs.Log.Debugf("encrypted Spites protobuf data with age keyPair %s, %d bytes", keyPair.KeyId, len(bs))
		}
	}

	// 3. 写入数据
	n, err := conn.Write(bs)
	if err != nil {
		return err
	}
	if n != len(bs) {
		return fmt.Errorf("send error, expect send %d, real send %d", len(bs), n)
	}

	if len(bs) <= 1000 {
		logs.Log.Debugf("write packet to %s , %d bytes", conn.RemoteAddr(), len(bs))
	} else {
		logs.Log.Debugf("write packet to %s , %d bytes", conn.RemoteAddr(), len(bs))
	}

	return nil
}

// SecureReadPacket - 读取Age加密的数据并解密为Spites
func (parser *MessageParser) SecureReadPacket(conn io.ReadWriteCloser, keyPair *clientpb.KeyPair) (uint32, *implantpb.Spites, error) {
	// 1. 读取header获取sessionId和数据长度
	sessionId, length, err := parser.ReadHeader(conn)
	if err != nil {
		return 0, nil, err
	}

	// 2. 读取加密的数据
	buf := make([]byte, length)
	_, err = io.ReadFull(conn, buf)
	if err != nil {
		return 0, nil, err
	}

	// 3. 如果有密钥对，使用Age解密数据
	if keyPair != nil && keyPair.PublicKey != "" && keyPair.PrivateKey != "" {
		decryptedBuf, decErr := cryptography.AgeDecrypt(keyPair.PrivateKey, buf)
		if decErr != nil {
			// 解密失败，尝试明文解析（兼容性）
			logs.Log.Debugf("failed to decrypt with age keyPair %s, trying plaintext: %v", keyPair.KeyId, decErr)
		} else {
			buf = decryptedBuf
			logs.Log.Debugf("decrypted Spites protobuf data with age keyPair %s, %d bytes", keyPair.KeyId, len(buf))
		}
	}

	// 4. 反序列化为Spites
	msg, err := parser.Parse(buf)
	return sessionId, msg, err
}
