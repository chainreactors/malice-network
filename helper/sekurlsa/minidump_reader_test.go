package sekurlsa

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestParseSystemInfoAcceptsNanodump48ByteStream(t *testing.T) {
	const (
		headerLen = 32
		dirLen    = 12
		infoLen   = 48
		infoRVA   = headerLen + dirLen
	)
	buf := make([]byte, infoRVA+infoLen)
	binary.LittleEndian.PutUint32(buf[0:4], miniDumpSignature)
	binary.LittleEndian.PutUint32(buf[4:8], 0xa793)
	binary.LittleEndian.PutUint32(buf[8:12], 1)
	binary.LittleEndian.PutUint32(buf[12:16], headerLen)
	binary.LittleEndian.PutUint32(buf[headerLen:headerLen+4], streamSystemInfo)
	binary.LittleEndian.PutUint32(buf[headerLen+4:headerLen+8], infoLen)
	binary.LittleEndian.PutUint32(buf[headerLen+8:headerLen+12], infoRVA)
	binary.LittleEndian.PutUint16(buf[infoRVA:infoRVA+2], 9) // AMD64
	binary.LittleEndian.PutUint32(buf[infoRVA+8:infoRVA+12], 10)
	binary.LittleEndian.PutUint32(buf[infoRVA+16:infoRVA+20], 17763)

	r, err := openReader(bytes.NewReader(buf), int64(len(buf)))
	if err != nil {
		t.Fatalf("openReader: %v", err)
	}
	if r.systemInfo.ProcessorArchitecture != 9 {
		t.Fatalf("arch = %d, want 9", r.systemInfo.ProcessorArchitecture)
	}
	if r.systemInfo.BuildNumber != 17763 {
		t.Fatalf("build = %d, want 17763", r.systemInfo.BuildNumber)
	}
}
