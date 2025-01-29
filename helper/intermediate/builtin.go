package intermediate

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/chainreactors/malice-network/helper/utils/pe"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/chainreactors/mals"
	"github.com/kballard/go-shellquote"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/helper/encoders/hash"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/helper/utils/handler"
)

const (
	GroupEncode   = "encode"
	GroupArtifact = "artifact"
)

// lua package
const (
	BeaconPackage  = "beacon"
	RpcPackage     = "rpc"
	ArmoryPackage  = "armory"
	BuiltinPackage = "builtin"
)

type BuiltinCallback func(content interface{}) (bool, error)

func RegisterBuiltin(rpc clientrpc.MaliceRPCClient) {
	RegisterCustomBuiltin(rpc)
	RegisterEncodeFunc(rpc)
	for _, fn := range mals.RegisterGRPCBuiltin(RpcPackage, rpc) {
		err := RegisterInternalFunc(RpcPackage, fn.Name, fn, nil)
		if err != nil {
			logs.Log.Errorf("register internal function %s failed: %s", fn.Name, err)
		}
	}
}

func RegisterCustomBuiltin(rpc clientrpc.MaliceRPCClient) {
	// 构建 sacrifice 进程消息
	RegisterFunction("new_sacrifice", func(ppid uint32, hidden, blockDll, disableETW bool, argue string) (*implantpb.SacrificeProcess, error) {
		return NewSacrificeProcessMessage(ppid, hidden, blockDll, disableETW, argue)
	})
	AddHelper(
		"new_sacrifice",

		&mals.Helper{
			Short: "new sacrifice process config",
			Input: []string{
				"ppid: parent process id",
				"hidden",
				"blockDll",
				"disableETW",
				"argue: arguments",
			},
			Output: []string{
				"implantpb.SacrificeProcess",
			},
			Example: `
sac = new_sacrifice(123, false, false, false, "")
`,
		},
	)

	// 构建 x86 二进制消息
	RegisterFunction("new_86_executable", func(module, filename, argsStr string, sacrifice *implantpb.SacrificeProcess) (*implantpb.ExecuteBinary, error) {
		cmdline, err := shellquote.Split(argsStr)
		if err != nil {
			return nil, err
		}
		return NewExecutable(module, filename, cmdline, "x86", sacrifice)
	})
	AddHelper("new_86_executable",
		&mals.Helper{
			Short: "new x86 process execute binary config",
			Input: []string{
				"module",
				"filename: path to the binary",
				"argsStr: command line arguments",
				"sacrifice: sacrifice process",
			},
			Output: []string{
				"ExecuteBinary",
			},
			Example: `
sac = new_sacrifice(123, false, false, false, "")
new_86_exec = new_86_executable("module", "filename", "args", sac)
`})

	// 构建 64 位二进制消息
	RegisterFunction("new_64_executable", func(module, filename, argsStr string, sacrifice *implantpb.SacrificeProcess) (*implantpb.ExecuteBinary, error) {
		cmdline, err := shellquote.Split(argsStr)
		if err != nil {
			return nil, err
		}
		return NewExecutable(module, filename, cmdline, "amd64", sacrifice)
	})
	AddHelper("new_64_executable",
		&mals.Helper{
			Short: "new x64 process execute binary config",
			Input: []string{
				"module",
				"filename: path to the binary",
				"argsStr: command line arguments",
				"sacrifice: sacrifice process",
			},
			Output: []string{
				"ExecuteBinary",
			},
			Example: `
sac = new_sacrifice(123, false, false, false, "")
new_64_exec = new_64_executable("module", "filename", "args", sac)
`})

	RegisterFunction("new_bypass", func(bypassAmsi, bypassEtw, bypassWLDP bool) map[string]string {
		params := make(map[string]string)
		if bypassAmsi {
			params["bypass_amsi"] = ""
		}
		if bypassEtw {
			params["bypass_etw"] = ""
		}
		if bypassWLDP {
			params["bypass_wldp"] = ""
		}
		return params
	})

	RegisterFunction("new_bypass_all", func() map[string]string {
		return map[string]string{
			"bypass_amsi": "",
			"bypass_etw":  "",
			"bypass_wldp": "",
		}
	})

	AddHelper(
		"new_bypass",
		&mals.Helper{
			Short: "new bypass options",
			Input: []string{
				"bypassAMSI",
				"bypassETW",
				"bypassWLDP",
			},
			Output: []string{
				"param: table, {\n    bypass_amsi = \"\",\n    bypass_etw = \"\",\n    bypass_wldp = \"\"\n}",
			},
			Example: `
params = new_bypass(true, true, true)
`,
		},
	)

	AddHelper(
		"new_bypass_all",
		&mals.Helper{
			Short: "new bypass all options",
			Input: []string{},
			Output: []string{
				"map[string]string: contains all bypass options (AMSI, ETW, WLDP)",
			},
			Example: `
params = new_bypass_all()
`,
		},
	)
	// 构建新的二进制消息
	RegisterFunction("new_binary", func(module, filename string, args []string,
		output bool, timeout uint32, arch, process string,
		sacrifice *implantpb.SacrificeProcess) (*implantpb.ExecuteBinary, error) {
		return NewBinary(module, filename, args, output, timeout, arch, process, sacrifice)
	})

	AddHelper("new_binary",
		&mals.Helper{
			Short: "new execute binary config",
			Input: []string{
				"module",
				"filename: path to the binary",
				"args: command line arguments",
				"output",
				"timeout",
				"arch",
				"process",
				"sacrifice: sacrifice process",
			},
			Output: []string{
				"ExecuteBinary",
			},
			Example: `
sac = new_sacrifice(123, false, false, false, "")
new_bin = new_binary("module", "filename", "args", true, 100, "amd64", "process", sac)
`})

	// 等待任务结果
	RegisterFunction("wait", func(task *clientpb.Task) (*clientpb.TaskContext, error) {
		return WaitResult(rpc, task)
	})

	// 获取任务结果
	RegisterFunction("get", func(task *clientpb.Task, index int32) (*clientpb.TaskContext, error) {
		return GetResult(rpc, task, index)
	})

	// 打印任务
	RegisterFunction("taskprint", func(task *clientpb.TaskContext) (*implantpb.Spite, error) {
		return PrintTask(task)
	})

	// 打印 assembly
	RegisterFunction("assemblyprint", func(task *clientpb.TaskContext) (string, error) {
		err := handler.AssertStatusAndSpite(task.GetSpite(), types.MsgBinaryResponse)
		if err != nil {
			return "", err
		}
		s, _ := ParseAssembly(task.Spite)
		logs.Log.Console(s)
		return s, nil
	})

	RegisterFunction("callback_file", func(filename string) (BuiltinCallback, error) {
		return func(content interface{}) (bool, error) {
			_, ok := content.(string)
			if !ok {
				return false, fmt.Errorf("expect content tpye string, found %s", reflect.TypeOf(content).String())
			}
			err := os.WriteFile(filename, []byte(content.(string)), 0644)
			if err != nil {
				return false, err
			}
			return true, nil
		}, nil
	})

	RegisterFunction("callback_append", func(filename string) (BuiltinCallback, error) {
		return func(content interface{}) (bool, error) {
			_, ok := content.(string)
			if !ok {
				return false, fmt.Errorf("expect content tpye string, found %s", reflect.TypeOf(content).String())
			}
			f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
			if err != nil {
				return false, err
			}
			defer f.Close() // 确保在函数结束时关闭文件

			// 写入内容
			if _, err := f.Write([]byte(content.(string))); err != nil {
				return false, err
			}
			return true, nil
		}, nil
	})

	RegisterFunction("callback_discard", func() (BuiltinCallback, error) {
		return func(content interface{}) (bool, error) {
			return true, nil
		}, nil
	})

}

func RegisterEncodeFunc(rpc clientrpc.MaliceRPCClient) {
	// bof 参数格式化
	// single arg, pack_bof("Z", "aa")
	RegisterFunction("pack_bof", func(format string, arg string) (string, error) {
		if len(format) != 1 {
			return "", fmt.Errorf("format length must be 1")
		}
		return pe.PackArg(format[0], arg)
	})
	AddHelper("pack_bof",
		&mals.Helper{
			Short: "pack bof single argument",
			Input: []string{
				"format",
				"arg",
			},
			Output: []string{
				"string",
			},
			Example: `pack_bof("Z", "aa")`,
		})

	// args, pack_bof_args("ZZ", {"aa", "bb"})
	RegisterFunction("pack_bof_args", func(format string, args []string) ([]string, error) {
		if len(format) != len(args) {
			return nil, fmt.Errorf("%d format and %d args,  length mismatch", len(format), len(args))
		}
		var packedArgs []string
		for i, arg := range args {
			packedArgs = append(packedArgs, format[i:i+1]+arg)
		}
		return pe.PackArgs(packedArgs)
	})

	AddHelper(
		"pack_bof_args",
		&mals.Helper{
			Short: "pack bof arguments",
			Input: []string{
				"format",
				"args",
			},
			Output: []string{
				"[]string",
			},
			Example: `
pack_bof_args("ZZ", {"aa", "bb"})
`,
		})

	// mal pack
	RegisterFunction("pack_binary", func(data string) (string, error) {
		return pe.PackBinary(data), nil
	})

	RegisterFunction("format_path", func(s string) (string, error) {
		return fileutils.FormatWindowPath(s), nil
	})
	AddHelper(
		"format_path",
		&mals.Helper{
			Short: "format windows path",
			Input: []string{
				"s",
			},
			Output: []string{
				"string",
			},
			Example: `
format_path("C:\\Windows\\System32\\calc.exe")
`,
		})
	// Base64函数
	RegisterFunction("base64_encode", func(input string) (string, error) {
		return base64.StdEncoding.EncodeToString([]byte(input)), nil
	})
	AddHelper(
		"base64_encode",
		&mals.Helper{
			Group:   GroupEncode,
			CMDName: "base64_encode",
			Input: []string{
				"input",
			},
			Output: []string{
				"string",
			},
			Example: `base64_encode("hello")`,
		})

	RegisterFunction("base64_decode", func(input string) (string, error) {
		data, err := base64.StdEncoding.DecodeString(input)
		if err != nil {
			return "", err
		}
		return string(data), nil
	})

	AddHelper(
		"base64_decode",
		&mals.Helper{
			Group: GroupEncode,
			Short: "base64 decode",
			Input: []string{
				"input",
			},
			Output: []string{
				"string",
			},
			Example: `base64_decode("aGVsbG8=")`,
		})

	RegisterFunction("arg_hex", func(input string) (string, error) {
		return "hex::" + hash.Hexlify([]byte(input)), nil
	})
	AddHelper(
		"arg_hex",
		&mals.Helper{
			Group: GroupEncode,
			Short: "hexlify encode",
			Input: []string{
				"input",
			},
			Output: []string{
				"string",
			},
			Example: `arg_hex("aa")`,
		})

	// random string
	RegisterFunction("random_string", func(length int) (string, error) {
		charArray := []rune("abcdefghijklmnopqrstuvwxyz123456789")
		randomStr := ""
		for i := 0; i < length; i++ {
			index, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charArray))))
			randomStr += string(charArray[index.Int64()])
		}
		return randomStr, nil
	})
	AddHelper(
		"random_string",
		&mals.Helper{
			Group: GroupEncode,
			Short: "generate random string",
			Input: []string{
				"length",
			},
			Output: []string{
				"string",
			},
			Example: `random_string(10)`,
		})
	// fileExists
	RegisterFunction("file_exists", func(path string) (bool, error) {
		_, err := os.Stat(path)
		if err == nil {
			return true, nil
		}
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, nil
	})
	AddHelper(
		"file_exists",
		&mals.Helper{
			Group: GroupEncode,
			Short: "check file exists",
			Input: []string{
				"path",
			},
			Output: []string{
				"bool",
			},
			Example: `file_exists("C:\\Windows\\System32\\calc.exe")`,
		})
	// match re
	RegisterFunction("ismatch", func(pattern, text string) (bool, []string) {
		reg, err := regexp.Compile(pattern)
		if err != nil {
			fmt.Println("regexp compile error: ", err)
			return false, nil
		}
		matches := reg.FindStringSubmatch(text)
		if matches != nil {
			return true, matches[1:]
		}
		return false, nil
	})
	AddHelper(
		"ismatch",
		&mals.Helper{
			Group: GroupEncode,
			Short: "regexp match",
			Input: []string{
				"pattern",
				"text",
			},
			Output: []string{
				"bool",
				"[]string",
			},
			Example: `ismatch("([a-z]+) ([0-9]+)", "hello 123")`,
		})

	// timestamp

	RegisterFunction("timestamp", func() string {
		return fmt.Sprintf("%d", time.Now().UnixNano()/int64(time.Millisecond))
	})

	RegisterFunction("timestamp_format", func(optionalFormat string) string {
		return time.Now().Format(optionalFormat)
	})

	AddHelper(
		"timestamp",
		&mals.Helper{
			Group:   GroupEncode,
			Short:   "Get current timestamp in milliseconds or formatted date string.",
			Input:   []string{"string (optional, format)"},
			Output:  []string{"string"},
			Example: `timestampOrFormatted(), timestampOrFormatted("01/02 15:04")`,
		},
	)

	RegisterFunction("is_full_path", func(path string) bool {
		return fileutils.CheckFullPath(path)
	})

	RegisterFunction("get_sess_dir", func(sessid string) string {
		session_dir := filepath.Join(assets.GetTempDir(), sessid)
		if _, err := os.Stat(session_dir); os.IsNotExist(err) {
			err = os.MkdirAll(session_dir, 0700)
			if err != nil {
				logs.Log.Errorf(err.Error())
			}
		}
		return session_dir
	})

	// 0o744 0744
	RegisterFunction("parse_octal", func(octalString string) int64 {
		var result int64
		var err error
		if strings.HasPrefix(octalString, "0o") {
			result, err = strconv.ParseInt(octalString[2:], 8, 64)
		} else if strings.HasPrefix(octalString, "0") && len(octalString) > 1 {
			result, err = strconv.ParseInt(octalString[1:], 8, 64)
		} else {
			result, err = strconv.ParseInt(octalString, 8, 64)
		}
		if err != nil {
			return -1
		}
		return result
	})
	AddHelper(
		"parse_octal",
		&mals.Helper{
			Group: GroupEncode,
			Short: "parse octal string to int64",
			Input: []string{
				"octalString",
			},
			Output: []string{
				"int64",
			},
			Example: `parse_octal("0o744")`,
		})

	RegisterFunction("parse_hex", func(hexString string) int64 {
		if strings.HasPrefix(hexString, "0x") {
			result, _ := strconv.ParseInt(hexString[2:], 16, 64)
			return result
		} else {
			return -1
		}
	})
	AddHelper(
		"parse_hex",
		&mals.Helper{
			Group: GroupEncode,
			Short: "parse hex string to int64",
			Input: []string{
				"hexString",
			},
			Output: []string{
				"int64",
			},
			Example: `parse_hex("0x1f4")`,
		})

}
