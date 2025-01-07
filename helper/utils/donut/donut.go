package donut

import (
	"github.com/wabzsy/gonut"
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
	config := gonut.DefaultConfig()
	config.Input = filename
	config.InputBin = pe
	config.Output = ""
	config.Arch = getDonutArch(arch)
	config.Args = params
	config.Bypass = gonut.DONUT_BYPASS_CONTINUE
	config.Format = gonut.DONUT_FORMAT_BINARY
	config.Entropy = gonut.DONUT_ENTROPY_NONE
	config.GonutCompress = gonut.GONUT_COMPRESS_NONE
	config.ExitOpt = gonut.DONUT_OPT_EXIT_THREAD
	config.Unicode = gonut.BoolType(isUnicode)
	config.Thread = gonut.BoolType(createNewThread)

	o := gonut.New(config)
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
	config := gonut.DefaultConfig()
	config.Input = filename
	config.InputBin = assembly
	config.Output = ""
	config.Arch = getDonutArch(arch)
	config.Args = params
	config.Class = className
	config.Method = method
	config.Domain = appDomain
	config.Runtime = "v4.0.30319"
	config.Bypass = gonut.DONUT_BYPASS_CONTINUE
	config.Format = gonut.DONUT_FORMAT_BINARY
	config.Entropy = gonut.DONUT_ENTROPY_DEFAULT
	config.Unicode = gonut.BoolType(false)

	o := gonut.New(config)
	if err := o.Create(); err != nil {
		return nil, err
	}

	return o.PicData, nil
}

func getDonutArch(arch string) gonut.ArchType {
	switch strings.ToLower(arch) {
	case "x32", "386":
		return gonut.DONUT_ARCH_X86
	case "x64", "amd64":
		return gonut.DONUT_ARCH_X64
	case "x84":
		return gonut.DONUT_ARCH_X96
	default:
		return gonut.DONUT_ARCH_X96
	}
}
