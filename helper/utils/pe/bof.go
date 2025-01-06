package pe

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type BOFArgsBuffer struct {
	Args []string
}

func (b *BOFArgsBuffer) AddData(d []byte) error {
	b.Args = append(b.Args, PackBinary(string(d)))
	return nil
}

func (b *BOFArgsBuffer) AddShort(d uint16) error {
	data, err := PackShort(d)
	if err != nil {
		return err
	}
	b.Args = append(b.Args, data)
	return nil
}

func (b *BOFArgsBuffer) AddInt(d uint32) error {
	data, err := PackInt(d)
	if err != nil {
		return err
	}
	b.Args = append(b.Args, data)
	return nil
}

func (b *BOFArgsBuffer) AddString(d string) error {
	b.Args = append(b.Args, PackString(d))
	return nil
}

func (b *BOFArgsBuffer) AddWString(d string) error {
	b.Args = append(b.Args, PackWideString(d))
	return nil
}

func (b *BOFArgsBuffer) GetArgs() []string {
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

func PackFile(data string) string {
	return "file:" + data
}

func PackURL(data string) string {
	return "url" + data
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

func UnPackBinary(data string) ([]byte, error) {
	if strings.HasPrefix(data, "bin:") {
		data = data[4:]
	}
	return base64.StdEncoding.DecodeString(data)
}

func UnPackFile(data string) ([]byte, error) {
	if strings.HasPrefix(data, "file:") {
		data = data[5:]
	}

	return os.ReadFile(data)
}

func UnpackURL(data string) ([]byte, error) {
	if strings.HasPrefix(data, "url:") {
		data = data[4:]
	}
	resp, err := http.Get(data)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == 200 {
		return io.ReadAll(resp.Body)
	} else {
		return nil, fmt.Errorf("request error %d", resp.StatusCode)
	}
}

func Unpack(data string) ([]byte, error) {
	unpakced := strings.SplitN(data, ":", 2)
	if len(unpakced) == 1 {
		return UnPackFile(data)
	}
	switch unpakced[0] {
	case "file":
		return UnPackFile(unpakced[1])
	case "bin":
		return UnPackBinary(unpakced[1])
	case "url":
		return UnpackURL(unpakced[1])
	default:
		return nil, fmt.Errorf("Unknown data type %s", unpakced[0])
	}
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
