package malefic

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/errs"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/utils/peek"
	"github.com/chainreactors/malice-network/server/internal/stream"
	"github.com/gookit/config/v2"
	"google.golang.org/protobuf/proto"
)

const (
	MsgStart        = 0
	MsgSessionStart = 1
	MsgSessionEnd   = 5
	HeaderLength    = 9
)

func NewMaleficParser() *MaleficParser {
	return &MaleficParser{
		StartDelimiter: 0xd1,
		EndDelimiter:   0xd2,
	}
}

type MaleficParser struct {
	StartDelimiter byte
	EndDelimiter   byte
}

func (parser *MaleficParser) PeekHeader(conn *peek.Conn) (uint32, uint32, error) {
	header, err := conn.Peek(HeaderLength)
	if err != nil {
		return 0, 0, err
	}

	if header[MsgStart] != parser.StartDelimiter {
		return 0, 0, errs.ErrInvalidStart
	}
	sessionId := header[MsgSessionStart:MsgSessionEnd]
	length := binary.LittleEndian.Uint32(header[MsgSessionEnd:])
	return binary.LittleEndian.Uint32(sessionId), length + 1, nil
}

func (parser *MaleficParser) ReadHeader(conn *peek.Conn) (uint32, uint32, error) {
	sid, length, err := parser.PeekHeader(conn)
	if err != nil {
		return 0, 0, err
	}
	//logs.Log.Debugf("%v read packet from %s , %d bytes", sid, conn.RemoteAddr(), length)
	if length > uint32(config.Uint(consts.ConfigMaxPacketLength))+consts.KB*16 {
		return 0, 0, fmt.Errorf("%w,expect: %d, recv: %d", errs.ErrPacketTooLarge, config.Int(consts.ConfigMaxPacketLength), length)
	}
	if _, err := conn.Reader.Discard(HeaderLength); err != nil {
		return 0, 0, err
	}
	return sid, length, nil
}

func (parser *MaleficParser) Parse(buf []byte) (*implantpb.Spites, error) {
	length := len(buf) - 1
	if buf[length] != parser.EndDelimiter {
		return nil, errs.ErrInvalidEnd
	}
	buf = buf[:length]
	buf, err := cryptostream.Decompress(buf)
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
	data, err = cryptostream.Compress(data)
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
