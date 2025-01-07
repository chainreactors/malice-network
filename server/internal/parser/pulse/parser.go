package pulse

import (
	"bytes"
	"encoding/binary"
	"github.com/chainreactors/malice-network/helper/encoders"
	"github.com/chainreactors/malice-network/helper/encoders/hash"
	"github.com/chainreactors/malice-network/helper/errs"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/helper/utils/handler"
	"github.com/chainreactors/malice-network/helper/utils/peek"
)

const (
	MsgStart      = 0
	MsgMagicStart = 1
	MsgMagicEnd   = 5
	HeaderLength  = 9
)

func NewPulseParser() *PulseParser {
	return &PulseParser{
		StartDelimiter: 0x41,
		EndDelimiter:   0x42,
		Magic:          hash.BJD2Hash("beautiful"),
	}
}

type PulseParser struct {
	StartDelimiter byte
	EndDelimiter   byte
	Magic          uint32
}

func (parser *PulseParser) PeekHeader(conn *peek.Conn) (uint32, uint32, error) {
	header, err := conn.Peek(HeaderLength)
	if err != nil {
		return 0, 0, err
	}

	if header[MsgStart] != parser.StartDelimiter {
		return 0, 0, errs.ErrInvalidStart
	}
	magic := encoders.BytesToUint32(header[MsgMagicStart:MsgMagicEnd])
	artifact := binary.LittleEndian.Uint32(header[MsgMagicEnd:])
	return magic, artifact, nil
}

func (parser *PulseParser) ReadHeader(conn *peek.Conn) (uint32, uint32, error) {
	magic, artifact, err := parser.PeekHeader(conn)
	if err != nil {
		return 0, 0, err
	}
	if magic != parser.Magic {
		return 0, 0, errs.ErrInvalidMagic
	}
	if _, err := conn.Reader.Discard(HeaderLength); err != nil {
		return 0, 0, err
	}
	end := make([]byte, 1)
	n, err := conn.Read(end)
	if err != nil {
		return 0, 0, err
	}
	if n != 1 || end[0] != parser.EndDelimiter {
		return 0, 0, errs.ErrInvalidEnd
	}
	return magic, artifact, nil
}

func (parser *PulseParser) Parse(buf []byte) (*implantpb.Spites, error) {
	return nil, nil
}

func (parser *PulseParser) Marshal(spites *implantpb.Spites, magic uint32) ([]byte, error) {
	var buf bytes.Buffer
	if len(spites.Spites) == 0 {
		return nil, errs.ErrNullSpites
	}

	err := handler.AssertSpite(spites.Spites[0], types.MsgInit)
	if err != nil {
		return nil, err
	}
	data := spites.Spites[0].GetInit().Data
	buf.WriteByte(parser.StartDelimiter)
	binary.Write(&buf, binary.LittleEndian, encoders.Uint32ToBytes(magic))
	binary.Write(&buf, binary.LittleEndian, int32(len(data)))
	buf.Write(data)
	buf.WriteByte(parser.EndDelimiter)
	//logs.Log.Debugf("marshal %v %d bytes", buf.Bytes()[:9], len(data))
	return buf.Bytes(), nil
}
