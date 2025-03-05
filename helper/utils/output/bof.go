package output

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strings"

	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
)

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
	for _, resp := range bofResps {
		switch resp.CallbackType {
		case CALLBACK_OUTPUT, CALLBACK_OUTPUT_OEM, CALLBACK_OUTPUT_UTF8:
			results.WriteString(string(resp.Data))
		case CALLBACK_ERROR:
			results.WriteString(fmt.Sprintf("Error occurred: %s", string(resp.Data)))
		case CALLBACK_SCREENSHOT:
			results.WriteString(fmt.Sprintf("Screenshot data received (size: %d)\n", len(resp.Data)-4))
		case CALLBACK_FILE:
			results.WriteString(fmt.Sprintf("[>] File operation started: %s\n", string(resp.Data[8:])))
		case CALLBACK_FILE_WRITE:
			results.WriteString(fmt.Sprintf("[+] File data received (size: %d) ...\n", len(resp.Data)-4))
		case CALLBACK_FILE_CLOSE:
			results.WriteString("[✓] File operation completed\n")
		default:
			results.WriteString(fmt.Sprintf("Callback type %d: %s\n", resp.CallbackType, string(resp.Data)))
		}
	}
	return results.String()
}

func ParseBOFResponse(ctx *clientpb.TaskContext) (interface{}, error) {
	reader := bytes.NewReader(ctx.Spite.GetBinaryResponse().GetData())
	var bofResps BOFResponses

	for {
		bofResp := &BOFResponse{}

		err := binary.Read(reader, binary.LittleEndian, &bofResp.OutputType)
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("failed to read OutputType: %v", err)
		}

		err = binary.Read(reader, binary.LittleEndian, &bofResp.CallbackType)
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("failed to read CallbackType: %v", err)
		}

		err = binary.Read(reader, binary.LittleEndian, &bofResp.Length)
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("failed to read Length: %v", err)
		}

		strData := make([]byte, bofResp.Length)
		_, err = io.ReadFull(reader, strData)
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("failed to read StrData: %v", err)
		}

		bofResp.Data = strData

		bofResps = append(bofResps, bofResp)
	}

	return bofResps, nil
}
