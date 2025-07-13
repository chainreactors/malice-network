package gonut

import (
	"bytes"
	crand "crypto/rand"
	"encoding/binary"
	"fmt"
	"github.com/h2non/filetype"
	"io"
	"math/rand"
	"os"
	"reflect"
	"time"
	"unsafe"
)

func makeRandomStr(table string, length int) string {
	bs := []byte(table)
	var result []byte
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < length; i++ {
		result = append(result, bs[r.Intn(len(bs))])
	}
	return string(result)
}

// GenRandomString Generates a pseudo-random string
// donut/donut.c
// static int gen_random_string(void *output, uint64_t len) { ... }
func GenRandomString(length int) string {
	return makeRandomStr("HMN34P67R9TWCXYF", length) // https://stackoverflow.com/a/27459196
}

// GenRandomBytes Generates pseudo-random bytes.
// donut/donut.c
// static int gen_random(void *buf, uint64_t len) { ... }
func GenRandomBytes(n int) []byte {
	b := make([]byte, n)
	if _, err := io.ReadFull(crand.Reader, b); err != nil {
		panic(err)
	}
	return b
}

func BytesToStruct(buf []byte, v any) error {
	return binary.Read(bytes.NewReader(buf), binary.LittleEndian, v)
}

func StructToBytes(v any) []byte {
	buffer := bytes.NewBuffer(nil)
	if err := binary.Write(buffer, binary.LittleEndian, v); err != nil {
		panic(err)
	}
	return buffer.Bytes()
}

func UnsafeStructToBytes(ptr any) []byte {
	v := reflect.ValueOf(ptr)

	if v.Kind() != reflect.Pointer {
		panic("need a pointer")
	}

	return *(*[]byte)(unsafe.Pointer(
		&reflect.SliceHeader{
			Data: v.Pointer(),
			Len:  int(v.Elem().Type().Size()),
			Cap:  int(v.Elem().Type().Size()),
		},
	))
}

func GetExtension(filepath string) (string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %w", filepath, err)
	}
	defer file.Close()

	buf := make([]byte, 261)
	_, err = file.Read(buf)
	if err != nil {
		return "", fmt.Errorf("read file error: %w", err)
	}

	kind, err := filetype.Match(buf)
	if err != nil {
		return "", fmt.Errorf("unknown file type %s", err)
	}
	return "." + kind.Extension, nil
}
