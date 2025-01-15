package pe

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
	content, err := UnPackFile(data)
	if err == nil {
		return content, nil
	}
	unpakced := strings.SplitN(data, ":", 2)
	result, err := UnPackFile(data)
	if err == nil {
		return result, err
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
	var results strings.Builder
	for _, bofResp := range bofResps {
		var result string
		switch bofResp.CallbackType {
		case CALLBACK_OUTPUT, CALLBACK_OUTPUT_OEM, CALLBACK_OUTPUT_UTF8:
			result = string(bofResp.Data)
		case CALLBACK_ERROR:
			result = fmt.Sprintf("Error occurred: %s", string(bofResp.Data))
		}
		results.WriteString(result + "\n")
	}

	return results.String()
}

func (bofResps BOFResponses) Handler(sess *clientpb.Session) string {
	var err error
	var results strings.Builder

	fileMap := make(map[string]*os.File)

	for _, bofResp := range bofResps {
		var result string
		switch bofResp.CallbackType {
		case CALLBACK_OUTPUT, CALLBACK_OUTPUT_OEM, CALLBACK_OUTPUT_UTF8:
			result = string(bofResp.Data)
		case CALLBACK_ERROR:
			result = fmt.Sprintf("Error occurred: %s", string(bofResp.Data))
		case CALLBACK_SCREENSHOT:
			fileName := "screenshot.jpg"
			result = func() string {
				if bofResp.Length-4 <= 0 {
					return fmt.Sprintf("Null screenshot data")
				}
				screenfile, err := assets.GenerateTempFile(sess.SessionId, fileName)
				if err != nil {
					return fmt.Sprintf("Failed to create screenshot file")
				}
				defer func() {
					err := screenfile.Close()
					if err != nil {
						return
					}
				}()
				data := bofResp.Data[4:]
				if _, err := screenfile.Write(data); err != nil {
					return fmt.Sprintf("Failed to write screenshot data: %s", err.Error())
				}

				return fmt.Sprintf("Screenshot saved to %s", screenfile.Name())
			}()
		case CALLBACK_FILE:
			result = func() string {
				fileId := fmt.Sprintf("%d", binary.LittleEndian.Uint32(bofResp.Data[:4]))
				fileName := string(bofResp.Data[8:])
				file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					return fmt.Sprintf("Could not open file '%s' (ID: %s): %s", filepath.Base(file.Name()), fileId, err)
				}
				fileMap[fileId] = file
				return fmt.Sprintf("File '%s' (ID: %s) opened successfully", filepath.Base(file.Name()), fileId)
			}()
		case CALLBACK_FILE_WRITE:
			result = func() string {
				fileId := fmt.Sprintf("%d", binary.LittleEndian.Uint32(bofResp.Data[:4]))
				file := fileMap[fileId]
				if file == nil {
					return fmt.Sprintf("No open file to write to (ID: %s)", fileId)
				}
				_, err = file.Write(bofResp.Data[4:])
				if err != nil {
					return fmt.Sprintf("Error writing to file (ID: %s): %s", fileId, err)
				}
				return fmt.Sprintf("Data(Size: %d) written to file (ID: %s) successfully", bofResp.Length-4, fileId)
			}()
		case CALLBACK_FILE_CLOSE:
			result = func() string {
				fileId := fmt.Sprintf("%d", binary.LittleEndian.Uint32(bofResp.Data[:4]))
				file := fileMap[fileId]
				if file == nil {
					return fmt.Sprintf("No open file to close (ID: %s)", fileId)
				}
				err = file.Close()
				if err != nil {
					return fmt.Sprintf("Error closing file (ID: %s): %s", fileId, err)
				}
				delete(fileMap, fileId)
				return fmt.Sprintf("File (ID: %s) closed successfully", fileId)
			}()
		default:
			result = func() string {
				return fmt.Sprintf("Unimplemented callback type : %d", bofResp.CallbackType)
			}()
		}
		results.WriteString(result + "\n")
	}
	// Close any remaining open files
	for fileId, file := range fileMap {
		if file != nil {
			err := file.Close()
			if err != nil {
				results.WriteString(fmt.Sprintf("Error closing file (ID: %s): %s\n", fileId, err))
			} else {
				results.WriteString(fmt.Sprintf("File (ID: %s) closed automatically due to end of processing\n", fileId))
			}
			delete(fileMap, fileId)
		}
	}
	return results.String()
}
