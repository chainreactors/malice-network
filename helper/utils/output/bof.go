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
	CallbackOutput      = 0
	CallbackFile        = 0x02
	CallbackFileWrite   = 0x08
	CallbackFileClose   = 0x09
	CallbackScreenshot  = 0x03
	CallbackError       = 0x0d
	CallbackOutputOem   = 0x1e
	CallbackOutputUtf8  = 0x20
	CallbackSystemError = 0x4d
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
		case CallbackOutput, CallbackOutputOem, CallbackOutputUtf8:
			results.WriteString(string(resp.Data))
		case CallbackError:
			results.WriteString(fmt.Sprintf("[!] Error occurred: %s", string(resp.Data)))
		case CallbackScreenshot:
			results.WriteString(fmt.Sprintf("Screenshot data received (size: %d)\n", len(resp.Data)-4))
		case CallbackFile:
			results.WriteString(fmt.Sprintf("[>] File operation started: %s\n", string(resp.Data[8:])))
		case CallbackFileWrite:
			results.WriteString(fmt.Sprintf("[+] File data received (size: %d) ...\n", len(resp.Data)-4))
		case CallbackFileClose:
			results.WriteString("[✓] File operation completed\n")
		case CallbackSystemError:
			results.WriteString(fmt.Sprintf("[!] System error occurred: %s\n", string(resp.Data)))
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
			return nil, fmt.Errorf("failed to read StrData: err = %v", err)
		}

		bofResp.Data = strData

		bofResps = append(bofResps, bofResp)
	}

	return bofResps, nil
}
