package parser

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/server/internal/parser/malefic"
)

type headerThenEOFConn struct {
	header         []byte
	offset         int
	maxReadRequest int
}

func (c *headerThenEOFConn) Read(p []byte) (int, error) {
	if len(p) > c.maxReadRequest {
		c.maxReadRequest = len(p)
	}
	if c.offset == len(c.header) {
		return 0, io.EOF
	}
	n := copy(p, c.header[c.offset:])
	c.offset += n
	return n, nil
}

func (c *headerThenEOFConn) Write(p []byte) (int, error) { return len(p), nil }
func (c *headerThenEOFConn) Close() error                { return nil }

func TestReadPacket_DeclaredLengthUsesBoundedReadBuffer(t *testing.T) {
	const configuredLimit = uint32(1024)
	declaredLength := uint32(payloadReadChunkSize * 4)

	header := make([]byte, malefic.HeaderLength)
	header[malefic.MsgStart] = malefic.DefaultStartDelimiter
	binary.LittleEndian.PutUint32(header[malefic.MsgSessionStart:malefic.MsgSessionEnd], 1)
	binary.LittleEndian.PutUint32(header[malefic.MsgSessionEnd:], declaredLength)

	p, err := NewParser(consts.ImplantMalefic)
	if err != nil {
		t.Fatalf("NewParser returned an unexpected error: %v", err)
	}
	p.WithMaxPacketLength(configuredLimit)
	conn := &headerThenEOFConn{header: header}

	_, _, err = p.ReadPacket(conn)
	if !errors.Is(err, io.EOF) {
		t.Fatalf("ReadPacket error = %v, want EOF after the header", err)
	}
	if conn.maxReadRequest > payloadReadChunkSize {
		t.Fatalf("ReadPacket requested a %d-byte read buffer, want at most %d", conn.maxReadRequest, payloadReadChunkSize)
	}
}

func TestReadPacket_AcceptsValidFrameLargerThanPacketLength(t *testing.T) {
	const configuredPacketLength = uint32(1024)
	p, err := NewParser(consts.ImplantMalefic)
	if err != nil {
		t.Fatalf("NewParser returned an unexpected error: %v", err)
	}
	p.WithMaxPacketLength(configuredPacketLength)

	stdout := make([]byte, payloadReadChunkSize*2)
	x := uint32(1)
	for i := range stdout {
		x ^= x << 13
		x ^= x >> 17
		x ^= x << 5
		stdout[i] = byte(x)
	}
	spites := &implantpb.Spites{Spites: []*implantpb.Spite{{
		TaskId: 1,
		Body: &implantpb.Spite_ExecResponse{ExecResponse: &implantpb.ExecResponse{
			Stdout: stdout,
		}},
	}}}
	raw, err := p.Marshal(spites, 1)
	if err != nil {
		t.Fatalf("Marshal returned an unexpected error: %v", err)
	}
	declaredLength := binary.LittleEndian.Uint32(raw[malefic.MsgSessionEnd:malefic.HeaderLength])
	warningThreshold := configuredPacketLength + uint32(consts.KB)*16
	if declaredLength <= warningThreshold {
		t.Fatalf("test frame length = %d, want larger than chunking threshold %d", declaredLength, warningThreshold)
	}

	_, got, err := p.ReadPacket(&rwcBuf{Buffer: bytes.NewBuffer(raw)})
	if err != nil {
		t.Fatalf("ReadPacket rejected a valid large frame: %v", err)
	}
	if !bytes.Equal(got.Spites[0].GetExecResponse().GetStdout(), stdout) {
		t.Fatal("ReadPacket changed the large exec response payload")
	}
}
