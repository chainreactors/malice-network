package repl

import (
	"errors"
	"fmt"
	"github.com/chainreactors/malice-network/client/core/intermediate"
	"github.com/chainreactors/malice-network/client/core/intermediate/builtin"
	"github.com/chainreactors/malice-network/client/core/plugin"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
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

func (plguins *Plugins) LoadPlugin(manifest *plugin.MalManiFest, con *Console) (*Plugin, error) {
	switch manifest.Type {
	case plugin.LuaScript:
		return plguins.LoadLuaScript(manifest, con)
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

	plug.InitLua(con)
	plguins.Plugins[manifest.Name] = plug
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

	plug.registerLuaFunction("active", func() (*clientpb.Session, error) {
		return con.GetInteractive().Session, nil
	})

	//// 获取资源文件名
	plug.registerLuaFunction("script_resource", func(filename string) (string, error) {
		return builtin.GetResourceFile(plug.Name, filename)
	})

	// 读取资源文件内容
	plug.registerLuaFunction("read_resource", func(filename string) (string, error) {
		return builtin.ReadResourceFile(plug.Name, filename)
	})

	for name, fn := range intermediate.InternalFunctions {
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

			plug.CMDs = append(plug.CMDs, malCmd)
			cmd.AddCommand(malCmd)
			Log.Debugf("Registered Command: %s\n", funcName)
		}
	})
	return nil
}

func (plug *Plugin) registerLuaFunction(name string, fn interface{}) {
	wrappedFunc := intermediate.WrapInternalFunc(fn)
	plug.LuaVM.SetGlobal(name, plug.LuaVM.NewFunction(intermediate.WrapFuncForLua(wrappedFunc)))
}
