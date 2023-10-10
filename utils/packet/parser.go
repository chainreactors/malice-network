package packet

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/proto/implant/commonpb"
	"google.golang.org/protobuf/proto"
	"io"
	"net"
)

const (
	StartDelimiter = 0xd1
	EndDelimiter   = 0xd2
	MsgType        = 5
	MsgStart       = 0
	HeaderLength   = 9
)

func ParseMessage(header, body []byte) (proto.Message, error) {
	if len(header) != HeaderLength {
		return nil, errors.New("invalid header length")
	}
	if header[MsgStart] != StartDelimiter {
		return nil, errors.New("invalid header start")
	}
	if body[len(body)-1] != EndDelimiter {
		return nil, errors.New("invalid body end")
	}
	var msg proto.Message
	if header[MsgType] == 1 {
		msg = &commonpb.Spite{}
		err := proto.Unmarshal(body[:len(body)-1], msg)
		if err != nil {
			return nil, err
		}
	} else if header[MsgType] == 2 {
		msg = &commonpb.Promise{}
		err := proto.Unmarshal(body[:len(body)-1], msg)
		if err != nil {
			return nil, err
		}
	}

	return msg, nil
}

func MarshalMessage(msg proto.Message) ([]byte, error) {
	var buf bytes.Buffer

	data, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}

	buf.WriteByte(StartDelimiter)
	err = binary.Write(&buf, binary.LittleEndian, int32(len(data)))
	if err != nil {
		return nil, err
	}

	switch msg.(type) {
	case *commonpb.Spite:
		err = binary.Write(&buf, binary.LittleEndian, int8(1))
	case *commonpb.Promise:
		err = binary.Write(&buf, binary.LittleEndian, int8(2))
	default:
		err = binary.Write(&buf, binary.LittleEndian, int8(0))
	}

	buf.Write(make([]byte, 3))
	if err != nil {
		return nil, err
	}
	buf.Write(data)
	buf.WriteByte(EndDelimiter)
	return buf.Bytes(), nil
}

func ReadMessage(conn net.Conn) (proto.Message, error) {
	header := make([]byte, HeaderLength)
	n, err := io.ReadFull(conn, header)
	if err != nil || n != HeaderLength {
		logs.Log.Errorf("socket error (read msg-length): %v", err)
		return nil, err
	}

	dataLength := int(binary.LittleEndian.Uint32(header[1:5])) + 1
	if dataLength <= 0 {
		return nil, errors.New("zero data length")
	}

	dataBuf := make([]byte, dataLength)

	n, err = io.ReadFull(conn, dataBuf)

	if err != nil || n != dataLength {
		logs.Log.Errorf("socket error (read data): %v", err)
		return nil, err
	}

	return ParseMessage(header, dataBuf)
}

func WriteMessage(conn net.Conn, msg proto.Message) error {
	bs, err := MarshalMessage(msg)
	if err != nil {
		return err
	}
	_, err = conn.Write(bs)
	if err != nil {
		return err
	}
	return nil
}
