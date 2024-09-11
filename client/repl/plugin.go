package repl

import (
	"context"
	"errors"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core/intermediate"
	"github.com/chainreactors/malice-network/client/core/intermediate/builtin"
	"github.com/chainreactors/malice-network/client/core/plugin"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/handler"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/kballard/go-shellquote"
	"github.com/spf13/cobra"
	lua "github.com/yuin/gopher-lua"
	"google.golang.org/protobuf/proto"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

const (
	LuaScript = "lua"
	TCLScript = "tcl"
	CNAScript = "cna"
	GoPlugin  = "go"
)

var (
	ErrorAlreadyScriptName = errors.New("already exist script name")
)

func NewPlugins() *Plugins {
	plugins := &Plugins{
		Plugins: map[string]*Plugin{},
	}

	return plugins
}

type Plugins struct {
	Plugins map[string]*Plugin
}

func (plguins *Plugins) LoadPlugin(manifest *plugin.MalManiFest, con *Console) (*Plugin, error) {
	switch manifest.Type {
	case LuaScript:
		return plguins.LoadLuaScript(manifest, con)
	case TCLScript:
		// TODO
		return nil, fmt.Errorf("not impl")
	case GoPlugin:
		// TODO
		return nil, fmt.Errorf("not impl")
	default:
		return nil, fmt.Errorf("not found valid script type: %s", manifest.Type)
	}
}

func (plguins *Plugins) LoadLuaScript(manifest *plugin.MalManiFest, con *Console) (*Plugin, error) {
	// 检查脚本名称是否已存在
	if _, ok := plguins.Plugins[manifest.Name]; ok {
		return nil, ErrorAlreadyScriptName
	}

	// 创建并存储新的插件
	plug, err := NewPlugin(manifest, con)
	if err != nil {
		return nil, err
	}

	plguins.Plugins[manifest.Name] = plug

	// 将脚本添加到预加载模块中
	//L.PreloadModule(manifest.Name, func(L *lua.LState) int {
	//	if err := L.DoString(string(content)); err != nil {
	//		Log.Errorf("failed to preload Lua script: %s", err.Error())
	//		return 0
	//	}
	//	return 1
	//})

	return plug, nil
}

func NewLuaVM(con *Console) *lua.LState {
	vm := lua.NewState()
	vm.OpenLibs()

	intermediate.RegisterProtobufMessageType(vm)
	rpcType := reflect.TypeOf(con.Rpc)
	rpcValue := reflect.ValueOf(con.Rpc)

	for i := 0; i < rpcType.NumMethod(); i++ {
		method := rpcType.Method(i)
		methodName := method.Name

		// 获取方法的输入参数类型
		methodInputType := method.Type.In(2) // 第二个参数是 proto.Message 类型

		// 忽略流式方法
		methodReturnType := method.Type.Out(0)
		if methodReturnType.Kind() == reflect.Interface && methodReturnType.Name() == "ClientStream" {
			continue
		}

		// 将方法包装为 InternalFunc
		internalFunc := func(args ...interface{}) (interface{}, error) {
			var ctx context.Context
			if sess := con.GetInteractive(); sess == nil {
				ctx = context.Background()
			} else {
				ctx = sess.Context()
			}

			// 准备 proto.Message 参数
			var protoMsg reflect.Value
			if len(args) == 0 {
				if methodInputType == reflect.TypeOf(&clientpb.Empty{}) {
					protoMsg = reflect.ValueOf(&clientpb.Empty{})
				} else {
					protoMsg = reflect.Zero(methodInputType) // 创建一个零值的 proto.Message
				}
			} else {
				// 确保传入的第一个参数是 proto.Message
				msg, ok := args[0].(proto.Message)
				if !ok {
					return nil, fmt.Errorf("argument must be proto.Message")
				}
				protoMsg = reflect.ValueOf(msg)
			}

			// 准备调用方法的参数列表
			callArgs := []reflect.Value{
				reflect.ValueOf(ctx), // context.Context
				protoMsg,             // proto.Message
			}

			// 调用方法
			results := method.Func.Call(append([]reflect.Value{rpcValue}, callArgs...))

			// 处理返回值，假设返回值格式为 (*proto.Message, error)
			if !results[1].IsNil() {
				return results[0].Interface(), results[1].Interface().(error)
			}
			return results[0].Interface(), nil
		}

		sig := intermediate.GetInternalFuncSignature(method.Func)
		sig.ArgTypes = sig.ArgTypes[2:3]
		intermediate.RegisterInternalFunc(methodName, internalFunc, sig)
	}

	// 将InternalFunctions中的函数注册到lua
	for name, fn := range intermediate.InternalFunctions {
		vm.SetGlobal(name, vm.NewFunction(intermediate.WrapDynamicFuncForLua(fn)))
	}

	return vm
}

func NewPlugin(manifest *plugin.MalManiFest, con *Console) (*Plugin, error) {
	path := filepath.Join(assets.GetMalsDir(), manifest.Name)
	content, err := os.ReadFile(filepath.Join(path, manifest.EntryFile))
	if err != nil {
		return nil, err
	}

	plug := &Plugin{
		MalManiFest: manifest,
		Enable:      true,
		Content:     content,
		Path:        path,
	}

	err = plug.RegisterLuaBuiltInFunctions(con)
	if err != nil {
		return nil, err
	}

	plug.LuaVM = NewLuaVM(con)

	if err = plug.LuaVM.DoString(string(plug.Content)); err != nil {
		return nil, fmt.Errorf("failed to load Lua script: %w", err)
	}

	return plug, nil
}

type Plugin struct {
	*plugin.MalManiFest
	Enable  bool
	Content []byte
	Path    string
	LuaVM   *lua.LState
	CMDs    []*cobra.Command
}

func CmdExists(name string, cmd *cobra.Command) bool {
	for _, c := range cmd.Commands() {
		if name == c.Name() {
			return true
		}
	}
	return false
}

func (plugin *Plugin) registerLuaFunction(name string, fn interface{}) {
	wrappedFunc, sig := WrapInternalFunc(fn)
	intermediate.RegisterInternalFunc(name, wrappedFunc, sig)
}

func (plugin *Plugin) RegisterLuaBuiltInFunctions(con *Console) error {
	plugin.registerLuaFunction("active", func() (*clientpb.Session, error) {
		return con.GetInteractive().Session, nil
	})

	// 获取资源文件名
	plugin.registerLuaFunction("script_resource", func(filename string) (string, error) {
		return builtin.GetResourceFile(plugin.Name, filename)
	})

	// 读取资源文件内容
	plugin.registerLuaFunction("read_resource", func(filename string) (string, error) {
		return builtin.ReadResourceFile(plugin.Name, filename)
	})

	// 构建 x86 二进制消息
	plugin.registerLuaFunction("new_86_executable", func(module, filename, argsStr string, sacrifice *implantpb.SacrificeProcess) (*implantpb.ExecuteBinary, error) {
		cmdline, err := shellquote.Split(argsStr)
		if err != nil {
			return nil, err
		}
		return builtin.NewExecutable(module, filename, cmdline, "x86", sacrifice)
	})

	// 构建 64 位二进制消息
	plugin.registerLuaFunction("new_64_executable", func(module, filename, argsStr string, sacrifice *implantpb.SacrificeProcess) (*implantpb.ExecuteBinary, error) {
		cmdline, err := shellquote.Split(argsStr)
		if err != nil {
			return nil, err
		}
		return builtin.NewExecutable(module, filename, cmdline, "amd64", sacrifice)
	})

	// 构建新的二进制消息
	plugin.registerLuaFunction("new_binary", func(module, filename string, args []string,
		output bool, timeout int, arch, process string,
		sacrifice *implantpb.SacrificeProcess) (*implantpb.ExecuteBinary, error) {
		return builtin.NewBinary(module, filename, args, output, timeout, arch, process, sacrifice)
	})

	// 构建 sacrifice 进程消息
	plugin.registerLuaFunction("new_sacrifice", func(ppid int64, hidden, blockDll, disableETW bool, argue string) (*implantpb.SacrificeProcess, error) {
		return builtin.NewSacrificeProcessMessage(ppid, hidden, blockDll, disableETW, argue)
	})

	// 等待任务结果
	plugin.registerLuaFunction("wait", func(task *clientpb.Task) (*clientpb.TaskContext, error) {
		return builtin.WaitResult(con.Rpc, task)
	})

	// 获取任务结果
	plugin.registerLuaFunction("get", func(task *clientpb.Task, index int32) (*clientpb.TaskContext, error) {
		return builtin.GetResult(con.Rpc, task, index)
	})

	// 打印任务
	plugin.registerLuaFunction("taskprint", func(task *clientpb.TaskContext) (*implantpb.Spite, error) {
		return builtin.PrintTask(task)
	})

	// 打印 assembly
	plugin.registerLuaFunction("assemblyprint", func(task *clientpb.TaskContext) (string, error) {
		err := handler.AssertStatusAndResponse(task.GetSpite(), types.MsgAssemblyResponse)
		if err != nil {
			return "", err
		}
		s, _ := builtin.ParseAssembly(task.Spite)
		logs.Log.Console(s)
		return s, nil
	})

	// 校验 spite
	plugin.registerLuaFunction("ok", func(task *clientpb.TaskContext) (bool, error) {
		s, _ := builtin.ParseStatus(task.Spite)
		return s, nil
	})

	return nil
}

func (plugin *Plugin) ReverseRegisterLuaFunctions(con *Console, cmd *cobra.Command) error {
	vm := plugin.LuaVM
	globals := vm.Get(lua.GlobalsIndex).(*lua.LTable)
	//globals.ForEach(func(key lua.LValue, value lua.LValue) {
	//	if fn, ok := value.(*lua.LFunction); ok {
	//		funcName := key.String()
	//		if strings.HasPrefix(funcName, "command_") {
	//			// 注册到 RPCFunctions 中
	//			intermediate.InternalFunctions[funcName] = func(req ...interface{}) (interface{}, error) {
	//				vm.Push(fn) // 将函数推入栈
	//
	//				// 将所有参数推入栈
	//				for _, arg := range req {
	//					vm.Push(lua.LString(fmt.Sprintf("%v", arg)))
	//				}
	//
	//				// 调用函数
	//				if err := vm.PCall(len(req), lua.MultRet, nil); err != nil {
	//					return nil, fmt.Errorf("error calling Lua function %s: %w", funcName, err)
	//				}
	//
	//				// 获取返回值
	//				results := make([]interface{}, 0, vm.GetTop())
	//				for i := 1; i <= vm.GetTop(); i++ {
	//					results = append(results, vm.Get(i))
	//				}
	//
	//				// 如果有返回值，返回第一个值，否则返回nil
	//				if len(results) > 0 {
	//					return results[0], nil
	//				}
	//				return nil, nil
	//			}
	//			fmt.Printf("Registered Lua function to RPCFunctions: %s\n", funcName)
	//		}
	//	}
	//})

	globals.ForEach(func(key lua.LValue, value lua.LValue) {
		funcName := key.String()

		if fn, ok := value.(*lua.LFunction); ok {
			if !strings.HasPrefix(funcName, "command_") {
				return
			}
			cmdName := strings.TrimPrefix(funcName, "command_")
			if CmdExists(cmdName, cmd) {
				Log.Warnf("%s already exists, skipped\n", funcName)
				return
			}

			var paramNames []string
			for _, param := range fn.Proto.DbgLocals {
				paramNames = append(paramNames, param.Name)
			}

			// 创建新的 Cobra 命令
			malCmd := &cobra.Command{
				Use:   cmdName,
				Short: fmt.Sprintf("Lua function %s", funcName),
				RunE: func(cmd *cobra.Command, args []string) error {
					vm.Push(fn) // 将函数推入栈

					for _, paramName := range paramNames {
						if paramName == "args" {
							// 特殊处理 "args" 参数
							vm.Push(lua.LString(shellquote.Join(args...)))
						} else {
							// 获取 flag 的值并推入 Lua
							val, err := cmd.Flags().GetString(paramName)
							if err != nil {
								return fmt.Errorf("error getting flag %s: %w", paramName, err)
							}
							vm.Push(lua.LString(val))
						}
					}

					session := con.GetInteractive()
					go func() {
						if err := vm.PCall(len(args), lua.MultRet, nil); err != nil {
							session.Log.Errorf("error calling Lua function %s: %s", funcName, err.Error())
							return
						}

						resultCount := vm.GetTop()
						for i := 1; i <= resultCount; i++ {
							// 从栈顶依次弹出返回值
							result := vm.Get(-resultCount + i - 1)
							session.Log.Consolef("%v\n", result)
						}
						vm.Pop(resultCount)
					}()

					return nil
				},
				GroupID: consts.MalGroup,
			}

			for _, paramName := range paramNames {
				if paramName == "args" {
					continue
				}
				malCmd.Flags().String(paramName, "", fmt.Sprintf("parameter %s for %s", paramName, funcName))
			}

			plugin.CMDs = append(plugin.CMDs, malCmd)
			cmd.AddCommand(malCmd)
			Log.Debugf("Registered Command: %s\n", funcName)
		}
	})
	return nil
}
