package encoders

import (
	"encoding/binary"
)

func Uint32ToBytes(n uint32) []byte {
	bytes := make([]byte, 4)
	// 使用 LittleEndian 将 uint32 转换为字节
	binary.LittleEndian.PutUint32(bytes, n)
	return bytes
}

func BytesToUint32(b []byte) uint32 {
	return binary.LittleEndian.Uint32(b)
}
