package packet

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/proto/implant/commonpb"
	"github.com/golang/protobuf/proto"
	"io"
	"net"
)

const (
	SpiteStart = 0xd1
	SpiteEnd   = 0xd2
)

func ParseSpite(header, body []byte) (*commonpb.Spite, error) {
	if len(header) != 5 {
		return nil, errors.New("invalid header length")
	}
	if header[0] != SpiteStart {
		return nil, errors.New("invalid header start")
	}
	if body[len(body)-1] != SpiteEnd {
		return nil, errors.New("invalid body end")
	}

	spite := &commonpb.Spite{}
	err := proto.Unmarshal(body[:len(body)-1], spite)
	if err != nil {
		return nil, err
	}

	return spite, nil
}

func MarshalSpite(spite *commonpb.Spite) ([]byte, error) {
	var buf bytes.Buffer

	data, err := proto.Marshal(spite)
	if err != nil {
		return nil, err
	}

	buf.WriteByte(SpiteStart)
	binary.Write(&buf, binary.LittleEndian, len(data))
	buf.Write(data)
	buf.WriteByte(SpiteEnd)
	return buf.Bytes(), nil
}

func ReadSpite(conn net.Conn) (*commonpb.Spite, error) {
	spiteHeader := make([]byte, 5)
	n, err := io.ReadFull(conn, spiteHeader)
	if err != nil || n != 5 {
		logs.Log.Errorf("Socket error (read msg-length): %v", err)
		return nil, err
	}

	dataLength := int(binary.LittleEndian.Uint32(spiteHeader[4:]))
	if dataLength <= 0 {
		return nil, errors.New("zero data length")
	}

	dataBuf := make([]byte, dataLength)

	n, err = io.ReadFull(conn, dataBuf)

	if err != nil || n != dataLength {
		logs.Log.Errorf("Socket error (read data): %v", err)
		return nil, err
	}

	return ParseSpite(spiteHeader, dataBuf)
}

func WriteSpite(conn net.Conn, spite *commonpb.Spite) error {
	bs, err := MarshalSpite(spite)
	if err != nil {
		return err
	}
	_, err = conn.Write(bs)
	if err != nil {
		return err
	}
	return nil
}
