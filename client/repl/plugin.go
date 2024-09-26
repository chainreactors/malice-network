package repl

import (
	"errors"
	"fmt"
	"github.com/chainreactors/malice-network/client/core/intermediate"
	"github.com/chainreactors/malice-network/client/core/plugin"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/kballard/go-shellquote"
	"github.com/spf13/cobra"
	lua "github.com/yuin/gopher-lua"
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
	cmd := con.ImplantMenu()

	err := plug.RegisterLuaBuiltin(vm)
	if err != nil {
		return err
	}

	for name, fn := range intermediate.InternalFunctions {
		//fmt.Printf("Registering internal function: %s %v\n", name, fn.ArgTypes)
		vm.SetGlobal(name, vm.NewFunction(intermediate.WrapFuncForLua(fn)))
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
			if !strings.HasPrefix(funcName, "command_") {
				return
			}
			cmdName := strings.TrimPrefix(funcName, "command_")
			if CmdExists(cmdName, cmd) {
				con.Log.Warnf("%s already exists, skipped\n", funcName)
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
						switch paramName {
						case "cmdline":
							vm.Push(lua.LString(shellquote.Join(args...)))
						case "args":
							vm.Push(intermediate.ConvertGoValueToLua(vm, args))
						default:
							val, err := cmd.Flags().GetString(paramName)
							if err != nil {
								return fmt.Errorf("error getting flag %s: %w", paramName, err)
							}
							vm.Push(lua.LString(val))
						}
					}

					go func() {
						if err := vm.PCall(len(paramNames), lua.MultRet, nil); err != nil {
							con.Log.Errorf("error calling Lua function %s: %s", funcName, err.Error())
							return
						}

						resultCount := vm.GetTop()
						//for i := 1; i <= resultCount; i++ {
						//	// 从栈顶依次弹出返回值
						//	result := vm.Get(-resultCount + i - 1)
						//	con.Log.Consolef("%v\n", result)
						//}
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

			plug.CMDs = append(plug.CMDs, malCmd)
			con.Log.Debugf("Registered Command: %s\n", funcName)
		}
	})
	return nil
}

func (plug *Plugin) registerLuaFunction(name string, fn interface{}) {
	wrappedFunc := intermediate.WrapInternalFunc(fn)
	plug.LuaVM.SetGlobal(name, plug.LuaVM.NewFunction(intermediate.WrapFuncForLua(wrappedFunc)))
}
