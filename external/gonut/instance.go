package gonut

import "unsafe"

// InstanceType instance type
type InstanceType uint32

func (x InstanceType) Name() string {
	switch x {
	case DONUT_INSTANCE_EMBED:
		return "Embedded"
	case DONUT_INSTANCE_HTTP:
		return "HTTP"
	case DONUT_INSTANCE_DNS:
		return "DNS"
	default:
		return "Unknown"
	}
}

const (
	DONUT_INSTANCE_EMBED InstanceType = 1 // Self-contained
	DONUT_INSTANCE_HTTP  InstanceType = 2 // Download from remote HTTP/HTTPS server
	DONUT_INSTANCE_DNS   InstanceType = 3 // Download from remote DNS server
)

// DonutInstance
// donut/include/donut.h
// typedef struct _DONUT_INSTANCE { ... }
type DonutInstance struct {
	Len uint32 // total size of instance

	Key Crypt // decrypts instance if encryption enabled (32 bytes total = 16+16)

	Iv uint64 // the 64-bit initial value for maru hash

	Hash [128]uint64 // holds up to 64 api hashes/addrs {api}

	ExitOpt ExitType    // 1 to call RtlExitUserProcess and terminate the host process, 2 to never exit or cleanup and block
	Entropy EntropyType // indicates entropy level
	OEP     uint32      // original entrypoint

	// everything from here is encrypted
	ApiCount uint32               // the 64-bit hashes of API required for instance to work
	DllNames [DONUT_MAX_NAME]byte // a list of DLL strings to load, separated by semi-colon

	DataName   [8]byte  // ".data"
	KernelBase [12]byte // "kernelbase"
	Amsi       [8]byte  // "amsi"
	Clr        [4]byte  // "clr"
	Wldp       [8]byte  // "wldp"
	Ntdll      [8]byte  // "ntdll"

	CmdSymbols [DONUT_MAX_NAME]byte // symbols related to command line
	ExitApi    [DONUT_MAX_NAME]byte // exit-related API

	Bypass             BypassType  // indicates behaviour of byassing AMSI/WLDP/ETW
	Headers            HeadersType // indicates whether to overwrite PE headers
	WldpQuery          [32]byte    // WldpQueryDynamicCodeTrust
	WldpIsApproved     [32]byte    // WldpIsClassInApprovedList
	AmsiInit           [16]byte    // AmsiInitialize
	AmsiScanBuf        [16]byte    // AmsiScanBuffer
	AmsiScanStr        [16]byte    // AmsiScanString
	EtwEventWrite      [16]byte    // EtwEventWrite
	EtwEventUnregister [20]byte    // EtwEventUnregister
	EtwRet64           [1]byte     // "ret" instruction for Etw
	EtwRet32           [4]byte     // "ret 14h" instruction for Etw

	Wscript    [8]byte  // WScript
	WscriptExe [12]byte // wscript.exe

	Decoy [MAX_PATH * 2]byte // path of decoy module

	X_IID_IUnknown  GUID
	X_IID_IDispatch GUID

	//  GUID required to load .NET assemblies
	X_CLSID_CLRMetaHost    GUID
	X_IID_ICLRMetaHost     GUID
	X_IID_ICLRRuntimeInfo  GUID
	X_CLSID_CorRuntimeHost GUID
	X_IID_ICorRuntimeHost  GUID
	X_IID_AppDomain        GUID

	//  GUID required to run VBS and JS files
	X_CLSID_ScriptLanguage        GUID // vbs or js
	X_IID_IHost                   GUID // wscript object
	X_IID_IActiveScript           GUID // engine
	X_IID_IActiveScriptSite       GUID // implementation
	X_IID_IActiveScriptSiteWindow GUID // basic GUI stuff
	X_IID_IActiveScriptParse32    GUID // parser
	X_IID_IActiveScriptParse64    GUID

	Type InstanceType // DONUT_INSTANCE_EMBED, DONUT_INSTANCE_HTTP

	Server   [DONUT_MAX_NAME]byte // staging server hosting donut module
	UserName [DONUT_MAX_NAME]byte // username for web server
	Password [DONUT_MAX_NAME]byte // password for web server
	HTTPReq  [8]byte              // just a buffer for "GET"

	Sig [DONUT_MAX_NAME]byte // string to hash
	Mac uint64               // to verify decryption ok

	ModuleKey Crypt  // used to decrypt module
	ModuleLen uint64 // total size of module

	Module DonutModule // Module
}

type Instance struct {
	DonutInstance
	ModuleData []byte
}

func (z *Instance) ToBytes() (result []byte) {
	moduleSize := unsafe.Sizeof(z.DonutInstance)

	result = UnsafeStructToBytes(&z.DonutInstance)[:unsafe.Offsetof(z.DonutInstance.Module)]
	result = append(result, z.ModuleData...)

	if len(result) < len(z.ModuleData)+int(moduleSize) {
		padding := len(z.ModuleData) + int(moduleSize) - len(result)
		result = append(result, make([]byte, padding)...)
	}

	return result
}
