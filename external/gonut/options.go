package gonut

// ArchType target architecture
type ArchType int

func (x ArchType) Name() string {
	switch x {
	case DONUT_ARCH_ANY:
		return "Any"
	case DONUT_ARCH_X86:
		return "x86"
	case DONUT_ARCH_X64:
		return "amd64"
	case DONUT_ARCH_X96:
		return "x86+amd64"
	default:
		return "Unknown"
	}
}

const (
	DONUT_ARCH_ANY ArchType = -1 // just for vbs,js and xsl files
	DONUT_ARCH_X86 ArchType = 1  // x86
	DONUT_ARCH_X64 ArchType = 2  // AMD64
	DONUT_ARCH_X96 ArchType = 3  // AMD64 + x86
)

// FormatType format type
type FormatType int

func (x FormatType) Name() string {
	switch x {
	case DONUT_FORMAT_BINARY:
		return "Binary"
	case DONUT_FORMAT_BASE64:
		return "Base64"
	case DONUT_FORMAT_C:
		return "C"
	case DONUT_FORMAT_RUBY:
		return "Ruby"
	case DONUT_FORMAT_PYTHON:
		return "Python"
	case DONUT_FORMAT_POWERSHELL:
		return "PowerShell"
	case DONUT_FORMAT_CSHARP:
		return "C#"
	case DONUT_FORMAT_HEX:
		return "Hex"
	case DONUT_FORMAT_UUID:
		return "UUID"
	case DONUT_FORMAT_GO:
		return "Golang"
	case DONUT_FORMAT_RUST:
		return "Rust"
	default:
		return "Unknown"
	}
}

const (
	DONUT_FORMAT_BINARY     FormatType = 1
	DONUT_FORMAT_BASE64     FormatType = 2
	DONUT_FORMAT_C          FormatType = 3
	DONUT_FORMAT_RUBY       FormatType = 4
	DONUT_FORMAT_PYTHON     FormatType = 5
	DONUT_FORMAT_POWERSHELL FormatType = 6
	DONUT_FORMAT_CSHARP     FormatType = 7
	DONUT_FORMAT_HEX        FormatType = 8
	DONUT_FORMAT_UUID       FormatType = 9
	DONUT_FORMAT_GO         FormatType = 10
	DONUT_FORMAT_RUST       FormatType = 11
)

// CompressionType compression engine (Gonut)
type CompressionType uint32

func (x CompressionType) Name() string {
	switch x {
	case GONUT_COMPRESS_NONE:
		return "None"
	case GONUT_COMPRESS_APLIB:
		return "aPLib (experimental)"
	case GONUT_COMPRESS_LZNT1_RTL:
		return "LZNT1 (RtlCompressBuffer)"
	case GONUT_COMPRESS_XPRESS_RTL:
		return "Xpress (RtlCompressBuffer)"
	case GONUT_COMPRESS_LZNT1:
		return "LZNT1 (experimental)"
	case GONUT_COMPRESS_XPRESS:
		return "Xpress (experimental)"
	default:
		return "Unknown"
	}
}

// gonut compression engine
const (
	GONUT_COMPRESS_NONE       CompressionType = 1
	GONUT_COMPRESS_APLIB      CompressionType = 2
	GONUT_COMPRESS_LZNT1_RTL  CompressionType = 3 // windows only
	GONUT_COMPRESS_XPRESS_RTL CompressionType = 4 // windows only
	GONUT_COMPRESS_LZNT1      CompressionType = 5
	GONUT_COMPRESS_XPRESS     CompressionType = 6
)

// DonutCompressionType compression engine
type DonutCompressionType uint32

func (x DonutCompressionType) Name() string {
	switch x {
	case DONUT_COMPRESS_NONE:
		return "None"
	case DONUT_COMPRESS_APLIB:
		return "aPLib"
	case DONUT_COMPRESS_LZNT1:
		return "LZNT1"
	case DONUT_COMPRESS_XPRESS:
		return "Xpress"
	default:
		return "Unknown"
	}
}

// donut compression engine
const (
	DONUT_COMPRESS_NONE   DonutCompressionType = 1
	DONUT_COMPRESS_APLIB  DonutCompressionType = 2
	DONUT_COMPRESS_LZNT1  DonutCompressionType = 3 // COMPRESSION_FORMAT_LZNT1
	DONUT_COMPRESS_XPRESS DonutCompressionType = 4 // COMPRESSION_FORMAT_XPRESS
)

// EntropyType entropy level
type EntropyType uint32

func (x EntropyType) Name() string {
	switch x {
	case DONUT_ENTROPY_NONE:
		return "None"
	case DONUT_ENTROPY_RANDOM:
		return "Random names"
	case DONUT_ENTROPY_DEFAULT:
		return "Random names + Encryption"
	default:
		return "Unknown"
	}
}

const (
	DONUT_ENTROPY_NONE    EntropyType = 1 // don't use any entropy
	DONUT_ENTROPY_RANDOM  EntropyType = 2 // use random names
	DONUT_ENTROPY_DEFAULT EntropyType = 3 // use random names + symmetric encryption
)

// ExitType misc options
type ExitType uint32

func (x ExitType) Name() string {
	switch x {
	case DONUT_OPT_EXIT_THREAD:
		return "Thread"
	case DONUT_OPT_EXIT_PROCESS:
		return "Process"
	case DONUT_OPT_EXIT_BLOCK:
		return "Block"
	default:
		return "Unknown"
	}
}

const (
	DONUT_OPT_EXIT_THREAD  ExitType = 1 // return to the caller which calls RtlExitUserThread
	DONUT_OPT_EXIT_PROCESS ExitType = 2 // call RtlExitUserProcess to terminate host process
	DONUT_OPT_EXIT_BLOCK   ExitType = 3 // after the main shellcode ends, do not exit or cleanup and block indefinitely
)

// BypassType AMSI/WLDP/ETW options
type BypassType uint32

func (x BypassType) Name() string {
	switch x {
	case DONUT_BYPASS_NONE:
		return "None"
	case DONUT_BYPASS_ABORT:
		return "Abort after failure"
	case DONUT_BYPASS_CONTINUE:
		return "Continue after failure"
	default:
		return "Unknown"
	}
}

const (
	DONUT_BYPASS_NONE     BypassType = 1 // Disables bypassing AMSI/WLDP/ETW
	DONUT_BYPASS_ABORT    BypassType = 2 // If bypassing AMSI/WLDP/ETW fails, the loader stops running
	DONUT_BYPASS_CONTINUE BypassType = 3 // If bypassing AMSI/WLDP/ETW fails, the loader continues running
)

// HeadersType Preserve PE headers options
type HeadersType uint32

func (x HeadersType) Name() string {
	switch x {
	case DONUT_HEADERS_OVERWRITE:
		return "Overwrite"
	case DONUT_HEADERS_KEEP:
		return "Keep"
	default:
		return "Unknown"
	}
}

const (
	DONUT_HEADERS_OVERWRITE HeadersType = 1 // Overwrite PE headers
	DONUT_HEADERS_KEEP      HeadersType = 2 // Preserve PE headers
)

type BoolType bool

func (x BoolType) Name() string {
	if x {
		return "Yes"
	} else {
		return "No"
	}
}

func (x BoolType) ToUint32() uint32 {
	if x {
		return 1
	} else {
		return 0
	}
}

const (
	GONUT_UNICODE_TRUE  BoolType = true
	GONUT_UNICODE_FALSE BoolType = false
)

const (
	GONUT_THREAD_TRUE  BoolType = true
	GONUT_THREAD_FALSE BoolType = false
)
