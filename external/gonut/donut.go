package gonut

import (
	"os"
	"path/filepath"
	"strings"
)

// DonutShellcodeFromFile 从给定的 PE 文件生成 Donut shellcode
func DonutShellcodeFromFile(filePath string, arch string, params string) (data []byte, err error) {
	pe, err := os.ReadFile(filePath)
	if err != nil {
		return
	}
	return DonutShellcodeFromPE(filepath.Base(filePath), pe, arch, params, false, true)
}

// DonutShellcodeFromPE 从给定的 PE 数据生成 Donut shellcode
func DonutShellcodeFromPE(filename string, pe []byte, arch string, params string, isUnicode bool, createNewThread bool) (data []byte, err error) {
	config := DefaultConfig()
	config.Input = filename
	config.InputBin = pe
	config.Output = ""
	config.Arch = getDonutArch(arch)
	//config.Arch = gonut.DONUT_ARCH_X96
	config.Args = params
	config.Bypass = DONUT_BYPASS_NONE
	config.Format = DONUT_FORMAT_BINARY
	config.Entropy = DONUT_ENTROPY_NONE
	config.GonutCompress = GONUT_COMPRESS_NONE
	config.ExitOpt = DONUT_OPT_EXIT_PROCESS
	config.Headers = DONUT_HEADERS_OVERWRITE
	config.Unicode = BoolType(isUnicode)
	config.Thread = BoolType(createNewThread)
	o := New(config)
	if err = o.Create(); err != nil {
		return nil, err
	}

	return o.PicData, nil
}

func DonutFromAssemblyFromFile(filePath string, arch string, params string, method string, className string, appDomain string) ([]byte, error) {
	assembly, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return DonutFromAssembly(filePath, assembly, arch, params, method, className, appDomain)
}

// DonutFromAssembly 从 .NET 程序集生成 donut shellcode
func DonutFromAssembly(filename string, assembly []byte, arch string, params string, method string, className string, appDomain string) ([]byte, error) {
	config := DefaultConfig()
	config.Input = filename
	config.InputBin = assembly
	config.Output = ""
	config.Arch = getDonutArch(arch)
	config.Args = params
	config.Class = className
	config.Method = method
	config.Domain = appDomain
	config.Runtime = "v4.0.30319"
	config.Bypass = DONUT_BYPASS_CONTINUE
	config.Format = DONUT_FORMAT_BINARY
	config.Entropy = DONUT_ENTROPY_DEFAULT
	config.Unicode = BoolType(false)

	o := New(config)
	if err := o.Create(); err != nil {
		return nil, err
	}

	return o.PicData, nil
}

func getDonutArch(arch string) ArchType {
	switch strings.ToLower(arch) {
	case "x32", "386":
		return DONUT_ARCH_X86
	case "x64", "amd64":
		return DONUT_ARCH_X64
	case "x84":
		return DONUT_ARCH_X96
	default:
		return DONUT_ARCH_X96
	}
}
