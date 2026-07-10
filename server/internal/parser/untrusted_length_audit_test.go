//go:build audit

package parser

import (
	"encoding/binary"
	"io"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/malice-network/server/internal/parser/malefic"
)

type auditHeaderThenEOFConn struct {
	header         []byte
	offset         int
	maxReadRequest int
}

func (c *auditHeaderThenEOFConn) Read(p []byte) (int, error) {
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

func (c *auditHeaderThenEOFConn) Write(p []byte) (int, error) { return len(p), nil }
func (c *auditHeaderThenEOFConn) Close() error                { return nil }

func TestAuditReadPacket_RejectsOversizedDeclarationBeforePayloadAllocation(t *testing.T) {
	const configuredLimit = uint32(1024)
	declaredLength := configuredLimit + uint32(consts.KB)*16 + 1

	header := make([]byte, malefic.HeaderLength)
	header[malefic.MsgStart] = malefic.DefaultStartDelimiter
	binary.LittleEndian.PutUint32(header[malefic.MsgSessionStart:malefic.MsgSessionEnd], 1)
	binary.LittleEndian.PutUint32(header[malefic.MsgSessionEnd:], declaredLength)

	p, err := NewParser(consts.ImplantMalefic)
	if err != nil {
		t.Fatalf("NewParser returned an unexpected error: %v", err)
	}
	p.WithMaxPacketLength(configuredLimit)
	conn := &auditHeaderThenEOFConn{header: header}

	_, _, _ = p.ReadPacket(conn)
	if conn.maxReadRequest > malefic.HeaderLength {
		t.Fatalf("ReadPacket requested a %d-byte payload buffer for an oversized declaration; want rejection after the %d-byte header", conn.maxReadRequest, malefic.HeaderLength)
	}
}
