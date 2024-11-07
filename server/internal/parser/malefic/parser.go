package malefic

import (
	"bytes"
	"encoding/binary"
	"errors"
	cryptostream "github.com/chainreactors/malice-network/helper/cryptography/stream"
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

func (parser *MaleficParser) PeekHeader(conn *peek.Conn) (uint32, int, error) {
	header, err := conn.Peek(HeaderLength)
	if err != nil {
		return 0, 0, err
	}

	if header[MsgStart] != StartDelimiter {
		return 0, 0, ErrInvalidStart
	}
	sessionId := header[MsgSessionStart:MsgSessionEnd]
	length := int(binary.LittleEndian.Uint32(header[MsgSessionEnd:]))
	return binary.LittleEndian.Uint32(sessionId), length + 1, nil
}

func (parser *MaleficParser) Parse(buf []byte) (*implantpb.Spites, error) {
	length := len(buf) - 1
	if buf[length] != EndDelimiter {
		return nil, ErrInvalidEnd
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
	buf.WriteByte(StartDelimiter)
	binary.Write(&buf, binary.LittleEndian, sid)
	err = binary.Write(&buf, binary.LittleEndian, int32(len(data)))
	if err != nil {
		return nil, err
	}

	buf.Write(data)
	buf.WriteByte(EndDelimiter)
	//logs.Log.Debugf("marshal %v %d bytes", buf.Bytes()[:9], len(data))
	return buf.Bytes(), nil
}
