package parser

import (
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/errs"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/utils/peek"
	"github.com/chainreactors/malice-network/server/internal/parser/malefic"
	"github.com/chainreactors/malice-network/server/internal/parser/pulse"
	"io"
)

// PacketParser packet parser, like malefic, beacon ...
type PacketParser interface {
	PeekHeader(conn *peek.Conn) (uint32, int, error)
	ReadHeader(conn *peek.Conn) (uint32, int, error)
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

	n, err := conn.Write(bs)
	if err != nil {
		return err
	}
	if n != len(bs) {
		return fmt.Errorf("send error, expect send %d, real send %d", len(bs), n)
	}
	if len(bs) <= 1000 {
		logs.Log.Debugf("write packet to %s , %d bytes, %v", conn.RemoteAddr(), len(bs), msg)
	} else {
		logs.Log.Debugf("write packet to %s , %d bytes", conn.RemoteAddr(), len(bs))
	}

	return nil
}
