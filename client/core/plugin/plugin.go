package plugin

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/core/intermediate"
	"github.com/kballard/go-shellquote"
	"github.com/spf13/cobra"
	lua "github.com/yuin/gopher-lua"
	"os"
	"path/filepath"
	"slices"
)

const (
	LuaScript = "lua"
	TCLScript = "tcl"
	CNAScript = "cna"
	GoPlugin  = "go"
)

func NewPlugin(manifest *MalManiFest) (*Plugin, error) {
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
		CMDs:        make(Commands),
	}

	return plug, nil
}

type Plugin struct {
	*MalManiFest
	Enable  bool
	Content []byte
	Path    string
	CMDs    Commands
}

func (plug *Plugin) RegisterLuaBuiltin(vm *lua.LState) error {
	plugDir := filepath.Join(assets.GetMalsDir(), plug.Name)
	vm.SetGlobal("plugin_dir", lua.LString(plugDir))
	vm.SetGlobal("plugin_resource_dir", lua.LString(filepath.Join(plugDir, "resources")))
	vm.SetGlobal("plugin_name", lua.LString(plug.Name))
	packageMod := vm.GetGlobal("package").(*lua.LTable)
	luaPath := lua.LuaPathDefault + ";" + plugDir + "\\?.lua"
	vm.SetField(packageMod, "path", lua.LString(luaPath))
	// 读取resource文件
	plug.registerLuaFunction(vm, "script_resource", func(filename string) (string, error) {
		return intermediate.GetResourceFile(plug.Name, filename)
	})

	plug.registerLuaFunction(vm, "find_resource", func(sess *core.Session, base string, ext string) (string, error) {
		return intermediate.FindResourceFile(plug.Name, base, sess.Os.Arch, ext)
	})

	// 读取资源文件内容
	plug.registerLuaFunction(vm, "read_resource", func(filename string) (string, error) {
		return intermediate.ReadResourceFile(plug.Name, filename)
	})

	plug.registerLuaFunction(vm, "help", func(name string, long string) (bool, error) {
		cmd := plug.CMDs.Find(name)
		cmd.Long = long
		return true, nil
	})

	plug.registerLuaFunction(vm, "example", func(name string, example string) (bool, error) {
		cmd := plug.CMDs.Find(name)
		cmd.Example = example
		return true, nil
	})

	plug.registerLuaFunction(vm, "command", func(name string, fn *lua.LFunction, short string) (bool, error) {
		cmd := plug.CMDs.Find(name)

		var paramNames []string
		for _, param := range fn.Proto.DbgLocals {
			paramNames = append(paramNames, param.Name)
		}

		// 创建新的 Cobra 命令
		malCmd := &cobra.Command{
			Use:   cmd.Name,
			Short: short,
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
						logs.Log.Console(content)
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
						logs.Log.Errorf("error calling Lua %s:\n%s", fn.String(), err.Error())
						return
					}

					resultCount := vm.GetTop()
					for i := 1; i <= resultCount; i++ {
						// 从栈顶依次弹出返回值
						result := vm.Get(-resultCount + i - 1)
						_, err := outFunc(result.String())
						if err != nil {
							logs.Log.Errorf("error calling outFunc:\n%s", err.Error())
							return
						}
					}
					vm.Pop(resultCount)
				}()

				return nil
			},
		}

		malCmd.Flags().StringP("file", "f", "", "output file")
		for _, paramName := range paramNames {
			if slices.Contains(ReservedWords, paramName) {
				continue
			}
			malCmd.Flags().String(paramName, "", paramName)
		}
		logs.Log.Debugf("Registered Command: %s\n", cmd.Name)
		plug.CMDs.SetCommand(name, malCmd)
		return true, nil
	})

	return nil
}

func (plug *Plugin) registerLuaFunction(vm *lua.LState, name string, fn interface{}) {
	wrappedFunc := intermediate.WrapInternalFunc(fn)
	wrappedFunc.Package = intermediate.BuiltinPackage
	wrappedFunc.Name = name
	vm.SetGlobal(name, vm.NewFunction(intermediate.WrapFuncForLua(wrappedFunc)))
}
