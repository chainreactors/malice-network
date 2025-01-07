package gonut

import "unsafe"

// ModuleType module type
// donut/include/donut.h
type ModuleType uint32

func (x ModuleType) Name() string {
	switch x {
	case DONUT_MODULE_NET_DLL:
		return ".NET DLL"
	case DONUT_MODULE_NET_EXE:
		return ".NET EXE"
	case DONUT_MODULE_DLL:
		return "DLL"
	case DONUT_MODULE_EXE:
		return "EXE"
	case DONUT_MODULE_VBS:
		return "VBScript"
	case DONUT_MODULE_JS:
		return "JScript"
	default:
		return "Unknown"
	}
}

// donut/include/donut.h
const (
	DONUT_MODULE_NET_DLL ModuleType = 1 // .NET DLL. Requires class and method
	DONUT_MODULE_NET_EXE ModuleType = 2 // .NET EXE. Executes Main if no class and method provided
	DONUT_MODULE_DLL     ModuleType = 3 // Unmanaged DLL, function is optional
	DONUT_MODULE_EXE     ModuleType = 4 // Unmanaged EXE
	DONUT_MODULE_VBS     ModuleType = 5 // VBScript
	DONUT_MODULE_JS      ModuleType = 6 // JavaScript or JScript
)

// DonutModule
// donut/include/donut.h
// typedef struct _DONUT_MODULE { ... }
type DonutModule struct {
	Type     ModuleType           // EXE/DLL/JS/VBS
	Thread   uint32               // run entrypoint of unmanaged EXE as a thread
	Compress DonutCompressionType // indicates engine used for compression

	Runtime [DONUT_MAX_NAME]byte // runtime version for .NET EXE/DLL
	Domain  [DONUT_MAX_NAME]byte // domain name to use for .NET EXE/DLL
	Cls     [DONUT_MAX_NAME]byte // name of class and optional namespace for .NET EXE/DLL
	Method  [DONUT_MAX_NAME]byte // name of method to invoke for .NET DLL or api for unmanaged DLL

	Args    [DONUT_MAX_NAME]byte // (Param) string arguments for both managed and unmanaged DLL/EXE
	Unicode uint32               // convert param to unicode for unmanaged DLL function

	Sig [DONUT_SIG_LEN]byte // string to verify decryption
	Mac uint64              // hash of sig, to verify decryption was ok

	ZLen uint32 // compressed size of EXE/DLL/JS/VBS file
	Len  uint32 // real size of EXE/DLL/JS/VBS file

	Data [4]byte // data of EXE/DLL/JS/VBS file
}

type Module struct {
	DonutModule
	Data []byte // data of EXE/DLL/JS/VBS file
}

func (m *Module) ToBytes() (result []byte) {
	moduleSize := unsafe.Sizeof(m.DonutModule)

	result = UnsafeStructToBytes(&m.DonutModule)[:unsafe.Offsetof(m.DonutModule.Data)]
	result = append(result, m.Data...)

	if len(result) < len(m.Data)+int(moduleSize) {
		padding := len(m.Data) + int(moduleSize) - len(result)
		result = append(result, make([]byte, padding)...)
	}

	return result
}
