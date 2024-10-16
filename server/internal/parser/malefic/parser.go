package malefic

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/utils/peek"
	"google.golang.org/protobuf/proto"
)

const (
	StartDelimiter  = 0xd1
	EndDelimiter    = 0xd2
	MsgStart        = 0
	MsgSessionStart = 1
	MsgSessionEnd   = 5
	HeaderLength    = 9
)

var (
	ErrInvalidStart  = errors.New("read invalid start delimiter")
	ErrInvalidEnd    = errors.New("read invalid end delimiter")
	ErrInvalidHeader = errors.New("invalid header")
	ErrNullSpites    = errors.New("parsed 0 spites")
	ErrInvalidId     = errors.New("invalid session id")
)

type MaleficParser struct{}

func (parser *MaleficParser) marshalMessage(msg *implantpb.Spites, sid []byte) ([]byte, error) {
	var buf bytes.Buffer
	if len(sid) != 4 {
		return nil, ErrInvalidId
	}

	data, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}

	buf.WriteByte(StartDelimiter)
	buf.Write(sid)
	err = binary.Write(&buf, binary.LittleEndian, int32(len(data)))
	if err != nil {
		return nil, err
	}

	buf.Write(data)
	buf.WriteByte(EndDelimiter)
	return buf.Bytes(), nil
}

func (parser *MaleficParser) PeekHeader(conn *peek.Conn) ([]byte, int, error) {
	header, err := conn.Peek(HeaderLength)
	if err != nil {
		return nil, 0, err
	}

	if header[MsgStart] != StartDelimiter {
		return nil, 0, ErrInvalidStart
	}
	sessionId := header[MsgSessionStart:MsgSessionEnd]
	length := int(binary.LittleEndian.Uint32(header[MsgSessionEnd:]))
	return sessionId, length + 1, nil
}

func (parser *MaleficParser) Parse(buf []byte) (*implantpb.Spites, error) {
	length := len(buf) - 1
	if buf[length] != EndDelimiter {
		return nil, ErrInvalidEnd
	}
	buf = buf[:length]
	spites := &implantpb.Spites{}
	err := proto.Unmarshal(buf, spites)
	if err != nil {
		return nil, err
	}
	return spites, nil
}

func (parser *MaleficParser) Marshal(spites *implantpb.Spites, sid []byte) ([]byte, error) {
	return parser.marshalMessage(spites, sid)
}
