package helper

import (
	"bytes"
	"encoding/binary"
	"github.com/chainreactors/malice-network/helper/consts"
)

func ShortSessionID(id string) string {
	return id[:8]
}

const (
	IMAGE_FILE_DLL              = 0x2000
	IMAGE_FILE_EXECUTABLE_IMAGE = 0x0002
)

func CheckPEType(content []byte) int {
	if len(content) < 64 {
		return consts.UnknownFile
	}

	// Check the DOS header magic number
	if content[0] != 0x4D || content[1] != 0x5A { // "MZ"
		return consts.UnknownFile
	}

	// Read the offset to PE header
	e_lfanew := int32(binary.LittleEndian.Uint32(content[60:64]))

	// Check if the file is large enough to contain the PE header
	if len(content) < int(e_lfanew)+24 {
		return consts.UnknownFile
	}

	// Check the PE signature
	peSignature := content[e_lfanew : e_lfanew+4]
	if !bytes.Equal(peSignature, []byte("PE\x00\x00")) {
		return consts.UnknownFile
	}

	// Read the Characteristics field
	characteristics := binary.LittleEndian.Uint16(content[e_lfanew+22 : e_lfanew+24])

	switch {
	case characteristics&IMAGE_FILE_DLL != 0:
		return consts.DLLFile
	case characteristics&IMAGE_FILE_EXECUTABLE_IMAGE != 0:
		return consts.EXEFile
	default:
		return consts.UnknownFile
	}
}
