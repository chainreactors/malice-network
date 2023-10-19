package hash

import (
	"crypto/md5"
	"encoding/hex"
)

func UnHexlify(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

func Hexlify(b []byte) string {
	return hex.EncodeToString(b)
}

func Md5Hash(raw []byte) string {
	m := md5.Sum(raw)
	return Hexlify(m[:])
}
