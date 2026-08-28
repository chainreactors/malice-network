package output

import (
	"encoding/binary"
	"io"
	"os"
)

const (
	minidumpSignature     = 0x504D444D // 'MDMP'
	minidumpVersion       = 0xA793
	minidumpHeaderSize    = 32
	minidumpDirectorySize = 12
	maxMinidumpStreams    = 64
)

// RestoreInvalidMinidumpSignature rewrites Signature and Version when the file
// looks like a nanodump minidump with a deliberately invalid header.
// Already-valid dumps (--valid, WerFault shtinkering / silent-process-exit) are
// left unchanged. Returns true if the header was patched.
func RestoreInvalidMinidumpSignature(path string) (bool, error) {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return false, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return false, err
	}
	if info.Size() < minidumpHeaderSize {
		return false, nil
	}

	header := make([]byte, minidumpHeaderSize)
	if _, err := io.ReadFull(f, header); err != nil {
		return false, err
	}
	if binary.LittleEndian.Uint32(header[0:4]) == minidumpSignature {
		return false, nil
	}
	if !looksLikeInvalidMinidump(f, header, info.Size()) {
		return false, nil
	}

	binary.LittleEndian.PutUint32(header[0:4], minidumpSignature)
	binary.LittleEndian.PutUint32(header[4:8], minidumpVersion)
	if _, err := f.WriteAt(header[0:8], 0); err != nil {
		return false, err
	}
	return true, nil
}

func looksLikeInvalidMinidump(f *os.File, header []byte, size int64) bool {
	if len(header) < minidumpHeaderSize {
		return false
	}
	numStreams := binary.LittleEndian.Uint32(header[8:12])
	rva := binary.LittleEndian.Uint32(header[12:16])
	if numStreams == 0 || numStreams > maxMinidumpStreams {
		return false
	}
	dirEnd := uint64(rva) + uint64(numStreams)*minidumpDirectorySize
	if uint64(rva) < minidumpHeaderSize || dirEnd > uint64(size) {
		return false
	}

	entry := make([]byte, minidumpDirectorySize)
	if _, err := f.ReadAt(entry, int64(rva)); err != nil {
		return false
	}
	return isPlausibleMinidumpStreamType(binary.LittleEndian.Uint32(entry[0:4]))
}

func isPlausibleMinidumpStreamType(streamType uint32) bool {
	switch {
	case streamType <= 0x16:
		return true
	case streamType >= 0x8000 && streamType <= 0x8006:
		return true
	default:
		return false
	}
}
