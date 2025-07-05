package malefic

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/errs"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/utils/compress"
	"github.com/gookit/config/v2"
	"google.golang.org/protobuf/proto"
	"io"
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
	StartDelimiter byte
	EndDelimiter   byte
}

func ParseSid(data []byte) uint32 {
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
		return 0, 0, errs.ErrInvalidStart
	}
	sessionId := ParseSid(header)
	length := binary.LittleEndian.Uint32(header[MsgSessionEnd:])
	if length > uint32(config.Uint(consts.ConfigMaxPacketLength))+consts.KB*16 {
		return 0, 0, fmt.Errorf("%w,expect: %d, recv: %d", errs.ErrPacketTooLarge, config.Int(consts.ConfigMaxPacketLength), length)
	}

	return sessionId, length + 1, nil
}

func (parser *MaleficParser) ReadHeader(conn io.ReadWriteCloser) (uint32, uint32, error) {
	sid, length, err := parser.readHeader(conn)
	if err != nil {
		return 0, 0, err
	}
	//logs.Log.Debugf("%v read packet from %s , %d bytes", sid, conn.RemoteAddr(), length)
	if length > uint32(config.Uint(consts.ConfigMaxPacketLength))+consts.KB*16 {
		return 0, 0, fmt.Errorf("%w,expect: %d, recv: %d", errs.ErrPacketTooLarge, config.Int(consts.ConfigMaxPacketLength), length)
	}
	return sid, length, nil
}

func (parser *MaleficParser) Parse(buf []byte) (*implantpb.Spites, error) {
	length := len(buf) - 1
	if buf[length] != parser.EndDelimiter {
		return nil, errs.ErrInvalidEnd
	}
	buf = buf[:length]
	buf, err := compress.Decompress(buf)
	if err != nil {
		return nil, err
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
