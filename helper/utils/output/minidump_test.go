package output

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestRestoreInvalidMinidumpSignature(t *testing.T) {
	tests := []struct {
		name      string
		dump      []byte
		want      bool
		wantSig   uint32
		wantVers  uint32
		unchanged bool
	}{
		{
			name:     "invalid nanodump header",
			dump:     fakeMinidump(0x11223344, 0x55667788, 3, 32, 7),
			want:     true,
			wantSig:  minidumpSignature,
			wantVers: minidumpVersion,
		},
		{
			name:      "already valid signature",
			dump:      fakeMinidump(minidumpSignature, minidumpVersion, 3, 32, 7),
			want:      false,
			unchanged: true,
		},
		{
			name:      "random file",
			dump:      []byte("not a minidump at all, just some text payload"),
			want:      false,
			unchanged: true,
		},
		{
			name:      "too small",
			dump:      []byte("MD"),
			want:      false,
			unchanged: true,
		},
		{
			name:      "implausible stream count",
			dump:      fakeMinidump(0x11223344, 0x55667788, 0, 32, 7),
			want:      false,
			unchanged: true,
		},
		{
			name:      "implausible stream type",
			dump:      fakeMinidump(0x11223344, 0x55667788, 3, 32, 0x12345678),
			want:      false,
			unchanged: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "lsass.dmp")
			if err := os.WriteFile(path, tt.dump, 0o600); err != nil {
				t.Fatalf("WriteFile: %v", err)
			}

			got, err := RestoreInvalidMinidumpSignature(path)
			if err != nil {
				t.Fatalf("RestoreInvalidMinidumpSignature: %v", err)
			}
			if got != tt.want {
				t.Fatalf("restored = %v, want %v", got, tt.want)
			}

			gotBytes, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile: %v", err)
			}
			if tt.unchanged {
				if string(gotBytes) != string(tt.dump) {
					t.Fatalf("file was modified")
				}
				return
			}
			if binary.LittleEndian.Uint32(gotBytes[0:4]) != tt.wantSig {
				t.Fatalf("signature = 0x%x, want 0x%x", binary.LittleEndian.Uint32(gotBytes[0:4]), tt.wantSig)
			}
			if binary.LittleEndian.Uint32(gotBytes[4:8]) != tt.wantVers {
				t.Fatalf("version = 0x%x, want 0x%x", binary.LittleEndian.Uint32(gotBytes[4:8]), tt.wantVers)
			}
			if string(gotBytes[8:]) != string(tt.dump[8:]) {
				t.Fatalf("bytes after header were modified")
			}
		})
	}
}

func fakeMinidump(sig, version, streams, rva, streamType uint32) []byte {
	dirSize := int(streams) * minidumpDirectorySize
	buf := make([]byte, minidumpHeaderSize+dirSize)
	binary.LittleEndian.PutUint32(buf[0:4], sig)
	binary.LittleEndian.PutUint32(buf[4:8], version)
	binary.LittleEndian.PutUint32(buf[8:12], streams)
	binary.LittleEndian.PutUint32(buf[12:16], rva)
	if streams > 0 && int(rva)+minidumpDirectorySize <= len(buf) {
		binary.LittleEndian.PutUint32(buf[rva:], streamType)
	}
	return buf
}
