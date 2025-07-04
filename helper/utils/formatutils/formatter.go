package formatutils

import (
	"fmt"
	"strings"
)

type FormatResult struct {
	Data      []byte
	Extension string
}

type FormatInfo struct {
	Extension string
	Desc      string
	Converter func([]byte) []byte
}

type Formatter struct {
	supportedFormats map[string]FormatInfo
}

// NewFormatter creates a new formatter with all supported formats
func NewFormatter() *Formatter {
	formatter := &Formatter{
		supportedFormats: make(map[string]FormatInfo),
	}

	// Register all supported formats with descriptions
	formatter.register("raw", FormatInfo{".bin", "raw binary format", func(data []byte) []byte { return data }})
	//formatter.register("bin", FormatInfo{".bin", "binary format", func(data []byte) []byte { return data }})
	//formatter.register("binary", FormatInfo{".bin", "binary format", func(data []byte) []byte { return data }})
	//formatter.register("native", FormatInfo{".exe", "native executable format", func(data []byte) []byte { return data }})
	//formatter.register("exe", FormatInfo{".exe", "Windows executable format", func(data []byte) []byte { return data }})
	//formatter.register("elf", FormatInfo{".elf", "Linux ELF executable format", func(data []byte) []byte { return data }})

	formatter.register("c", FormatInfo{".c", "C language format", toC})
	formatter.register("csharp", FormatInfo{".cs", "C# language format", toCSharp})
	formatter.register("java", FormatInfo{".java", "Java language format", toJava})
	//formatter.register("go", FormatInfo{".go", "Go language format", toGo})
	formatter.register("golang", FormatInfo{".go", "Go language format", toGo})

	formatter.register("python", FormatInfo{".py", "Python language format", toPython})
	//formatter.register("py", FormatInfo{".py", "Python language format", toPython})
	formatter.register("perl", FormatInfo{".pl", "Perl language format", toPerl})
	//formatter.register("pl", FormatInfo{".pl", "Perl language format", toPerl})
	formatter.register("ruby", FormatInfo{".rb", "Ruby language format", toRuby})
	//formatter.register("rb", FormatInfo{".rb", "Ruby language format", toRuby})

	formatter.register("bash", FormatInfo{".sh", "Bash script format", toBash})
	//formatter.register("sh", FormatInfo{".sh", "shell script format", toBash})
	formatter.register("powershell", FormatInfo{".ps1", "PowerShell script format", toPowerShell})
	//formatter.register("ps1", FormatInfo{".ps1", "PowerShell script format", toPowerShell})

	formatter.register("hex", FormatInfo{".txt", "hexadecimal format", toHex})
	formatter.register("num", FormatInfo{".txt", "numeric format", toNum})
	//formatter.register("dw", FormatInfo{".txt", "dword format", toDword})
	formatter.register("dword", FormatInfo{".txt", "dword format", toDword})

	formatter.register("js_be", FormatInfo{".js", "JavaScript big-endian format", func(data []byte) []byte { return toJavaScript(data, true) }})
	formatter.register("js_le", FormatInfo{".js", "JavaScript little-endian format", func(data []byte) []byte { return toJavaScript(data, false) }})

	formatter.register("vbscript", FormatInfo{".vbs", "VBScript format", toVBScript})
	formatter.register("vbapplication", FormatInfo{".vba", "VBA application format", toVBApplication})

	return formatter
}

// register adds a format to the supported formats map
func (f *Formatter) register(name string, info FormatInfo) {
	f.supportedFormats[strings.ToLower(name)] = info
}

// GetSupportedFormats returns a list of all supported format names
func (f *Formatter) GetSupportedFormats() []string {
	formats := make([]string, 0, len(f.supportedFormats))
	for format := range f.supportedFormats {
		formats = append(formats, format)
	}
	return formats
}

// IsSupported checks if a format is supported
func (f *Formatter) IsSupported(format string) bool {
	_, exists := f.supportedFormats[strings.ToLower(format)]
	return exists
}

// GetFormatDescription returns the description for a specific format
func (f *Formatter) GetFormatDescription(format string) string {
	if info, exists := f.supportedFormats[strings.ToLower(format)]; exists {
		return info.Desc
	}
	return format + " format"
}

// GetFormatsWithDescriptions returns a map of format names to descriptions
func (f *Formatter) GetFormatsWithDescriptions() map[string]string {
	result := make(map[string]string)
	for format, info := range f.supportedFormats {
		result[format] = info.Desc
	}
	return result
}

// GetFormatExtension returns the file extension for a specific format
func (f *Formatter) GetFormatExtension(format string) string {
	if info, exists := f.supportedFormats[strings.ToLower(format)]; exists {
		return info.Extension
	}
	return ".txt"
}

// Convert converts raw shellcode bytes to the specified format
func (f *Formatter) Convert(data []byte, format string) (*FormatResult, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data")
	}

	formatInfo, exists := f.supportedFormats[strings.ToLower(format)]
	if !exists {
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	convertedData := formatInfo.Converter(data)
	return &FormatResult{
		Data:      convertedData,
		Extension: formatInfo.Extension,
	}, nil
}

// Helper functions for format conversion
func toC(data []byte) []byte {
	var result strings.Builder
	result.WriteString("unsigned char buf[] = \n\"")

	for i, b := range data {
		if i > 0 && i%16 == 0 {
			result.WriteString("\"\n\"")
		}
		result.WriteString(fmt.Sprintf("\\x%02x", b))
	}
	result.WriteString("\";")
	return []byte(result.String())
}

func toCSharp(data []byte) []byte {
	var result strings.Builder
	result.WriteString(fmt.Sprintf("byte[] buf = new byte[%d] {", len(data)))

	for i, b := range data {
		if i > 0 {
			result.WriteString(",")
		}
		if i%16 == 0 {
			result.WriteString("\n    ")
		} else {
			result.WriteString(" ")
		}
		result.WriteString(fmt.Sprintf("0x%02x", b))
	}
	result.WriteString("\n};")
	return []byte(result.String())
}

func toJava(data []byte) []byte {
	var result strings.Builder
	result.WriteString("byte buf[] = new byte[]\n{\n")

	for i, b := range data {
		if i > 0 {
			result.WriteString(",")
		}
		if i%8 == 0 {
			result.WriteString("\n    ")
		} else {
			result.WriteString(" ")
		}
		result.WriteString(fmt.Sprintf("(byte) 0x%02x", b))
	}
	result.WriteString("\n};\n")
	return []byte(result.String())
}

func toGo(data []byte) []byte {
	var result strings.Builder
	result.WriteString("buf := []byte{")

	for i, b := range data {
		if i > 0 {
			result.WriteString(",")
		}
		if i%16 == 0 {
			result.WriteString("\n    ")
		} else {
			result.WriteString(" ")
		}
		result.WriteString(fmt.Sprintf("0x%02x", b))
	}
	result.WriteString("\n}")
	return []byte(result.String())
}

func toPython(data []byte) []byte {
	var result strings.Builder
	result.WriteString("buf = b\"")

	for i, b := range data {
		if i > 0 && i%16 == 0 {
			result.WriteString("\"\n")
			result.WriteString("buf += b\"")
		}
		result.WriteString(fmt.Sprintf("\\x%02x", b))
	}
	result.WriteString("\"")
	return []byte(result.String())
}

func toPerl(data []byte) []byte {
	var result strings.Builder
	result.WriteString("my $buf = \n\"")

	for i, b := range data {
		if i > 0 && i%16 == 0 {
			result.WriteString("\" .\n\"")
		}
		result.WriteString(fmt.Sprintf("\\x%02x", b))
	}
	result.WriteString("\";")
	return []byte(result.String())
}

func toRuby(data []byte) []byte {
	var result strings.Builder
	result.WriteString("buf = \n\"")

	for i, b := range data {
		if i > 0 && i%16 == 0 {
			result.WriteString("\" +\n\"")
		}
		result.WriteString(fmt.Sprintf("\\x%02x", b))
	}
	result.WriteString("\"")
	return []byte(result.String())
}

func toBash(data []byte) []byte {
	var result strings.Builder
	result.WriteString("export buf=\\\n$'")

	for i, b := range data {
		if i > 0 && i%16 == 0 {
			result.WriteString("'\\\n$'")
		}
		result.WriteString(fmt.Sprintf("\\x%02x", b))
	}
	result.WriteString("'")
	return []byte(result.String())
}

func toPowerShell(data []byte) []byte {
	var result strings.Builder
	result.WriteString("$buf = @(")

	for i, b := range data {
		if i > 0 {
			result.WriteString(",")
		}
		if i%16 == 0 {
			result.WriteString("\n    ")
		} else {
			result.WriteString(" ")
		}
		result.WriteString(fmt.Sprintf("0x%02x", b))
	}
	result.WriteString("\n)")
	return []byte(result.String())
}

func toHex(data []byte) []byte {
	var result strings.Builder
	for i, b := range data {
		if i > 0 && i%16 == 0 {
			result.WriteString("\n")
		}
		result.WriteString(fmt.Sprintf("%02x", b))
	}
	return []byte(result.String())
}

func toNum(data []byte) []byte {
	var result strings.Builder
	for i, b := range data {
		if i > 0 {
			result.WriteString(",")
		}
		if i%16 == 0 {
			result.WriteString("\n")
		}
		result.WriteString(fmt.Sprintf("%d", b))
	}
	return []byte(result.String())
}

func toDword(data []byte) []byte {
	var result strings.Builder
	// Pad data to multiple of 4 bytes
	padded := make([]byte, len(data))
	copy(padded, data)
	for len(padded)%4 != 0 {
		padded = append(padded, 0)
	}

	for i := 0; i < len(padded); i += 4 {
		if i > 0 {
			result.WriteString(",")
		}
		if i%16 == 0 {
			result.WriteString("\n")
		}
		dword := uint32(padded[i]) | uint32(padded[i+1])<<8 | uint32(padded[i+2])<<16 | uint32(padded[i+3])<<24
		result.WriteString(fmt.Sprintf("0x%08x", dword))
	}
	return []byte(result.String())
}

func toJavaScript(data []byte, bigEndian bool) []byte {
	var result strings.Builder
	result.WriteString("var buf = [")

	for i, b := range data {
		if i > 0 {
			result.WriteString(",")
		}
		if i%16 == 0 {
			result.WriteString("\n    ")
		} else {
			result.WriteString(" ")
		}
		result.WriteString(fmt.Sprintf("0x%02x", b))
	}
	result.WriteString("\n];")
	return []byte(result.String())
}

func toVBScript(data []byte) []byte {
	if len(data) == 0 {
		return []byte("buf=\"\"")
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("buf=Chr(%d)", data[0]))

	for i := 1; i < len(data); i++ {
		if i%100 == 0 {
			result.WriteString("\r\nbuf=buf")
		}
		result.WriteString(fmt.Sprintf("&Chr(%d)", data[i]))
	}
	return []byte(result.String())
}

func toVBApplication(data []byte) []byte {
	if len(data) == 0 {
		return []byte("buf = Array()")
	}

	var result strings.Builder
	result.WriteString("buf = Array(")

	for i, b := range data {
		if i > 0 {
			result.WriteString(",")
		}
		if i > 1 && i%80 == 0 {
			result.WriteString(" _\r\n")
		}
		result.WriteString(fmt.Sprintf("%d", b))
	}
	result.WriteString(")\r\n")
	return []byte(result.String())
}
