package encoders

import (
	"bytes"
	"github.com/saintfish/chardet"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
	"unicode/utf8"
)

func AutoDecode(b []byte) string {
	if len(b) == 0 {
		return ""
	}

	// 1. 检查 UTF-8 BOM
	if bytes.HasPrefix(b, []byte{0xEF, 0xBB, 0xBF}) {
		return string(b[3:])
	}

	// 2. 尝试 UTF-8 严格验证
	if utf8.Valid(b) {
		return string(b)
	}

	// 3. 小数据块（< 1KB）优先尝试 GBK（Windows 中文常见）
	if len(b) < 1024 {
		if decoded := tryDecode(simplifiedchinese.GBK.NewDecoder(), b); decoded != "" {
			return decoded
		}
		// GBK 失败，返回原始字符串
		return string(b)
	}

	// 4. 大数据块使用 chardet 检测
	detector := chardet.NewTextDetector()
	charset, err := detector.DetectBest(b)
	if err != nil {
		// 检测失败，尝试 GBK
		if decoded := tryDecode(simplifiedchinese.GBK.NewDecoder(), b); decoded != "" {
			return decoded
		}
		return string(b)
	}

	var enc encoding.Encoding
	switch charset.Charset {
	case "UTF-8":
		return string(b)
	case "GB2312", "GBK", "GB-18030":
		enc = simplifiedchinese.GBK
	case "Big5":
		// Big5 容易误判，先尝试 GBK
		if decoded := tryDecode(simplifiedchinese.GBK.NewDecoder(), b); decoded != "" {
			return decoded
		}
		enc = traditionalchinese.Big5
	case "UTF-16LE":
		enc = unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)
	case "UTF-16BE":
		enc = unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM)
	case "ISO-8859-1":
		// ISO-8859-1 通常是误判，尝试 GBK
		if decoded := tryDecode(simplifiedchinese.GBK.NewDecoder(), b); decoded != "" {
			return decoded
		}
		return string(b)
	default:
		// 未知编码，尝试 GBK
		if decoded := tryDecode(simplifiedchinese.GBK.NewDecoder(), b); decoded != "" {
			return decoded
		}
		return string(b)
	}

	if decoded := tryDecode(enc.NewDecoder(), b); decoded != "" {
		return decoded
	}
	return string(b)
}

func tryDecode(decoder *encoding.Decoder, b []byte) string {
	utf8Data, _, err := transform.Bytes(decoder, b)
	if err != nil {
		return ""
	}
	return string(utf8Data)
}
