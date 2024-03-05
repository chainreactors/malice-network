package packet

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"

	"google.golang.org/protobuf/proto"
	"io"
	"net"
	"time"
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
	ErrNullSpites    = errors.New("parsed 0 spite")
)

func ParseMessage(body []byte) (proto.Message, error) {
	msg := &implantpb.Spites{}
	err := proto.Unmarshal(body, msg)
	if err != nil {
		return nil, err
	}
	if len(msg.Spites) == 0 {
		return nil, ErrNullSpites
	}
	return msg, nil
}

func ParseHeader(header []byte) ([]byte, int, error) {
	if len(header) != HeaderLength {
		return nil, 0, ErrInvalidHeader
	}
	if header[MsgStart] != StartDelimiter {
		return nil, 0, ErrInvalidStart
	}
	sessionId := header[MsgSessionStart:MsgSessionEnd]
	length := int(binary.LittleEndian.Uint32(header[MsgSessionEnd:]))
	return sessionId, length, nil
}

func MarshalMessage(sessionId []byte, msg proto.Message) ([]byte, error) {
	var buf bytes.Buffer

	data, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}

	buf.WriteByte(StartDelimiter)
	buf.Write(sessionId)
	err = binary.Write(&buf, binary.LittleEndian, int32(len(data)))
	if err != nil {
		return nil, err
	}

	buf.Write(data)
	buf.WriteByte(EndDelimiter)
	return buf.Bytes(), nil
}

func ReadHeader(conn net.Conn) ([]byte, int, error) {
	header := make([]byte, HeaderLength)
	n, err := io.ReadFull(conn, header)
	if err != nil || n != HeaderLength {
		return nil, 0, err
	}
	return ParseHeader(header)
}

func ReadHeaderWithTimeout(conn net.Conn, timeout time.Duration) ([]byte, int, error) {
	header := make([]byte, HeaderLength)
	conn.SetReadDeadline(time.Now().Add(timeout))
	n, err := io.ReadFull(conn, header)
	if err != nil || n != HeaderLength {
		return nil, 0, err
	}
	return ParseHeader(header)
}

func ReadMessage(conn net.Conn, length int) (proto.Message, error) {
	dataBuf := make([]byte, length+1)
	n, err := io.ReadFull(conn, dataBuf)

	if err != nil || n != length+1 {
		return nil, err
	}

	if dataBuf[length] != EndDelimiter {
		return nil, ErrInvalidEnd
	}
	return ParseMessage(dataBuf[:length])
}

func ReadMessageWithTimeout(conn net.Conn, length int, timeout time.Duration) (proto.Message, error) {
	dataBuf := make([]byte, length+1)
	conn.SetReadDeadline(time.Now().Add(timeout))
	n, err := io.ReadFull(conn, dataBuf)

	if err != nil || n != length+1 {
		return nil, err
	}

	if dataBuf[length] != EndDelimiter {
		return nil, ErrInvalidEnd
	}
	return ParseMessage(dataBuf[:length])
}

func ReadPacket(conn net.Conn) ([]byte, proto.Message, error) {
	sessionId, length, err := ReadHeader(conn)
	if err != nil {
		return nil, nil, err
	}
	msg, err := ReadMessage(conn, length)
	if err != nil {
		return nil, nil, err
	}
	return sessionId, msg, nil
}

func ReadPacketWithTimeout(conn net.Conn, timeout time.Duration) ([]byte, proto.Message, error) {
	sessionId, length, err := ReadHeaderWithTimeout(conn, timeout)
	if err != nil {
		return nil, nil, err
	}
	msg, err := ReadMessageWithTimeout(conn, length, timeout)
	if err != nil {
		return nil, nil, err
	}
	return sessionId, msg, nil
}

func WritePacket(conn net.Conn, msg proto.Message, sessionId []byte) error {
	bs, err := MarshalMessage(sessionId, msg)
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

func WritePacketWithTimeout(conn net.Conn, msg proto.Message, sessionId []byte, timeout time.Duration) error {
	bs, err := MarshalMessage(sessionId, msg)
	if err != nil {
		return err
	}
	conn.SetWriteDeadline(time.Now().Add(timeout))
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
