package parser

import (
	"errors"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/utils/peek"
	"github.com/chainreactors/malice-network/server/internal/parser/malefic"
	"github.com/gookit/config/v2"
	"io"
)

var (
	ErrInvalidImplant = errors.New("invalid implant")
	ErrPacketTooLarge = errors.New("packet too large")
)

// PacketParser packet parser, like malefic, beacon ...
type PacketParser interface {
	PeekHeader(conn *peek.Conn) (uint32, int, error)
	Parse([]byte) (*implantpb.Spites, error)
	Marshal(*implantpb.Spites, uint32) ([]byte, error)
}

func NewParser(conn *peek.Conn) (*MessageParser, error) {
	discriminator, err := conn.Peek(9)
	if err != nil {
		return nil, err
	}

	switch discriminator[0] {
	case malefic.StartDelimiter:
		return &MessageParser{
			Implant:      consts.ImplantMalefic,
			PacketParser: &malefic.MaleficParser{},
		}, nil
	default:
		return nil, ErrInvalidImplant
	}
}

type MessageParser struct {
	Implant string
	PacketParser
}

func (parser *MessageParser) ReadHeader(conn *peek.Conn) (uint32, int, error) {
	switch parser.Implant {
	case consts.ImplantMalefic:
		sid, length, err := parser.PeekHeader(conn)
		if err != nil {
			return 0, 0, err
		}
		//logs.Log.Debugf("%v read packet from %s , %d bytes", sid, conn.RemoteAddr(), length)
		if length > config.Int(consts.ConfigMaxPacketLength) {
			return 0, 0, ErrPacketTooLarge
		}
		if n, err := conn.Reader.Discard(malefic.HeaderLength); err != nil {
			return 0, n, err
		}
		return sid, length, nil
	default:
		return 0, 0, ErrInvalidImplant
	}
}

func (parser *MessageParser) ReadMessage(conn *peek.Conn, length int) (*implantpb.Spites, error) {
	buf := make([]byte, length)
	_, err := io.ReadFull(conn, buf)
	if err != nil {
		return nil, err
	}
	return parser.Parse(buf)
}

func (parser *MessageParser) ReadPacket(conn *peek.Conn) (uint32, *implantpb.Spites, error) {
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

func (parser *MessageParser) WritePacket(conn *peek.Conn, msg *implantpb.Spites, sid uint32) error {
	bs, err := parser.Marshal(msg, sid)
	if err != nil {
		return err
	}

	_, err = conn.Write(bs)
	if err != nil {
		return err
	}

	if len(bs) <= 1000 {
		logs.Log.Debugf("write packet to %s , %d bytes, %v", conn.RemoteAddr(), len(bs), msg)
	} else {
		logs.Log.Debugf("write packet to %s , %d bytes", conn.RemoteAddr(), len(bs))
	}

	return nil
}
