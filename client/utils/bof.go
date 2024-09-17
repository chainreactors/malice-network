package utils

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/gookit/goutil/encodes"
	"golang.org/x/text/encoding/unicode"
	"strconv"
)

type BOFArgsBuffer struct {
	Buffer *bytes.Buffer
}

func (b *BOFArgsBuffer) AddData(d []byte) error {
	dataLen := uint32(len(d))
	err := binary.Write(b.Buffer, binary.LittleEndian, &dataLen)
	if err != nil {
		return err
	}
	return binary.Write(b.Buffer, binary.LittleEndian, &d)
}

func (b *BOFArgsBuffer) AddShort(d uint16) error {
	return binary.Write(b.Buffer, binary.LittleEndian, &d)
}

func (b *BOFArgsBuffer) AddInt(d uint32) error {
	return binary.Write(b.Buffer, binary.LittleEndian, &d)
}

func (b *BOFArgsBuffer) AddString(d string) error {
	stringLen := uint32(len(d)) + 1
	err := binary.Write(b.Buffer, binary.LittleEndian, &stringLen)
	if err != nil {
		return err
	}
	dBytes := append([]byte(d), 0x00)
	return binary.Write(b.Buffer, binary.LittleEndian, dBytes)
}

func (b *BOFArgsBuffer) AddWString(d string) error {
	encoder := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewEncoder()
	strBytes := append([]byte(d), 0x00)
	utf16Data, err := encoder.Bytes(strBytes)
	if err != nil {
		return err
	}
	stringLen := uint32(len(utf16Data))
	err = binary.Write(b.Buffer, binary.LittleEndian, &stringLen)
	if err != nil {
		return err
	}
	return binary.Write(b.Buffer, binary.LittleEndian, utf16Data)
}

func (b *BOFArgsBuffer) GetBuffer() ([]byte, error) {
	outBuffer := new(bytes.Buffer)
	err := binary.Write(outBuffer, binary.LittleEndian, uint32(b.Buffer.Len()))
	if err != nil {
		return nil, err
	}
	err = binary.Write(outBuffer, binary.LittleEndian, b.Buffer.Bytes())
	if err != nil {
		return nil, err
	}
	return outBuffer.Bytes(), nil
}

type IoMBOFArgsBuffer struct {
	Args []string
}

func (b *IoMBOFArgsBuffer) AddData(d []byte) error {
	b.Args = append(b.Args, PackBinary(string(d)))
	return nil
}

func (b *IoMBOFArgsBuffer) AddShort(d uint16) error {
	data, err := PackShort(d)
	if err != nil {
		return err
	}
	b.Args = append(b.Args, data)
	return nil
}

func (b *IoMBOFArgsBuffer) AddInt(d uint32) error {
	data, err := PackInt(d)
	if err != nil {
		return err
	}
	b.Args = append(b.Args, data)
	return nil
}

func (b *IoMBOFArgsBuffer) AddString(d string) error {
	b.Args = append(b.Args, PackString(d))
	return nil
}

func (b *IoMBOFArgsBuffer) AddWString(d string) error {
	b.Args = append(b.Args, PackWideString(d))
	return nil
}

func (b *IoMBOFArgsBuffer) GetArgs() []string {
	return b.Args
}

func PackArgs(data []string) ([]string, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var args []string
	for _, arg := range data {
		switch arg[0] {
		case 'b':
			args = append(args, PackBinary(arg[1:]))
		case 'i':
			data, err := PackIntString(arg[1:])
			if err != nil {
				return nil, fmt.Errorf("Int packing error:\n INPUT: '%s'\n ERROR:%s\n", arg[1:], err)
			}
			args = append(args, data)
		case 's':
			data, err := PackShortString(arg[1:])
			if err != nil {
				return nil, fmt.Errorf("Short packing error:\n INPUT: '%s'\n ERROR:%s\n", arg[1:], err)
			}
			args = append(args, data)
		case 'z':
			var packedData string
			// Handler for packing empty strings
			if len(arg) < 2 {
				packedData = PackString("")
			} else {
				packedData = PackString(arg[1:])
			}
			args = append(args, packedData)
		case 'Z':
			var packedData string
			if len(arg) < 2 {
				packedData = PackWideString("")
			} else {
				packedData = PackWideString(arg[1:])
			}
			args = append(args, packedData)
		default:
			return nil, fmt.Errorf("Data must be prefixed with 'b', 'i', 's','z', or 'Z'\n")
		}
	}
	return args, nil
}

func PackBinary(data string) string {
	return fmt.Sprintf("bin:" + encodes.B64Encode(data))
}

func PackInt(i uint32) (string, error) {
	return fmt.Sprintf("int:%d", i), nil
}

func PackIntString(s string) (string, error) {
	i, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return "", err
	}
	return PackInt(uint32(i))
}

func PackShort(i uint16) (string, error) {
	return fmt.Sprintf("short:%d", i), nil
}

func PackShortString(s string) (string, error) {
	i, err := strconv.ParseUint(s, 10, 16)
	if err != nil {
		return "", err
	}
	return PackShort(uint16(i))
}

func PackString(s string) string {
	return fmt.Sprintf(`str:"%s"`, s)
}

func PackWideString(s string) string {
	return fmt.Sprintf(`wstr:"%s"`, s)
}
