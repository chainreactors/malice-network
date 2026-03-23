package malefic

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/malice-network/helper/utils/compress"
	"github.com/gookit/config/v2"
	"google.golang.org/protobuf/proto"
	"io"
	"strings"
)

const (
	MsgStart              = 0
	MsgSessionStart       = 1
	MsgSessionEnd         = 5
	HeaderLength          = 9
	DefaultStartDelimiter = 0xd1
	DefaultEndDelimiter   = 0xd2
)

func NewMaleficParser() *MaleficParser {
	return &MaleficParser{
		StartDelimiter: DefaultStartDelimiter,
		EndDelimiter:   DefaultEndDelimiter,
	}
}

type MaleficParser struct {
	StartDelimiter   byte
	EndDelimiter     byte
	MaxPacketLength  uint32
	keyPair          *clientpb.KeyPair // Age 密钥对，用于加解密
	privateKeys      []string
}

// maxPacketLen returns the per-pipeline limit or falls back to global config.
func (parser *MaleficParser) maxPacketLen() uint32 {
	if parser.MaxPacketLength > 0 {
		return parser.MaxPacketLength
	}
	return uint32(config.Uint(consts.ConfigMaxPacketLength))
}

// WithSecure 设置 Age 密钥对用于加解密，返回新的 parser 实例
func (parser *MaleficParser) WithSecure(keyPair *clientpb.KeyPair) {
	parser.keyPair = keyPair
	parser.privateKeys = splitPrivateKeys(keyPair)
}

func splitPrivateKeys(keyPair *clientpb.KeyPair) []string {
	if keyPair == nil {
		return nil
	}

	raw := strings.TrimSpace(keyPair.PrivateKey)
	if raw == "" {
		return nil
	}

	seen := make(map[string]struct{})
	privateKeys := make([]string, 0, 2)
	for _, privateKey := range strings.Split(raw, "\n") {
		privateKey = strings.TrimSpace(privateKey)
		if privateKey == "" {
			continue
		}
		if _, ok := seen[privateKey]; ok {
			continue
		}
		seen[privateKey] = struct{}{}
		privateKeys = append(privateKeys, privateKey)
	}
	return privateKeys
}

func ParseSid(data []byte) uint32 {
	if len(data) < MsgSessionEnd {
		return 0
	}
	sessionId := data[MsgSessionStart:MsgSessionEnd]
	return binary.LittleEndian.Uint32(sessionId)
}

func (parser *MaleficParser) readHeader(conn io.ReadWriteCloser) (uint32, uint32, error) {
	header := make([]byte, HeaderLength)
	n, err := io.ReadFull(conn, header)
	if err != nil {
		return 0, 0, err
	}
	if n != HeaderLength {
		return 0, 0, fmt.Errorf("read header error, expect %d, real %d", HeaderLength, n)
	}

	if header[MsgStart] != parser.StartDelimiter {
		return 0, 0, types.ErrInvalidStart
	}
	sessionId := ParseSid(header)
	length := binary.LittleEndian.Uint32(header[MsgSessionEnd:])
	if length > parser.maxPacketLen()+consts.KB*16 {
		return 0, 0, fmt.Errorf("%w,expect: %d, recv: %d", types.ErrPacketTooLarge, parser.maxPacketLen(), length)
	}

	return sessionId, length + 1, nil
}

func (parser *MaleficParser) ReadHeader(conn io.ReadWriteCloser) (uint32, uint32, error) {
	sid, length, err := parser.readHeader(conn)
	if err != nil {
		return 0, 0, err
	}
	//logs.Log.Debugf("%v read packet from %s , %d bytes", sid, conn.RemoteAddr(), length)
	if length > parser.maxPacketLen()+consts.KB*16+1 {
		return 0, 0, fmt.Errorf("%w,expect: %d, recv: %d", types.ErrPacketTooLarge, parser.maxPacketLen(), length)
	}
	return sid, length, nil
}

func (parser *MaleficParser) Parse(buf []byte) (*implantpb.Spites, error) {
	if len(buf) == 0 {
		return nil, fmt.Errorf("empty payload")
	}
	length := len(buf) - 1
	if buf[length] != parser.EndDelimiter {
		return nil, types.ErrInvalidEnd
	}
	buf = buf[:length]

	if len(parser.privateKeys) > 0 {
		var decErr error
		decrypted := false
		for _, privateKey := range parser.privateKeys {
			decryptedBuf, err := cryptography.AgeDecrypt(privateKey, buf)
			if err == nil {
				buf = decryptedBuf
				decrypted = true
				break
			}
			decErr = err
		}

		if !decrypted && decErr != nil {
			// 兼容没有开启secure的session
			logs.Log.Debugf("trying plaintext: %v", decErr)
		}
	}

	buf, err := compress.Decompress(buf)
	if err != nil {
		logs.Log.Debugf("trying plaintext: %v", err)
	}

	spites := &implantpb.Spites{}
	err = proto.Unmarshal(buf, spites)
	if err != nil {
		return nil, err
	}
	return spites, nil
}

func (parser *MaleficParser) Marshal(spites *implantpb.Spites, sid uint32) ([]byte, error) {
	var buf bytes.Buffer

	data, err := proto.Marshal(spites)
	if err != nil {
		return nil, err
	}

	data, err = compress.Compress(data)
	if err != nil {
		return nil, err
	}

	if parser.keyPair != nil && parser.keyPair.PublicKey != "" && parser.keyPair.PrivateKey != "" {
		encryptedData, encErr := cryptography.AgeEncrypt(parser.keyPair.PublicKey, data)
		if encErr != nil {
			logs.Log.Debugf("%v", encErr)
			// 加密失败时使用明文（兼容性）
		} else {
			data = encryptedData
			logs.Log.Debugf("%d bytes", len(data))
		}
	}

	// 4. 构建最终的数据包
	buf.WriteByte(parser.StartDelimiter)
	err = binary.Write(&buf, binary.LittleEndian, sid)
	if err != nil {
		return nil, err
	}

	err = binary.Write(&buf, binary.LittleEndian, int32(len(data)))
	if err != nil {
		return nil, err
	}

	buf.Write(data)
	buf.WriteByte(parser.EndDelimiter)
	//logs.Log.Debugf("marshal %v %d bytes", buf.Bytes()[:9], len(data))
	return buf.Bytes(), nil
}
