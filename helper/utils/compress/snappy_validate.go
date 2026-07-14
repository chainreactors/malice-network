package compress

import (
	"encoding/binary"

	"github.com/golang/snappy"
)

// ValidateSnappyBlock verifies the complete block structure without allocating
// the declared decoded buffer. decodedLength must come from snappy.DecodedLen.
func ValidateSnappyBlock(compressedData []byte, decodedLength int) error {
	declaredLength, headerLength := binary.Uvarint(compressedData)
	if headerLength <= 0 || decodedLength < 0 || declaredLength != uint64(decodedLength) {
		return snappy.ErrCorrupt
	}

	src := compressedData[headerLength:]
	limit := uint64(decodedLength)
	var produced uint64
	for cursor := 0; cursor < len(src); {
		tag := src[cursor]
		var length, offset uint64

		switch tag & 0x03 {
		case 0: // Literal.
			literalLength := uint64(tag >> 2)
			headerBytes := 1
			if literalLength >= 60 {
				extraBytes := int(literalLength - 59)
				if len(src)-cursor < 1+extraBytes {
					return snappy.ErrCorrupt
				}
				literalLength = 0
				for i := 0; i < extraBytes; i++ {
					literalLength |= uint64(src[cursor+1+i]) << (8 * i)
				}
				headerBytes += extraBytes
			}
			length = literalLength + 1
			cursor += headerBytes
			if length > limit-produced || length > uint64(len(src)-cursor) {
				return snappy.ErrCorrupt
			}
			cursor += int(length)
			produced += length
			continue

		case 1: // Copy with an 11-bit offset.
			if len(src)-cursor < 2 {
				return snappy.ErrCorrupt
			}
			length = 4 + uint64(tag>>2&0x07)
			offset = uint64(tag&0xe0)<<3 | uint64(src[cursor+1])
			cursor += 2

		case 2: // Copy with a 16-bit offset.
			if len(src)-cursor < 3 {
				return snappy.ErrCorrupt
			}
			length = 1 + uint64(tag>>2)
			offset = uint64(binary.LittleEndian.Uint16(src[cursor+1 : cursor+3]))
			cursor += 3

		case 3: // Copy with a 32-bit offset.
			if len(src)-cursor < 5 {
				return snappy.ErrCorrupt
			}
			length = 1 + uint64(tag>>2)
			offset = uint64(binary.LittleEndian.Uint32(src[cursor+1 : cursor+5]))
			cursor += 5
		}

		if offset == 0 || offset > produced || length > limit-produced {
			return snappy.ErrCorrupt
		}
		produced += length
	}

	if produced != limit {
		return snappy.ErrCorrupt
	}
	return nil
}
