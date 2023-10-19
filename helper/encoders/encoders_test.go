package encoders

import (
	"crypto/rand"
	insecureRand "math/rand"
)

func randomData() []byte {
	buf := make([]byte, insecureRand.Intn(1024))
	rand.Read(buf)
	return buf
}
