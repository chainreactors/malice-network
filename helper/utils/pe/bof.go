package pe

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/chainreactors/malice-network/helper/intl"
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
			return nil, fmt.Errorf("'%v' have not enough arguments", args)
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

func UnpackEmbed(data string) ([]byte, error) {
	// 处理embed://path格式
	embedPath := "embed:" + data
	return intl.ReadEmbedResource(embedPath)
}

func Unpack(data string) ([]byte, error) {
	// 首先尝试直接作为文件路径读取
	content, err := UnPackFile(data)
	if err == nil {
		return content, nil
	}

	// 如果直接读取失败，解析数据类型
	unpacked := strings.SplitN(data, ":", 2)
	if len(unpacked) < 2 {
		return nil, fmt.Errorf("invalid data format: %s", data)
	}

	switch unpacked[0] {
	case "file":
		return UnPackFile(unpacked[1])
	case "embed":
		return UnpackEmbed(unpacked[1])
	case "bin":
		return UnPackBinary(unpacked[1])
	case "url":
		return UnpackURL(unpacked[1])
	default:
		return nil, fmt.Errorf("unknown data type %s", unpacked[0])
	}
}

const (
	CALLBACK_OUTPUT      = 0
	CALLBACK_FILE        = 0x02
	CALLBACK_FILE_WRITE  = 0x08
	CALLBACK_FILE_CLOSE  = 0x09
	CALLBACK_SCREENSHOT  = 0x03
	CALLBACK_ERROR       = 0x0d
	CALLBACK_OUTPUT_OEM  = 0x1e
	CALLBACK_OUTPUT_UTF8 = 0x20
)

type BOFResponse struct {
	CallbackType uint8
	OutputType   uint8
	Length       uint32
	Data         []byte
}

type BOFResponses []*BOFResponse

func (bofResps BOFResponses) String() string {
	return ""
}

func (bofResps BOFResponses) Handler() string {
	var results strings.Builder
	for _, bofResp := range bofResps {
		var result string
		switch bofResp.CallbackType {
		case CALLBACK_OUTPUT, CALLBACK_OUTPUT_OEM, CALLBACK_OUTPUT_UTF8:
			result = string(bofResp.Data)
		case CALLBACK_ERROR:
			result = fmt.Sprintf("Error occurred: %s", string(bofResp.Data))
		case CALLBACK_SCREENSHOT, CALLBACK_FILE, CALLBACK_FILE_WRITE, CALLBACK_FILE_CLOSE:
			continue
		default:
			result = fmt.Sprintf("Unimplemented callback type : %d", bofResp.CallbackType)
		}
		results.WriteString(result + "\n")
	}

	return results.String()
}
