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

type ImplantRpcFunc func(clientrpc.MaliceRPCClient, *Session, ...interface{}) (*clientpb.Task, error)
type implantCallback func(*clientpb.TaskContext) (interface{}, error)
type serverRpcFunc func(*Console, ...interface{}) (interface{}, error)

var (
	ErrorAlreadyScriptName = errors.New("already exist script name")
)

func WrapImplantFunc(rpc clientrpc.MaliceRPCClient, fun interface{}, callback implantCallback) intermediate.InternalFunc {
	// 使用反射将传入的函数适配为 ImplantRpcFunc
	wrappedFunc := WrapFunc(fun)

	return func(args ...interface{}) (interface{}, error) {
		// 调用适配后的 ImplantRpcFunc
		sess := args[0].(*Session)
		task, err := wrappedFunc(rpc, sess, args...)
		if err != nil {
			return nil, err
		}

		taskContext, err := rpc.WaitTaskFinish(context.Background(), task)
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

// WrapFunc 通过反射将任意符合参数签名的函数转换为 ImplantRpcFunc
func WrapFunc(funcToWrap interface{}) ImplantRpcFunc {
	return func(rpc clientrpc.MaliceRPCClient, sess *Session, params ...interface{}) (*clientpb.Task, error) {
		funcValue := reflect.ValueOf(funcToWrap)
		funcType := funcValue.Type()

		// 检查函数的参数数量是否匹配
		if funcType.NumIn() != len(params)+2 {
			return nil, fmt.Errorf("expected %d arguments, got %d", funcType.NumIn()-1, len(params))
		}

		// 构建参数切片
		in := make([]reflect.Value, len(params)+1)
		in[0] = reflect.ValueOf(rpc)
		in[1] = reflect.ValueOf(sess)
		for i, param := range params {
			if reflect.TypeOf(param) != funcType.In(i+1) {
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

func WrapServerFunc(con *Console, fun serverRpcFunc) intermediate.InternalFunc {
	return func(req ...interface{}) (interface{}, error) {
		resp, err := fun(con, req...)
		if err != nil {
			return nil, err
		}
		return resp, nil
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
			if ctx = con.ActiveTarget.Context(); ctx == nil {
				ctx = context.Background()
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
	// get resource filename
	plugin.registerLuaFunction("resource_file", func(args ...interface{}) (interface{}, error) {
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

func (plugin *Plugin) ReverseRegisterLuaFunctions(cmd *cobra.Command) error {
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
			malCmd := &cobra.Command{
				Use:   cmdName,
				Short: fmt.Sprintf("Lua function %s", funcName),
				RunE: func(cmd *cobra.Command, args []string) error {
					vm.Push(fn) // 将函数推入栈

					// 将所有参数推入栈
					for _, arg := range args {
						vm.Push(lua.LString(arg))
					}

					// 调用函数
					if err := vm.PCall(len(args), lua.MultRet, nil); err != nil {
						return fmt.Errorf("error calling Lua function %s: %w", funcName, err)
					}

					// 处理返回值
					for i := 1; i <= vm.GetTop(); i++ {
						fmt.Println(vm.Get(i))
					}

					return nil
				},
				GroupID: consts.MalGroup,
			}
			plugin.CMDs = append(plugin.CMDs, malCmd)
			cmd.AddCommand(malCmd)
			Log.Debugf("Registered Command: %s\n", funcName)
		}
	})
	return nil
}
