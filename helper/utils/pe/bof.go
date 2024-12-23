package pe

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/text/encoding/unicode"
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

func PackArg(format byte, arg string) (string, error) {
	switch format {
	case 'b':
		return PackBinary(arg), nil
	case 'i':
		return PackIntString(arg)
	case 's':
		return PackShortString(arg)
	case 'z':
		var packedData string
		// Handler for packing empty strings
		if len(arg) == 0 {
			packedData = PackString("")
		} else {
			packedData = PackString(arg)
		}
		return packedData, nil
	case 'Z':
		var packedData string
		if len(arg) == 0 {
			packedData = PackWideString("")
		} else {
			packedData = PackWideString(arg)
		}
		return packedData, nil
	default:
		return "", fmt.Errorf("Data must be prefixed with 'b', 'i', 's','z', or 'Z'\n")
	}
}
func PackArgs(data []string) ([]string, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var args []string
	var err error
	for _, arg := range data {
		if len(arg) < 1 {
			return nil, fmt.Errorf("'%' have not enough arguments", args)
		}
		format := arg[0]
		packedArg := ""
		if len(arg) > 1 {
			packedArg = arg[1:]
		}
		arg, err = PackArg(format, packedArg)
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
	}
	return args, nil
}

func PackBinary(data string) string {
	return fmt.Sprintf(`bin:%s`, base64.StdEncoding.EncodeToString([]byte(data)))
}

func PackInt(i uint32) (string, error) {
	return fmt.Sprintf(`int:%d`, i), nil
}

func PackIntString(s string) (string, error) {
	i, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return "", err
	}
	return PackInt(uint32(i))
}

func PackShort(i uint16) (string, error) {
	return fmt.Sprintf(`short:%d`, i), nil
}

func PackShortString(s string) (string, error) {
	i, err := strconv.ParseUint(s, 10, 16)
	if err != nil {
		return "", err
	}
	return PackShort(uint16(i))
}

func PackString(s string) string {
	return fmt.Sprintf(`str:%s`, s)
}

func PackWideString(s string) string {
	return fmt.Sprintf(`wstr:%s`, s)
}

const (
	CALLBACK_OUTPUT      = 0
	CALLBACK_SCREENSHOT  = 3
	CALLBACK_ERROR       = 13
	CALLBACK_OUTPUT_OEM  = 30
	CALLBACK_OUTPUT_UTF8 = 32
)

type BOFResponse struct {
	CallbackType uint8
	OutputType   uint8
	Length       uint32
	Data         []byte
}

func (bof *BOFResponse) String() string {
	switch bof.CallbackType {
	case CALLBACK_OUTPUT, CALLBACK_OUTPUT_OEM, CALLBACK_OUTPUT_UTF8:
		return string(bof.Data)
	case CALLBACK_ERROR:
		return fmt.Sprintf("Error: %s", string(bof.Data))
	case CALLBACK_SCREENSHOT:
		return "screenshot"
	default:
		return fmt.Sprintf("\nUnimplemented callback type ID: %d.\nData: %s", bof.CallbackType, bof.Data)
	}
}

type BOFResponses []*BOFResponse

func (bofs BOFResponses) String() string {
	var s strings.Builder
	for _, r := range bofs {
		s.WriteString(r.String() + "\n")
	}
	return s.String()
}
