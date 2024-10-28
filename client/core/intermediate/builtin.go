package intermediate

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/encoders/hash"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/helper/utils/file"
	"github.com/chainreactors/malice-network/helper/utils/handler"
	"github.com/chainreactors/malice-network/helper/utils/pe"
	"github.com/kballard/go-shellquote"
	"google.golang.org/protobuf/proto"
	"math/big"
	"os"
	"reflect"
	"regexp"
	"time"
)

type BuiltinCallback func(content interface{}) (bool, error)

func RegisterBuiltin(rpc clientrpc.MaliceRPCClient) {
	RegisterCustomBuiltin(rpc)
	RegisterGRPCBuiltin(rpc)
	RegisterEncodeFunc()
	RegisterPayloadFunction(rpc)
}

func RegisterCustomBuiltin(rpc clientrpc.MaliceRPCClient) {
	// 构建 x86 二进制消息
	RegisterFunction("new_86_executable", func(module, filename, argsStr string, sacrifice *implantpb.SacrificeProcess) (*implantpb.ExecuteBinary, error) {
		cmdline, err := shellquote.Split(argsStr)
		if err != nil {
			return nil, err
		}
		return NewExecutable(module, filename, cmdline, "x86", sacrifice)
	})

	// 构建 64 位二进制消息
	RegisterFunction("new_64_executable", func(module, filename, argsStr string, sacrifice *implantpb.SacrificeProcess) (*implantpb.ExecuteBinary, error) {
		cmdline, err := shellquote.Split(argsStr)
		if err != nil {
			return nil, err
		}
		return NewExecutable(module, filename, cmdline, "amd64", sacrifice)
	})

	// 构建新的二进制消息
	RegisterFunction("new_binary", func(module, filename string, args []string,
		output bool, timeout uint32, arch, process string,
		sacrifice *implantpb.SacrificeProcess) (*implantpb.ExecuteBinary, error) {
		return NewBinary(module, filename, args, output, timeout, arch, process, sacrifice)
	})

	// 构建 sacrifice 进程消息
	RegisterFunction("new_sacrifice", func(ppid int64, hidden, blockDll, disableETW bool, argue string) (*implantpb.SacrificeProcess, error) {
		return NewSacrificeProcessMessage(ppid, hidden, blockDll, disableETW, argue)
	})

	// 等待任务结果
	RegisterFunction("wait", func(task *clientpb.Task) (*clientpb.TaskContext, error) {
		return WaitResult(rpc, task)
	})

	// 获取任务结果
	RegisterFunction("get", func(task *clientpb.Task, index int32) (*clientpb.TaskContext, error) {
		return GetResult(rpc, task, index)
	})

	// bof 参数格式化
	// single arg, pack_bof("Z", "aa")
	RegisterFunction("pack_bof", func(format string, arg string) (string, error) {
		if len(format) != 1 {
			return "", fmt.Errorf("format length must be 1")
		}
		return pe.PackArg(format[0], arg)
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

	RegisterFunction("arg_hex", func(input string) (string, error) {
		return "hex::" + hash.Hexlify([]byte(input)), nil
	})

	RegisterFunction("format_path", func(s string) (string, error) {
		return file.FormatWindowPath(s), nil
	})

	// 打印任务
	RegisterFunction("taskprint", func(task *clientpb.TaskContext) (*implantpb.Spite, error) {
		return PrintTask(task)
	})

	// 打印 assembly
	RegisterFunction("assemblyprint", func(task *clientpb.TaskContext) (string, error) {
		err := handler.AssertStatusAndResponse(task.GetSpite(), types.MsgBinaryResponse)
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

func RegisterGRPCBuiltin(rpc clientrpc.MaliceRPCClient) {
	rpcType := reflect.TypeOf(rpc)
	rpcValue := reflect.ValueOf(rpc)

	for i := 0; i < rpcType.NumMethod(); i++ {
		method := rpcType.Method(i)
		methodName := method.Name

		// 忽略流式方法
		methodReturnType := method.Type.Out(0)
		if methodReturnType.Kind() == reflect.Interface && methodReturnType.Name() == "ClientStream" {
			continue
		}

		// 将方法包装为 InternalFunc
		rpcFunc := func(args ...interface{}) (interface{}, error) {
			// 检查是否传入了两个参数
			if len(args) != 2 {
				return nil, fmt.Errorf("expected 2 arguments: context and proto.Message")
			}

			// 确保第一个参数是 context.Context
			ctx, ok := args[0].(context.Context)
			if !ok {
				return nil, fmt.Errorf("first argument must be context.Context")
			}

			// 确保第二个参数是 proto.Message
			msg, ok := args[1].(proto.Message)
			if !ok {
				return nil, fmt.Errorf("second argument must be proto.Message")
			}

			// 准备调用方法的参数列表
			callArgs := []reflect.Value{
				reflect.ValueOf(ctx), // context.Context
				reflect.ValueOf(msg), // proto.Message
			}

			// 调用方法
			results := rpcValue.MethodByName(methodName).Call(callArgs)

			// 处理返回值
			var result interface{}
			if len(results) > 0 {
				result = results[0].Interface()
			}

			var err error
			if len(results) > 1 {
				if e, ok := results[1].Interface().(error); ok {
					err = e
				}
			}

			return result, err
		}

		// 创建 InternalFunc 实例并设置真实的参数和返回值类型
		internalFunc := GetInternalFuncSignature(method.Func.Interface())
		internalFunc.Func = rpcFunc
		internalFunc.ArgTypes = internalFunc.ArgTypes[1:3]

		// 注册函数
		RegisterInternalFunc(RpcPackage, methodName, internalFunc, nil)
	}
}

func RegisterEncodeFunc() {
	// Base64函数
	RegisterFunction("base64_encode", func(input string) (string, error) {
		return base64.StdEncoding.EncodeToString([]byte(input)), nil
	})
	RegisterFunction("base64_decode", func(input string) (string, error) {
		data, err := base64.StdEncoding.DecodeString(input)
		if err != nil {
			return "", err
		}
		return string(data), nil
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
	// timestamp
	RegisterFunction("timestampMillis", func() int64 {
		timestampMillis := time.Now().UnixNano() / int64(time.Millisecond)
		return timestampMillis
	})
	// tstamp
	RegisterFunction("tstamp", func(timestampMillis int64) string {
		seconds := timestampMillis / 1000
		nanoseconds := (timestampMillis % 1000) * int64(time.Millisecond)
		t := time.Unix(seconds, nanoseconds)
		return t.Format("01/02 15:04")
	})

}
func RegisterCSFunction(name string, fn interface{}) {
	// 生成shellcode
	RegisterFunction("payload_local", func(shellcode_path string) (string, error) {
		if shellcode_path != "" {
			shellcode, _ := os.ReadFile(shellcode_path)
			if _, err := os.Stat(shellcode_path); os.IsNotExist(err) {
				return "", fmt.Errorf("shellcode file does not exist: %s", shellcode_path)
			}
			return string(shellcode), nil
		} else {
			return "", nil
		}
	})
}

func RegisterPayloadFunction(rpc clientrpc.MaliceRPCClient) {
	RegisterFunction("payload_local", func(shellcodePath string) (string, error) {
		if shellcodePath != "" {
			shellcode, _ := os.ReadFile(shellcodePath)
			if _, err := os.Stat(shellcodePath); os.IsNotExist(err) {
				return "", fmt.Errorf("shellcode file does not exist: %s", shellcodePath)
			}
			return string(shellcode), nil
		} else {
			return "shellcode123", nil
		}
	})
}
