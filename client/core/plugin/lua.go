package plugin

import (
	"fmt"
	"github.com/kballard/go-shellquote"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	lua "github.com/yuin/gopher-lua"
	"golang.org/x/exp/slices"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/mals"
)

var (
	ReservedARGS    = "args"
	ReservedCMDLINE = "cmdline"
	ReservedCMD     = "cmd"
	ReservedWords   = []string{ReservedCMDLINE, ReservedARGS, ReservedCMD}

	ProtoPackage = []string{"implantpb", "clientpb", "modulepb"}
)

const (
	LuaInternal = iota
	LuaFlag
	LuaArg
	LuaReverse
)

type LuaParam struct {
	Name string
	Type int
}

type LuaPlugin struct {
	*DefaultPlugin
	vmFns    map[string]lua.LGFunction
	vmPool   *LuaVMPool
	onHookVM *LuaVMWrapper
}

func NewLuaMalPlugin(manifest *MalManiFest) (*LuaPlugin, error) {
	plug, err := NewPlugin(manifest)
	if err != nil {
		return nil, err
	}

	mal := &LuaPlugin{
		DefaultPlugin: plug,
		vmFns:         make(map[string]lua.LGFunction),
	}

	return mal, nil
}

func (plug *LuaPlugin) Run() error {
	var err error
	plug.vmPool, err = NewLuaVMPool(10, string(plug.Content), plug.Name)
	if err != nil {
		return err
	}
	plug.registerLuaFunction()
	err = plug.registerLuaOnHooks()
	if err != nil {
		return err
	}
	return nil
}

func (plug *LuaPlugin) Destroy() error {
	if plug.vmPool != nil {
		plug.vmPool.Destroy()
	}
	return nil
}

func (plug *LuaPlugin) Acquire() (*LuaVMWrapper, error) {
	wrapper, err := plug.vmPool.AcquireVM()
	if err != nil {
		return nil, err
	}

	if !wrapper.initialized {
		// 初始化 VM
		if err := plug.initVM(wrapper.LState); err != nil {
			plug.vmPool.ReleaseVM(wrapper)
			return nil, err
		}
		wrapper.initialized = true
	}

	return wrapper, nil
}

func (plug *LuaPlugin) Release(wrapper *LuaVMWrapper) {
	plug.vmPool.ReleaseVM(wrapper)
}

func (plug *LuaPlugin) initVM(vm *lua.LState) error {
	err := plug.RegisterLuaBuiltin(vm)
	if err != nil {
		return err
	}
	// 执行预编译的脚本
	lfunc := vm.NewFunctionFromProto(plug.vmPool.proto)
	vm.Push(lfunc)
	if err = vm.PCall(0, lua.MultRet, nil); err != nil {
		return fmt.Errorf("execute compiled script error: %v", err)
	}

	return nil
}

func (plug *LuaPlugin) registerLuaOnHooks() error {
	var err error
	plug.onHookVM, err = plug.Acquire()
	if err != nil {
		return err
	}
	// 注册所有的钩子
	plug.registerLuaOnHook("beacon_checkin", intermediate.EventCondition{Type: consts.EventSession, Op: consts.CtrlSessionCheckin})
	plug.registerLuaOnHook("beacon_initial", intermediate.EventCondition{Type: consts.EventSession, Op: consts.CtrlSessionRegister})
	plug.registerLuaOnHook("beacon_error", intermediate.EventCondition{Type: consts.EventSession, Op: consts.CtrlSessionError})
	plug.registerLuaOnHook("beacon_indicator", intermediate.EventCondition{Type: consts.EventSession, Op: consts.CtrlSessionLog})
	//plug.registerLuaOnHook("beacon_initial_empty", intermediate.EventCondition{Type: consts.EventSession, Op: consts.CtrlSessionDNS})
	//plug.registerLuaOnHook("beacon_input", intermediate.EventCondition{Type: consts.EventInput})
	//plug.registerLuaOnHook("beacon_mode", intermediate.EventCondition{Type: consts.EventModeChange})
	plug.registerLuaOnHook("beacon_output", intermediate.EventCondition{Type: consts.EventTask})
	plug.registerLuaOnHook("beacon_output_alt", intermediate.EventCondition{Type: consts.EventSession, Op: consts.CtrlSessionLog})
	plug.registerLuaOnHook("beacon_output_jobs", intermediate.EventCondition{Type: consts.EventTask, Op: consts.CtrlTaskFinish})
	plug.registerLuaOnHook("beacon_output_ls", intermediate.EventCondition{Type: consts.EventTask, Op: consts.CtrlTaskFinish, MessageType: types.MsgLs.String()})
	plug.registerLuaOnHook("beacon_output_ps", intermediate.EventCondition{Type: consts.EventTask, Op: consts.CtrlTaskFinish, MessageType: types.MsgPs.String()})
	plug.registerLuaOnHook("beacon_tasked", intermediate.EventCondition{Type: consts.EventClient, Op: consts.CtrlTaskCallback})

	// 注册其他非 Beacon 特定事件
	//plug.registerLuaOnHook("disconnect", intermediate.EventCondition{Type: consts.EventDisconnect})
	plug.registerLuaOnHook("event_action", intermediate.EventCondition{Type: consts.EventBroadcast})
	plug.registerLuaOnHook("event_beacon_initial", intermediate.EventCondition{Type: consts.EventSession, Op: consts.CtrlSessionInit})
	plug.registerLuaOnHook("event_join", intermediate.EventCondition{Type: consts.EventJoin, Op: consts.CtrlClientJoin})
	plug.registerLuaOnHook("event_notify", intermediate.EventCondition{Type: consts.EventNotify})
	//plug.registerLuaOnHook("event_nouser", intermediate.EventCondition{Type: consts.EventNotify, Op: consts.CtrlClientLeft})
	//plug.registerLuaOnHook("event_private", intermediate.EventCondition{Type: consts.EventBroadcast, Op: consts.CtrlTaskCallback})
	plug.registerLuaOnHook("event_public", intermediate.EventCondition{Type: consts.EventBroadcast})
	plug.registerLuaOnHook("event_quit", intermediate.EventCondition{Type: consts.EventLeft, Op: consts.CtrlClientLeft})

	// 注册心跳事件
	plug.registerLuaOnHook("heartbeat_1s", intermediate.EventCondition{Type: consts.EventSession, Op: consts.CtrlHeartbeat1s})
	plug.registerLuaOnHook("heartbeat_5s", intermediate.EventCondition{Type: consts.EventSession, Op: consts.CtrlHeartbeat5s})
	plug.registerLuaOnHook("heartbeat_10s", intermediate.EventCondition{Type: consts.EventSession, Op: consts.CtrlHeartbeat10s})
	plug.registerLuaOnHook("heartbeat_15s", intermediate.EventCondition{Type: consts.EventSession, Op: consts.CtrlHeartbeat15s})
	plug.registerLuaOnHook("heartbeat_30s", intermediate.EventCondition{Type: consts.EventSession, Op: consts.CtrlHeartbeat30s})
	plug.registerLuaOnHook("heartbeat_1m", intermediate.EventCondition{Type: consts.EventSession, Op: consts.CtrlHeartbeat1m})
	plug.registerLuaOnHook("heartbeat_5m", intermediate.EventCondition{Type: consts.EventSession, Op: consts.CtrlHeartbeat5m})
	plug.registerLuaOnHook("heartbeat_10m", intermediate.EventCondition{Type: consts.EventSession, Op: consts.CtrlHeartbeat10m})
	plug.registerLuaOnHook("heartbeat_15m", intermediate.EventCondition{Type: consts.EventSession, Op: consts.CtrlHeartbeat15m})
	plug.registerLuaOnHook("heartbeat_20m", intermediate.EventCondition{Type: consts.EventSession, Op: consts.CtrlHeartbeat20m})
	plug.registerLuaOnHook("heartbeat_30m", intermediate.EventCondition{Type: consts.EventSession, Op: consts.CtrlHeartbeat30m})
	plug.registerLuaOnHook("heartbeat_60m", intermediate.EventCondition{Type: consts.EventSession, Op: consts.CtrlHeartbeat60m})
	return nil
}

func (plug *LuaPlugin) RegisterLuaBuiltin(vm *lua.LState) error {
	plugDir := filepath.Join(assets.GetMalsDir(), plug.Name)
	vm.SetGlobal("plugin_dir", lua.LString(plugDir))
	vm.SetGlobal("plugin_resource_dir", lua.LString(filepath.Join(plugDir, "resources")))
	vm.SetGlobal("plugin_name", lua.LString(plug.Name))
	vm.SetGlobal("temp_dir", lua.LString(assets.GetTempDir()))
	vm.SetGlobal("resource_dir", lua.LString(assets.GetResourceDir()))
	packageMod := vm.GetGlobal("package").(*lua.LTable)
	luaPath := lua.LuaPathDefault + ";" + filepath.Join(plugDir, "?.lua")
	vm.SetField(packageMod, "path", lua.LString(luaPath))

	for name, fn := range plug.vmFns {
		vm.SetGlobal(name, vm.NewFunction(fn))
	}
	return nil
}

func (plug *LuaPlugin) registerLuaFunction() {
	// 读取resource文件路径（文件系统版本）
	plug.registerFunction("script_resource", func(filename string) (string, error) {
		resourceFile := filepath.Join(assets.GetMalsDir(), plug.Name, "resources", filename)
		return resourceFile, nil
	}, nil)

	plug.registerFunction("global_resource", func(filename string) (string, error) {
		resourceFile := filepath.Join(assets.GetResourceDir(), filename)
		return resourceFile, nil
	}, nil)

	plug.registerFunction("find_resource", func(sess *core.Session, base string, ext string) (string, error) {
		filename := fmt.Sprintf("%s_%s_%s", base, consts.FormatArch(sess.Os.Arch), ext)
		resourceFile := filepath.Join(assets.GetMalsDir(), plug.Name, "resources", filename)
		return resourceFile, nil
	}, nil)

	plug.registerFunction("find_global_resource", func(sess *core.Session, base string, ext string) (string, error) {
		filename := fmt.Sprintf("%s_%s_%s", base, consts.FormatArch(sess.Os.Arch), ext)
		resourceFile := filepath.Join(assets.GetResourceDir(), filename)
		return resourceFile, nil
	}, nil)

	// 读取资源文件内容（文件系统版本，会被EmbedPlugin覆盖）
	plug.registerFunction("read_resource", func(filename string) (string, error) {
		resourcePath := filepath.Join(assets.GetMalsDir(), plug.Name, "resources", filename)
		content, err := os.ReadFile(resourcePath)
		if err != nil {
			return "", fmt.Errorf("resource file not found: %s", filename)
		}
		return string(content), nil
	}, nil)

	plug.registerFunction("read_global_resource", func(filename string) (string, error) {
		resourcePath := filepath.Join(assets.GetResourceDir(), filename)
		content, err := os.ReadFile(resourcePath)
		if err != nil {
			return "", fmt.Errorf("global resource file not found: %s", filename)
		}
		return string(content), nil
	}, nil)

	plug.registerFunction("help", func(name string, long string) (bool, error) {
		cmd := plug.CMDs.Find(name)
		cmd.Long = long
		return true, nil
	}, &mals.Helper{Group: intermediate.ClientGroup})

	plug.registerFunction("example", func(name string, example string) (bool, error) {
		cmd := plug.CMDs.Find(name)
		cmd.Example = example
		return true, nil
	}, &mals.Helper{Group: intermediate.ClientGroup})

	plug.registerFunction("opsec", func(name string, opsec int) (bool, error) {
		cmd := plug.CMDs.Find(name)
		if cmd.Command == nil {
			return false, fmt.Errorf("command %s not found", name)
		}
		if cmd.Command.Annotations == nil {
			cmd.Command.Annotations = map[string]string{
				"opsec": strconv.Itoa(opsec),
			}
		} else {
			cmd.Command.Annotations["opsec"] = strconv.Itoa(opsec)
		}
		return true, nil
	}, &mals.Helper{Group: intermediate.ClientGroup})

	plug.registerFunction("command", func(name string, fn *lua.LFunction, short string, ttp string) (*cobra.Command, error) {
		cmd := plug.CMDs.Find(name)

		var params []*LuaParam
		for _, param := range fn.Proto.DbgLocals {
			if strings.HasPrefix(param.Name, "flag_") {
				params = append(params, &LuaParam{
					Name: strings.TrimPrefix(param.Name, "flag_"),
					Type: LuaFlag,
				})
			} else if strings.HasPrefix(param.Name, "arg_") {
				params = append(params, &LuaParam{
					Name: strings.TrimPrefix(param.Name, "arg_"),
					Type: LuaArg,
				})
			} else if slices.Contains(ReservedWords, param.Name) {
				params = append(params, &LuaParam{
					Name: param.Name,
					Type: LuaReverse,
				})
			}
		}

		// 创建新的 Cobra 命令
		malCmd := &cobra.Command{
			Use:   cmd.Name,
			Short: short,
			Annotations: map[string]string{
				"ttp": ttp,
			},
			Run: func(cmd *cobra.Command, args []string) {
				go func() {
					wrapper, err := plug.Acquire()
					if err != nil {
						logs.Log.Errorf("Failed to acquire VM: %v\n", err)
						return
					}
					defer plug.Release(wrapper)
					wrapper.Push(fn)

					for _, p := range params {
						switch p.Type {
						case LuaFlag:
							val, err := cmd.Flags().GetString(p.Name)
							if err != nil {
								logs.Log.Errorf("error getting flag %s: %s\n", p.Name, err.Error())
								return
							}
							wrapper.Push(lua.LString(val))
						case LuaArg:
							i, err := strconv.Atoi(p.Name)
							if err != nil {
								logs.Log.Errorf("error converting arg %s to int: %s\n", p.Name, err.Error())
								return
							}
							val := cmd.Flags().Arg(i)
							if val == "" {
								logs.Log.Warnf("arg %d is empty\n", i)
							}
							wrapper.Push(lua.LString(val))
						case LuaReverse:
							switch p.Name {
							case ReservedCMDLINE:
								wrapper.Push(lua.LString(shellquote.Join(args...)))
							case ReservedARGS:
								wrapper.Push(mals.ConvertGoValueToLua(wrapper.LState, args))
							case ReservedCMD:
								wrapper.Push(mals.ConvertGoValueToLua(wrapper.LState, cmd))
							}
						}
					}

					var outFunc intermediate.BuiltinCallback
					if outFile, _ := cmd.Flags().GetString("output_file"); outFile == "" {
						outFunc = func(content interface{}) (interface{}, error) {
							logs.Log.Consolef("%v\n", content)
							return true, nil
						}
					} else {
						outFunc = func(content interface{}) (interface{}, error) {
							cont, ok := content.(string)
							if !ok {
								return false, fmt.Errorf("expect content tpye string, found %s", reflect.TypeOf(content).String())
							}
							err := os.WriteFile(outFile, []byte(cont), 0644)
							if err != nil {
								return false, err
							}
							return true, nil
						}
					}

					if err := wrapper.PCall(len(params), lua.MultRet, nil); err != nil {
						logs.Log.Errorf("error calling Lua %s:\n%s", fn.String(), err.Error())
						return
					}

					resultCount := wrapper.GetTop()
					for i := 1; i <= resultCount; i++ {
						// 从栈顶依次弹出返回值
						result := wrapper.Get(-resultCount + i - 1)
						_, err := outFunc(mals.ConvertLuaValueToGo(result))
						if err != nil {
							logs.Log.Errorf("error calling outFunc:\n%s", err.Error())
							return
						}
					}
					wrapper.Pop(resultCount)
				}()
			},
		}

		set := pflag.NewFlagSet("mal common args", pflag.ExitOnError)
		set.StringP("output_file", "f", "", "output file")
		set.BoolP("help", "h", false, "print help")
		set.VisitAll(func(flag *pflag.Flag) {
			flag.Annotations = map[string][]string{
				"group": {"Common Arguments"},
			}
		})
		malCmd.Flags().AddFlagSet(set)

		for _, p := range params {
			if p.Type == LuaFlag {
				malCmd.Flags().String(p.Name, "", p.Name)
			}
		}

		logs.Log.Debugf("Registered Command: %s\n", cmd.Name)
		plug.CMDs.SetCommand(name, malCmd)
		return malCmd, nil
	}, &mals.Helper{Group: intermediate.ClientGroup})

}

func (plug *LuaPlugin) registerLuaOnHook(name string, condition intermediate.EventCondition) {
	vm := plug.onHookVM

	if fn := vm.GetGlobal("on_" + name); fn != lua.LNil {
		plug.Events[condition] = func(event *clientpb.Event) (bool, error) {
			if vm.IsClosed() {
				return false, core.ErrLuaVMDead
			}
			fn := vm.GetGlobal("on_" + name)
			vm.Push(fn)
			vm.Push(mals.ConvertGoValueToLua(vm.LState, event))

			if err := vm.PCall(1, lua.MultRet, nil); err != nil {
				return false, fmt.Errorf("error calling Lua function %s: %w", name, err)
			}

			vm.Pop(vm.GetTop())
			return true, nil
		}
	}
}

func (plug *LuaPlugin) registerFunction(name string, fn interface{}, helper *mals.Helper) {
	wrappedFunc := mals.WrapInternalFunc(fn)
	wrappedFunc.Package = intermediate.BuiltinPackage
	wrappedFunc.Name = name
	wrappedFunc.NoCache = true
	wrappedFunc.Helper = helper

	if intermediate.InternalFunctions[name] == nil {
		intermediate.InternalFunctions[name] = &intermediate.InternalFunc{MalFunction: wrappedFunc}
	}

	plug.vmFns[name] = mals.WrapFuncForLua(wrappedFunc)
}
