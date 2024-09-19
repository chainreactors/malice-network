package intermediate

import (
	"context"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/core/intermediate/builtin"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/helper/utils/handler"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/kballard/go-shellquote"
	"google.golang.org/protobuf/proto"
	"reflect"
	"strings"
)

type InternalFunc struct {
	Name           string
	Raw            interface{}
	Func           func(...interface{}) (interface{}, error)
	FinishCallback ImplantCallback // implant callback
	DoneCallback   ImplantCallback
	ArgTypes       []reflect.Type
	ReturnTypes    []reflect.Type
}

type ImplantCallback func(content *clientpb.TaskContext) (string, error)

var InternalFunctions = make(map[string]*InternalFunc)

func Register(rpc clientrpc.MaliceRPCClient) {
	RegisterBuiltinFunc(rpc)
	RegisterGRPCFunc(rpc)
}

// RegisterInternalFunc 注册并生成 Lua 定义文件
func RegisterInternalFunc(name string, fn *InternalFunc, callback ImplantCallback) error {
	name = strings.ReplaceAll(name, "-", "_")
	if callback != nil {
		fn.FinishCallback = callback
	}
	if _, ok := InternalFunctions[name]; ok {
		return fmt.Errorf("function %s already registered", name)
	}
	fn.Name = name
	InternalFunctions[name] = fn
	return nil
}

func RegisterInternalDoneCallback(name string, callback ImplantCallback) error {
	name = strings.ReplaceAll(name, "-", "_")
	if _, ok := InternalFunctions[name]; !ok {
		return fmt.Errorf("function %s not found", name)
	}
	InternalFunctions[name].DoneCallback = callback
	return nil
}

// 获取函数的参数和返回值类型
func GetInternalFuncSignature(fn interface{}) *InternalFunc {
	fnType := reflect.TypeOf(fn)

	// 获取参数类型
	numArgs := fnType.NumIn()
	argTypes := make([]reflect.Type, numArgs)
	for i := 0; i < numArgs; i++ {
		argTypes[i] = fnType.In(i)
	}

	// 获取返回值类型
	numReturns := fnType.NumOut()
	// 如果最后一个返回值是 error 类型，忽略它
	if numReturns > 0 && fnType.Out(numReturns-1) == reflect.TypeOf((*error)(nil)).Elem() {
		numReturns--
	}
	returnTypes := make([]reflect.Type, numReturns)
	for i := 0; i < numReturns; i++ {
		returnTypes[i] = fnType.Out(i)
	}
	return &InternalFunc{
		Raw:         fn,
		ArgTypes:    argTypes,
		ReturnTypes: returnTypes,
	}
}

func RegisterFunction(name string, fn interface{}) {
	wrappedFunc := WrapInternalFunc(fn)
	err := RegisterInternalFunc(name, wrappedFunc, nil)
	if err != nil {
		return
	}
}

func WrapInternalFunc(fun interface{}) *InternalFunc {
	internalFunc := GetInternalFuncSignature(fun)

	internalFunc.Func = func(params ...interface{}) (interface{}, error) {
		funcValue := reflect.ValueOf(fun)
		funcType := funcValue.Type()

		// 检查函数的参数数量是否匹配
		if funcType.NumIn() != len(params) {
			return nil, fmt.Errorf("expected %d arguments, got %d", funcType.NumIn(), len(params))
		}

		// 构建参数切片并检查参数类型
		in := make([]reflect.Value, len(params))
		for i, param := range params {
			expectedType := funcType.In(i)
			if reflect.TypeOf(param) != expectedType {
				return nil, fmt.Errorf("argument %d should be %v, got %v", i+1, expectedType, reflect.TypeOf(param))
			}
			in[i] = reflect.ValueOf(param)
		}

		// 调用原始函数并获取返回值
		results := funcValue.Call(in)

		// 处理返回值
		var result interface{}
		if len(results) > 0 {
			result = results[0].Interface()
		}

		var err error
		// 如果函数返回了多个值，最后一个值通常是 error
		if len(results) > 1 {
			if e, ok := results[len(results)-1].Interface().(error); ok {
				err = e
			}
		}

		return result, err
	}
	return internalFunc
}

func RegisterBuiltinFunc(rpc clientrpc.MaliceRPCClient) {
	// 构建 x86 二进制消息
	RegisterFunction("new_86_executable", func(module, filename, argsStr string, sacrifice *implantpb.SacrificeProcess) (*implantpb.ExecuteBinary, error) {
		cmdline, err := shellquote.Split(argsStr)
		if err != nil {
			return nil, err
		}
		return builtin.NewExecutable(module, filename, cmdline, "x86", sacrifice)
	})

	// 构建 64 位二进制消息
	RegisterFunction("new_64_executable", func(module, filename, argsStr string, sacrifice *implantpb.SacrificeProcess) (*implantpb.ExecuteBinary, error) {
		cmdline, err := shellquote.Split(argsStr)
		if err != nil {
			return nil, err
		}
		return builtin.NewExecutable(module, filename, cmdline, "amd64", sacrifice)
	})

	// 构建新的二进制消息
	RegisterFunction("new_binary", func(module, filename string, args []string,
		output bool, timeout uint32, arch, process string,
		sacrifice *implantpb.SacrificeProcess) (*implantpb.ExecuteBinary, error) {
		return builtin.NewBinary(module, filename, args, output, timeout, arch, process, sacrifice)
	})

	// 构建 sacrifice 进程消息
	RegisterFunction("new_sacrifice", func(ppid int64, hidden, blockDll, disableETW bool, argue string) (*implantpb.SacrificeProcess, error) {
		return builtin.NewSacrificeProcessMessage(ppid, hidden, blockDll, disableETW, argue)
	})

	// 等待任务结果
	RegisterFunction("wait", func(task *clientpb.Task) (*clientpb.TaskContext, error) {
		return builtin.WaitResult(rpc, task)
	})

	// 获取任务结果
	RegisterFunction("get", func(task *clientpb.Task, index int32) (*clientpb.TaskContext, error) {
		return builtin.GetResult(rpc, task, index)
	})

	// 打印任务
	RegisterFunction("taskprint", func(task *clientpb.TaskContext) (*implantpb.Spite, error) {
		return builtin.PrintTask(task)
	})

	// 打印 assembly
	RegisterFunction("assemblyprint", func(task *clientpb.TaskContext) (string, error) {
		err := handler.AssertStatusAndResponse(task.GetSpite(), types.MsgAssemblyResponse)
		if err != nil {
			return "", err
		}
		s, _ := builtin.ParseAssembly(task.Spite)
		logs.Log.Console(s)
		return s, nil
	})

}

func RegisterGRPCFunc(rpc clientrpc.MaliceRPCClient) {
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
		RegisterInternalFunc(methodName, internalFunc, nil)
	}
}
