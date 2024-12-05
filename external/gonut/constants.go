package gonut

const MAX_PATH = 260

// donut/include/donut.h
const (
	DONUT_MAX_NAME    = 256 // maximum length of string for domain, class, method and parameter names
	DONUT_MAX_DLL     = 8   // maximum number of DLL supported by instance
	DONUT_MAX_MODNAME = 8
	DONUT_SIG_LEN     = 8 // 64-bit string to verify decryption ok
	DONUT_VER_LEN     = 32
	DONUT_DOMAIN_LEN  = 8
)

// donut/include/donut.h
const (
	NTDLL_DLL    = "ntdll.dll"
	KERNEL32_DLL = "kernel32.dll"
	ADVAPI32_DLL = "advapi32.dll"
	CRYPT32_DLL  = "crypt32.dll"
	MSCOREE_DLL  = "mscoree.dll"
	OLE32_DLL    = "ole32.dll"
	OLEAUT32_DLL = "oleaut32.dll"
	WININET_DLL  = "wininet.dll"
	COMBASE_DLL  = "combase.dll"
	USER32_DLL   = "user32.dll"
	SHLWAPI_DLL  = "shlwapi.dll"
	SHELL32_DLL  = "shell32.dll"
)

// GUID
// donut/include/donut.h
// typedef struct _GUID { ... }
type GUID struct {
	Data1 uint32   // DWORD
	Data2 uint16   // WORD
	Data3 uint16   // WORD
	Data4 [8]uint8 // BYTE
}

var (
	// required to load .NET assemblies

	X_CLSID_CorRuntimeHost = GUID{0xcb2f6723, 0xab3a, 0x11d2, [8]uint8{0x9c, 0x40, 0x00, 0xc0, 0x4f, 0xa3, 0x0a, 0x3e}}
	X_IID_ICorRuntimeHost  = GUID{0xcb2f6722, 0xab3a, 0x11d2, [8]uint8{0x9c, 0x40, 0x00, 0xc0, 0x4f, 0xa3, 0x0a, 0x3e}}
	X_CLSID_CLRMetaHost    = GUID{0x9280188d, 0x0e8e, 0x4867, [8]uint8{0xb3, 0x0c, 0x7f, 0xa8, 0x38, 0x84, 0xe8, 0xde}}
	X_IID_ICLRMetaHost     = GUID{0xD332DB9E, 0xB9B3, 0x4125, [8]uint8{0x82, 0x07, 0xA1, 0x48, 0x84, 0xF5, 0x32, 0x16}}
	X_IID_ICLRRuntimeInfo  = GUID{0xBD39D1D2, 0xBA2F, 0x486a, [8]uint8{0x89, 0xB0, 0xB4, 0xB0, 0xCB, 0x46, 0x68, 0x91}}
	X_IID_AppDomain        = GUID{0x05F696DC, 0x2B29, 0x3663, [8]uint8{0xAD, 0x8B, 0xC4, 0x38, 0x9C, 0xF2, 0xA7, 0x13}}

	// required to load VBS and JS files

	X_IID_IUnknown                = GUID{0x00000000, 0x0000, 0x0000, [8]uint8{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
	X_IID_IDispatch               = GUID{0x00020400, 0x0000, 0x0000, [8]uint8{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
	X_IID_IHost                   = GUID{0x91afbd1b, 0x5feb, 0x43f5, [8]uint8{0xb0, 0x28, 0xe2, 0xca, 0x96, 0x06, 0x17, 0xec}}
	X_IID_IActiveScript           = GUID{0xbb1a2ae1, 0xa4f9, 0x11cf, [8]uint8{0x8f, 0x20, 0x00, 0x80, 0x5f, 0x2c, 0xd0, 0x64}}
	X_IID_IActiveScriptSite       = GUID{0xdb01a1e3, 0xa42b, 0x11cf, [8]uint8{0x8f, 0x20, 0x00, 0x80, 0x5f, 0x2c, 0xd0, 0x64}}
	X_IID_IActiveScriptSiteWindow = GUID{0xd10f6761, 0x83e9, 0x11cf, [8]uint8{0x8f, 0x20, 0x00, 0x80, 0x5f, 0x2c, 0xd0, 0x64}}
	X_IID_IActiveScriptParse32    = GUID{0xbb1a2ae2, 0xa4f9, 0x11cf, [8]uint8{0x8f, 0x20, 0x00, 0x80, 0x5f, 0x2c, 0xd0, 0x64}}
	X_IID_IActiveScriptParse64    = GUID{0xc7ef7658, 0xe1ee, 0x480e, [8]uint8{0x97, 0xea, 0xd5, 0x2c, 0xb4, 0xd7, 0x6d, 0x17}}
	X_CLSID_VBScript              = GUID{0xB54F3741, 0x5B07, 0x11cf, [8]uint8{0xA4, 0xB0, 0x00, 0xAA, 0x00, 0x4A, 0x55, 0xE8}}
	X_CLSID_JScript               = GUID{0xF414C260, 0x6AC0, 0x11CF, [8]uint8{0xB6, 0xD1, 0x00, 0xAA, 0x00, 0xBB, 0xBB, 0x58}}
)

// ApiImport
// donut/include/donut.h
// typedef struct _API_IMPORT { ... }
type ApiImport struct {
	Module string
	Name   string
}

// DLL_NAMES required for each API used by the loader
// donut/donut.c
// #define DLL_NAMES "ole32;oleaut32;wininet;mscoree;shell32"
const (
	DLL_NAMES = "ole32;oleaut32;wininet;mscoree;shell32"
)

// ApiImports These must be in the same order as the DONUT_INSTANCE structure defined in donut.h
// donut/donut.c
// static API_IMPORT api_imports[] = { ... }
var ApiImports = []ApiImport{
	{KERNEL32_DLL, "LoadLibraryA"},
	{KERNEL32_DLL, "GetProcAddress"},
	{KERNEL32_DLL, "GetModuleHandleA"},
	{KERNEL32_DLL, "VirtualAlloc"},
	{KERNEL32_DLL, "VirtualFree"},
	{KERNEL32_DLL, "VirtualQuery"},
	{KERNEL32_DLL, "VirtualProtect"},
	{KERNEL32_DLL, "Sleep"},
	{KERNEL32_DLL, "MultiByteToWideChar"},
	{KERNEL32_DLL, "GetUserDefaultLCID"},
	{KERNEL32_DLL, "WaitForSingleObject"},
	{KERNEL32_DLL, "CreateThread"},
	{KERNEL32_DLL, "CreateFileA"},
	{KERNEL32_DLL, "GetFileSizeEx"},
	{KERNEL32_DLL, "GetThreadContext"},
	{KERNEL32_DLL, "GetCurrentThread"},
	{KERNEL32_DLL, "GetCurrentProcess"},
	{KERNEL32_DLL, "GetCommandLineA"},
	{KERNEL32_DLL, "GetCommandLineW"},
	{KERNEL32_DLL, "HeapAlloc"},
	{KERNEL32_DLL, "HeapReAlloc"},
	{KERNEL32_DLL, "GetProcessHeap"},
	{KERNEL32_DLL, "HeapFree"},
	{KERNEL32_DLL, "GetLastError"},
	{KERNEL32_DLL, "CloseHandle"},

	{SHELL32_DLL, "CommandLineToArgvW"},

	{OLEAUT32_DLL, "SafeArrayCreate"},
	{OLEAUT32_DLL, "SafeArrayCreateVector"},
	{OLEAUT32_DLL, "SafeArrayPutElement"},
	{OLEAUT32_DLL, "SafeArrayDestroy"},
	{OLEAUT32_DLL, "SafeArrayGetLBound"},
	{OLEAUT32_DLL, "SafeArrayGetUBound"},
	{OLEAUT32_DLL, "SysAllocString"},
	{OLEAUT32_DLL, "SysFreeString"},
	{OLEAUT32_DLL, "LoadTypeLib"},

	{WININET_DLL, "InternetCrackUrlA"},
	{WININET_DLL, "InternetOpenA"},
	{WININET_DLL, "InternetConnectA"},
	{WININET_DLL, "InternetSetOptionA"},
	{WININET_DLL, "InternetReadFile"},
	{WININET_DLL, "InternetQueryDataAvailable"},
	{WININET_DLL, "InternetCloseHandle"},
	{WININET_DLL, "HttpOpenRequestA"},
	{WININET_DLL, "HttpSendRequestA"},
	{WININET_DLL, "HttpQueryInfoA"},

	{MSCOREE_DLL, "CorBindToRuntime"},
	{MSCOREE_DLL, "CLRCreateInstance"},

	{OLE32_DLL, "CoInitializeEx"},
	{OLE32_DLL, "CoCreateInstance"},
	{OLE32_DLL, "CoUninitialize"},

	{NTDLL_DLL, "RtlEqualUnicodeString"},
	{NTDLL_DLL, "RtlEqualString"},
	{NTDLL_DLL, "RtlUnicodeStringToAnsiString"},
	{NTDLL_DLL, "RtlInitUnicodeString"},
	{NTDLL_DLL, "RtlExitUserThread"},
	{NTDLL_DLL, "RtlExitUserProcess"},
	{NTDLL_DLL, "RtlCreateUnicodeString"},
	{NTDLL_DLL, "RtlGetCompressionWorkSpaceSize"},
	{NTDLL_DLL, "RtlDecompressBuffer"},
	{NTDLL_DLL, "NtContinue"},
	{NTDLL_DLL, "NtCreateSection"},
	{NTDLL_DLL, "NtMapViewOfSection"},
	{NTDLL_DLL, "NtUnmapViewOfSection"},
	//{KERNEL32_DLL, "AddVectoredExceptionHandler"},
	//{KERNEL32_DLL, "RemoveVectoredExceptionHandler"},
	//{NTDLL_DLL,    "RtlFreeUnicodeString"},
	//{NTDLL_DLL,    "RtlFreeString"},

	// v1.1 update
	// Module Overloading 相关
	{NTDLL_DLL, "RtlCreateUnicodeString"}, // 创建 Unicode 字符串
	{NTDLL_DLL, "NtCreateSection"},        // 创建内存段
	{NTDLL_DLL, "NtMapViewOfSection"},     // 映射内存段视图
	{NTDLL_DLL, "NtUnmapViewOfSection"},   // 取消映射内存段视图

	// 压缩相关
	{NTDLL_DLL, "RtlGetCompressionWorkSpaceSize"}, // 获取压缩工作空间大小
	{NTDLL_DLL, "RtlDecompressBuffer"},            // 解压缩缓冲区

	// 异常处理相关
	{NTDLL_DLL, "NtContinue"},

	{"", ""}, // last one always contains two NULL pointers
}
