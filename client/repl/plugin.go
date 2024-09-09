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
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
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
)

type implantFunc func(rpc clientrpc.MaliceRPCClient, sess *Session, params ...interface{}) (*clientpb.Task, error)
type ImplantCallback func(*clientpb.TaskContext) (interface{}, error)

var (
	ErrorAlreadyScriptName = errors.New("already exist script name")
)

func wrapImplantFunc(fun interface{}) implantFunc {
	return func(rpc clientrpc.MaliceRPCClient, sess *Session, params ...interface{}) (*clientpb.Task, error) {
		funcValue := reflect.ValueOf(fun)
		funcType := funcValue.Type()

		// 检查函数的参数数量是否匹配
		if funcType.NumIn() != len(params)+2 {
			return nil, fmt.Errorf("expected %d arguments, got %d", funcType.NumIn()-1, len(params))
		}

		// 构建参数切片
		in := make([]reflect.Value, len(params)+2)
		in[0] = reflect.ValueOf(rpc)
		in[1] = reflect.ValueOf(sess)
		for i, param := range params {
			if reflect.TypeOf(param) != funcType.In(i+2) {
				return nil, fmt.Errorf("argument %d should be %v, got %v", i+1, funcType.In(i+1), reflect.TypeOf(param))
			}
			in[i+2] = reflect.ValueOf(param)
		}

		// 调用函数并返回结果
		results := funcValue.Call(in)

		// 处理返回值并转换为 (*clientpb.Task, error)
		task, _ := results[0].Interface().(*clientpb.Task)
		var err error
		if results[1].Interface() != nil {
			err = results[1].Interface().(error)
		}

		return task, err
	}
}

func WrapImplantFunc(con *Console, fun interface{}, callback ImplantCallback) intermediate.InternalFunc {
	wrappedFunc := wrapImplantFunc(fun)

	return func(args ...interface{}) (interface{}, error) {
		task, err := wrappedFunc(con.Rpc, con.GetInteractive(), args...)
		if err != nil {
			return nil, err
		}

		taskContext, err := con.Rpc.WaitTaskFinish(context.Background(), task)
		if err != nil {
			return nil, err
		}

		if callback != nil {
			return callback(taskContext)
		} else {
			return taskContext, nil
		}
	}
}

func WrapActiveFunc(con *Console, fun interface{}, callback ImplantCallback) intermediate.InternalFunc {
	wrappedFunc := wrapImplantFunc(fun)

	return func(args ...interface{}) (interface{}, error) {
		var sess *Session
		if len(args) == 0 {
			return nil, fmt.Errorf("implant func first args must be session")
		} else {
			var ok bool
			sess, ok = args[0].(*Session)
			if !ok {
				return nil, fmt.Errorf("implant func first args must be session")
			}
			args = args[1:]
		}

		task, err := wrappedFunc(con.Rpc, sess, args...)
		if err != nil {
			return nil, err
		}

		taskContext, err := con.Rpc.WaitTaskFinish(context.Background(), task)
		if err != nil {
			return nil, err
		}

		if callback != nil {
			return callback(taskContext)
		} else {
			return taskContext, nil
		}
	}
}

func WrapServerFunc(con *Console, fun interface{}) intermediate.InternalFunc {
	wrappedFunc := func(con *Console, params ...interface{}) (interface{}, error) {
		funcValue := reflect.ValueOf(fun)
		funcType := funcValue.Type()

		// 检查函数的参数数量是否匹配
		if funcType.NumIn() != len(params)+1 {
			return nil, fmt.Errorf("expected %d arguments, got %d", funcType.NumIn()-1, len(params))
		}

		// 构建参数切片
		in := make([]reflect.Value, len(params)+1)
		in[0] = reflect.ValueOf(con)
		for i, param := range params {
			if reflect.TypeOf(param) != funcType.In(i+1) {
				return nil, fmt.Errorf("argument %d should be %v, got %v", i+1, funcType.In(i+1), reflect.TypeOf(param))
			}
			in[i+1] = reflect.ValueOf(param)
		}

		// 调用函数并返回结果
		results := funcValue.Call(in)

		// 假设函数有两个返回值，第一个是返回值，第二个是错误
		var err error
		if len(results) == 2 && results[1].Interface() != nil {
			err = results[1].Interface().(error)
		}

		return results[0].Interface(), err
	}

	return func(args ...interface{}) (interface{}, error) {
		return wrappedFunc(con, args...)
	}
}

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

		luaFunc := intermediate.WrapDynamicFuncForLua(internalFunc)

		// 在 Lua 中注册该方法
		vm.SetGlobal(methodName, vm.NewFunction(luaFunc))
	}

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
	plug.LuaVM = NewLuaVM(con)

	err = plug.RegisterLuaBuiltInFunctions(con)
	if err != nil {
		return nil, err
	}

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

func (plugin *Plugin) registerLuaFunction(name string, fn intermediate.InternalFunc, expectedArgs []reflect.Type) {
	plugin.LuaVM.SetGlobal(name, plugin.LuaVM.NewFunction(intermediate.WrapFuncForLua(fn, expectedArgs)))
}

func (plugin *Plugin) RegisterLuaBuiltInFunctions(con *Console) error {
	plugin.registerLuaFunction("active", func(args ...interface{}) (interface{}, error) {
		return con.GetInteractive(), nil
	}, []reflect.Type{})

	// get resource filename
	plugin.registerLuaFunction("script_resource", func(args ...interface{}) (interface{}, error) {
		filename := args[0].(string)
		return builtin.GetResourceFile(plugin.Name, filename)
	}, []reflect.Type{reflect.TypeOf("")})

	// read resource file content
	plugin.registerLuaFunction("read_resource", func(args ...interface{}) (interface{}, error) {
		filename := args[0].(string)
		return builtin.ReadResourceFile(plugin.Name, filename)
	}, []reflect.Type{reflect.TypeOf("")})

	// build binary message
	plugin.registerLuaFunction("new_binary", func(args ...interface{}) (interface{}, error) {
		module := args[0].(string)
		filename := args[1].(string)
		argsStr := args[2].(string)
		sacrifice := args[3].(*implantpb.SacrificeProcess)
		return builtin.NewBinaryMessage(plugin.Name, module, filename, argsStr, sacrifice)
	}, []reflect.Type{reflect.TypeOf(""), reflect.TypeOf(""), reflect.TypeOf(""), reflect.TypeOf(&implantpb.SacrificeProcess{})})

	// build sacrifice process message
	plugin.registerLuaFunction("new_sacrifice", func(args ...interface{}) (interface{}, error) {
		processName := args[0].(string)
		ppid := args[1].(int64)
		blockDll := args[2].(bool)
		argue := args[3].(string)
		argsStr := args[4].(string)
		return builtin.NewSacrificeProcessMessage(processName, ppid, blockDll, argue, argsStr)
	}, []reflect.Type{reflect.TypeOf(""), reflect.TypeOf(int64(0)), reflect.TypeOf(true), reflect.TypeOf(""), reflect.TypeOf("")})

	plugin.registerLuaFunction("wait", func(args ...interface{}) (interface{}, error) {
		task := args[0].(*clientpb.Task)
		return builtin.WaitResult(con.Rpc, task)
	}, []reflect.Type{reflect.TypeOf(&clientpb.Task{})})

	plugin.registerLuaFunction("get", func(args ...interface{}) (interface{}, error) {
		task := args[0].(*clientpb.Task)
		index := args[1].(int32)
		return builtin.GetResult(con.Rpc, task, index)
	}, []reflect.Type{reflect.TypeOf(&clientpb.Task{}), reflect.TypeOf(int32(0))})

	plugin.registerLuaFunction("taskprint", func(args ...interface{}) (interface{}, error) {
		task := args[0].(*clientpb.TaskContext)
		return builtin.PrintTask(task)
	}, []reflect.Type{reflect.TypeOf(&clientpb.TaskContext{})})

	plugin.registerLuaFunction("assemblyprint", func(args ...interface{}) (interface{}, error) {
		task := args[0].(*clientpb.TaskContext)
		err := handler.AssertStatusAndResponse(task.GetSpite(), types.MsgAssemblyResponse)
		if err != nil {
			return nil, err
		}
		s, _ := builtin.ParseAssembly(task.Spite)
		logs.Log.Console(s)
		return s, nil
	}, []reflect.Type{reflect.TypeOf(&clientpb.TaskContext{})})

	// lua:
	// ok(spite) -> bool
	plugin.registerLuaFunction("ok", func(args ...interface{}) (interface{}, error) {
		task := args[0].(*clientpb.TaskContext)
		s, _ := builtin.ParseStatus(task.Spite)
		return s, nil
	}, []reflect.Type{reflect.TypeOf(&clientpb.TaskContext{})})
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
