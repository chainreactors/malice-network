package gonut

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math"
)

type FormatTemplate struct {
	Data []byte
}

func (f *FormatTemplate) ToBinary() []byte {
	return f.Data
}

func (f *FormatTemplate) ToBase64() []byte {
	return []byte(base64.StdEncoding.EncodeToString(f.Data))
}

func (f *FormatTemplate) ToHex() []byte {
	return []byte(hex.EncodeToString(f.Data))
}

func (f *FormatTemplate) ToRubyC() []byte {
	buffer := bytes.NewBufferString("unsigned char buf[] = ")
	rows := Convert1d2d(f.Data, 16)
	for _, row := range rows {
		buffer.WriteString("\n\"")
		for _, c := range row {
			buffer.WriteString(fmt.Sprintf("\\x%02x", c))
		}
		buffer.WriteString("\"")
	}

	buffer.WriteString(";\n")
	return buffer.Bytes()
}

func (f *FormatTemplate) ToPython() []byte {
	buffer := bytes.NewBufferString("buf =  b\"\"")
	rows := Convert1d2d(f.Data, 12)
	for _, row := range rows {
		buffer.WriteString("\nbuf += b\"")
		for _, c := range row {
			buffer.WriteString(fmt.Sprintf("\\x%02x", c))
		}
		buffer.WriteString("\"")
	}

	buffer.WriteString("\n")
	return buffer.Bytes()
}

func (f *FormatTemplate) ToPowerShell() []byte {
	buffer := bytes.NewBufferString("[Byte[]] $buf = ")
	for _, c := range f.Data {
		buffer.WriteString(fmt.Sprintf("0x%02x,", c))
	}

	buffer.Bytes()[buffer.Len()-1] = '\n'
	return buffer.Bytes()
}

func (f *FormatTemplate) ToCSharp() []byte {
	buffer := bytes.NewBufferString(fmt.Sprintf("byte[] buf = new byte[%d] {", len(f.Data)))
	rows := Convert1d2d(f.Data, 12)
	for _, row := range rows {
		buffer.WriteString("\n")
		for _, c := range row {
			buffer.WriteString(fmt.Sprintf("0x%02x,", c))
		}
	}

	buffer.Bytes()[buffer.Len()-1] = '}'
	buffer.WriteString(";\n")
	return buffer.Bytes()
}

func (f *FormatTemplate) ToGolang() []byte {
	buffer := bytes.NewBufferString(fmt.Sprintf("buf :=  []byte{"))
	rows := Convert1d2d(f.Data, 12)
	for _, row := range rows {
		buffer.WriteString("\n")
		for _, c := range row {
			buffer.WriteString(fmt.Sprintf("0x%02x,", c))
		}
	}

	buffer.WriteString("\n}\n")
	return buffer.Bytes()
}

func (f *FormatTemplate) ToRust() []byte {
	buffer := bytes.NewBufferString(fmt.Sprintf("let buf: [u8; %d] = [", len(f.Data)))
	rows := Convert1d2d(f.Data, 12)
	for _, row := range rows {
		buffer.WriteString("\n")
		for _, c := range row {
			buffer.WriteString(fmt.Sprintf("0x%02x,", c))
		}
	}

	buffer.WriteString("\n];\n")
	return buffer.Bytes()
}

func (f *FormatTemplate) ToUUID() []byte {
	buffer := bytes.NewBufferString("")
	rows := Convert1d2d(f.Data, 16)
	for _, row := range rows {
		if len(row) < 16 {
			row = append(row, bytes.Repeat([]byte{byte(0x90)}, 16-len(row))...)
		}
		buffer.WriteString(fmt.Sprintf("%02x%02x%02x%02x-", row[3], row[2], row[1], row[0]))
		buffer.WriteString(fmt.Sprintf("%02x%02x-", row[5], row[4]))
		buffer.WriteString(fmt.Sprintf("%02x%02x-", row[7], row[6]))
		buffer.WriteString(fmt.Sprintf("%02x%02x-", row[8], row[9]))
		buffer.WriteString(fmt.Sprintf("%08x", row[10:16]))
		buffer.WriteString("\n")
	}

	return buffer.Bytes()
}

func Convert1d2d(data []byte, n int) [][]byte {
	length := len(data)
	count := int(math.Ceil(float64(length) / float64(n)))
	rows := make([][]byte, count)
	for i := 0; i < count; i++ {
		start := i * n
		end := start + n
		if end > length {
			end = length
		}
		rows[i] = data[start:end]
	}
	return rows
}

func NewFormatTemplate(data []byte) *FormatTemplate {
	return &FormatTemplate{Data: data}
}
