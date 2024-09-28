package repl

import (
	"context"
	"errors"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/core/intermediate"
	"github.com/chainreactors/malice-network/client/core/plugin"
	"github.com/chainreactors/malice-network/helper/utils/handler"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/chainreactors/tui"
	"github.com/kballard/go-shellquote"
	"github.com/spf13/cobra"
	lua "github.com/yuin/gopher-lua"
	"os"
	"reflect"
	"slices"
	"strings"
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

func (plugins *Plugins) LoadPlugin(manifest *plugin.MalManiFest, con *Console) (*Plugin, error) {
	switch manifest.Type {
	case plugin.LuaScript:
		return plugins.LoadLuaScript(manifest, con)
	case plugin.TCLScript:
		// TODO
		return nil, fmt.Errorf("not impl")
	case plugin.GoPlugin:
		// TODO
		return nil, fmt.Errorf("not impl")
	default:
		return nil, fmt.Errorf("not found valid script type: %s", manifest.Type)
	}
}

func (plugins *Plugins) LoadLuaScript(manifest *plugin.MalManiFest, con *Console) (*Plugin, error) {
	// 检查脚本名称是否已存在
	if _, ok := plugins.Plugins[manifest.Name]; ok {
		return nil, ErrorAlreadyScriptName
	}

	// 创建并存储新的插件
	plug, err := NewPlugin(manifest, con)
	if err != nil {
		return nil, err
	}

	err = plug.InitLua(con)
	if err != nil {
		return nil, err
	}
	plugins.Plugins[manifest.Name] = plug
	// 全局模块
	//L.PreloadModule(manifest.Name, func(L *lua.LState) int {
	//	if err := L.DoString(string(content)); err != nil {
	//		Log.Errorf("failed to preload Lua script: %s", err.Error())
	//		return 0
	//	}
	//	return 1
	//})

	return plug, nil
}

func NewPlugin(manifest *plugin.MalManiFest, con *Console) (*Plugin, error) {
	plug, err := plugin.NewPlugin(manifest)
	if err != nil {
		return nil, err
	}

	return &Plugin{
		Plugin: plug,
	}, nil
}

type Plugin struct {
	*plugin.Plugin
	LuaVM *lua.LState
	CMDs  []*cobra.Command
}

func (plug *Plugin) InitLua(con *Console) error {
	vm := plugin.NewLuaVM()
	plug.LuaVM = vm
	plugCmd := &cobra.Command{
		Use:   plug.Name,
		Short: fmt.Sprintf("%s cmds", plug.Name),
	}
	err := plug.RegisterLuaBuiltin(vm)
	if err != nil {
		return err
	}

	if err := vm.DoString(string(plug.Content)); err != nil {
		return fmt.Errorf("failed to load Lua script: %w", err)
	}

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
			cmd := plugCmd
			if !strings.HasPrefix(funcName, "command_") {
				return
			}
			var cmdName string
			cmdNames := strings.Split(funcName[8:], "__")
			if len(cmdNames) == 1 {
				cmdName = cmdNames[0]
			} else {
				for i := 0; i < len(cmdNames)-1; i++ {
					subName := cmdNames[i]
					if c := GetCmd(cmd, subName); c != nil {
						cmd = c
					} else {
						subCmd := &cobra.Command{
							Use:   subName,
							Short: fmt.Sprintf("%s group", subName),
						}
						cmd.AddCommand(subCmd)
						cmd = subCmd
					}
				}
				cmdName = cmdNames[len(cmdNames)-1]
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
						switch paramName {
						case "cmdline":
							vm.Push(lua.LString(shellquote.Join(args...)))
						case "args":
							vm.Push(intermediate.ConvertGoValueToLua(vm, args))
						default:
							val, err := cmd.Flags().GetString(paramName)
							if err != nil {
								logs.Log.Errorf("error getting flag %s: %s", paramName, err.Error())
								return err
							}
							vm.Push(lua.LString(val))
						}
					}

					var outFunc intermediate.BuiltinCallback
					if outFile, _ := cmd.Flags().GetString("file"); outFile == "" {
						outFunc = func(content string) (bool, error) {
							con.Log.Console(content)
							return true, nil
						}
					} else {
						outFunc = func(content string) (bool, error) {
							err := os.WriteFile(outFile, []byte(content), 0644)
							if err != nil {
								return false, err
							}
							return true, nil
						}
					}
					go func() {
						if err := vm.PCall(len(paramNames), lua.MultRet, nil); err != nil {
							con.Log.Errorf("error calling Lua function %s:\n%s", funcName, err.Error())
							return
						}

						resultCount := vm.GetTop()
						for i := 1; i <= resultCount; i++ {
							// 从栈顶依次弹出返回值
							result := vm.Get(-resultCount + i - 1)
							_, err := outFunc(result.String())
							if err != nil {
								con.Log.Errorf("error calling outFunc:\n%s", err.Error())
								return
							}
						}
						vm.Pop(resultCount)
					}()

					return nil
				},
			}

			malCmd.Flags().StringP("file", "f", "", "enable, output")
			for _, paramName := range paramNames {
				if slices.Contains(plugin.ReservedWords, paramName) {
					continue
				}
				malCmd.Flags().String(paramName, "", fmt.Sprintf("parameter %s for %s", paramName, funcName))
			}
			cmd.AddCommand(malCmd)
			con.Log.Debugf("Registered Command: %s\n", funcName)
		}
	})
	plug.CMDs = plugCmd.Commands()
	return nil
}

type implantFunc func(rpc clientrpc.MaliceRPCClient, sess *core.Session, params ...interface{}) (*clientpb.Task, error)
type ImplantPluginCallback func(content *clientpb.TaskContext) (interface{}, error)

func WrapImplantCallback(callback ImplantPluginCallback) intermediate.ImplantCallback {
	return func(content *clientpb.TaskContext) (string, error) {
		res, err := callback(content)
		if err != nil {
			return "", err
		}
		switch res.(type) {
		case string:
			output := res.(string)
			if output == "" {
				return "not output", nil
			} else {
				return output, nil
			}
		case bool:
			if res.(bool) {
				return fmt.Sprintf("%s ok", content.Task.Type), nil
			} else {
				return fmt.Sprintf("%s failed", content.Task.Type), nil
			}
		default:
			return fmt.Sprintf("%v", res), nil
		}
	}
}

func wrapImplantFunc(fun interface{}) implantFunc {
	return func(rpc clientrpc.MaliceRPCClient, sess *core.Session, params ...interface{}) (*clientpb.Task, error) {
		funcValue := reflect.ValueOf(fun)
		funcType := funcValue.Type()

		// debug
		//fmt.Println(runtime.FuncForPC(reflect.ValueOf(fun).Pointer()).Name())
		//for i := 0; i < funcType.NumIn(); i++ {
		//	fmt.Println(funcType.In(i).String())
		//}
		//fmt.Printf("%v\n", params)

		// 检查函数的参数数量是否匹配, rpc与session是强制要求的默认值, 自动+2
		if funcType.NumIn() != len(params)+2 {
			return nil, fmt.Errorf("expected %d arguments, got %d", funcType.NumIn(), len(params))
		}

		in := make([]reflect.Value, len(params)+2)
		in[0] = reflect.ValueOf(rpc)
		in[1] = reflect.ValueOf(sess)
		for i, param := range params {
			expectedType := funcType.In(i + 2)
			paramType := reflect.TypeOf(param)
			if paramType.Kind() == reflect.Int64 {
				param = intermediate.ConvertNumericType(param.(int64), expectedType.Kind())
			}
			if reflect.TypeOf(param) != expectedType {
				return nil, fmt.Errorf("argument %d should be %v, got %v", i+1, funcType.In(i+3), reflect.TypeOf(param))
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

func WrapImplantFunc(con *Console, fun interface{}, callback ImplantPluginCallback) *intermediate.InternalFunc {
	wrappedFunc := wrapImplantFunc(fun)

	interFunc := intermediate.GetInternalFuncSignature(fun)
	interFunc.ArgTypes = interFunc.ArgTypes[2:]
	interFunc.Func = func(args ...interface{}) (interface{}, error) {
		var sess *core.Session
		if len(args) == 0 {
			return nil, fmt.Errorf("implant func first args must be session")
		} else {
			var ok bool
			sess, ok = args[0].(*core.Session)
			if !ok {
				return nil, fmt.Errorf("implant func first args must be session")
			}
			args = args[1:]
		}

		task, err := wrappedFunc(con.Rpc, sess, args...)
		if err != nil {
			return nil, err
		}
		sess.Console(task, fmt.Sprintf("args %v", args))
		content, err := con.Rpc.WaitTaskFinish(context.Background(), task)
		if err != nil {
			return nil, err
		}

		tui.Down(0)
		err = handler.HandleMaleficError(content.Spite)
		if err != nil {
			con.Log.Errorf(err.Error())
			return nil, err
		}

		if callback != nil {
			return callback(content)
		} else {
			return content, nil
		}
	}
	return interFunc
}

func WrapServerFunc(con *Console, fun interface{}) *intermediate.InternalFunc {
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
	internalFunc := intermediate.GetInternalFuncSignature(fun)
	internalFunc.ArgTypes = internalFunc.ArgTypes[1:]
	internalFunc.Func = func(args ...interface{}) (interface{}, error) {
		return wrappedFunc(con, args...)
	}

	return internalFunc
}
