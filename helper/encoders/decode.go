package encoders

import (
	"github.com/saintfish/chardet"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
	"io/ioutil"
	"strings"
)

func AutoDecode(b []byte) string {
	detector := chardet.NewTextDetector()
	charset, err := detector.DetectBest(b)
	if err != nil {
		return string(b)
	}

	var enc encoding.Encoding
	switch charset.Charset {
	case "UTF-8":
		return string(b)
	case "GB2312", "GBK", "GB-18030":
		enc = simplifiedchinese.GBK // GBK 兼容 GB2312 和 GB18030
	case "Big5":
		enc = traditionalchinese.Big5
	case "UTF-16LE":
		enc = unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)
	case "UTF-16BE":
		enc = unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM)
	case "ISO-8859-1":
		enc = encoding.Nop // ISO-8859-1 兼容 UTF-8
	default:
		return string(b)
	}

	reader := transform.NewReader(strings.NewReader(string(b)), enc.NewDecoder())
	utf8Data, err := ioutil.ReadAll(reader)
	if err != nil {
		return string(b)
	}

	return string(utf8Data)
}
