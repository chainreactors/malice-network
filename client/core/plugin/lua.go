package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kballard/go-shellquote"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	lua "github.com/yuin/gopher-lua"
	"github.com/yuin/gopher-lua/parse"
	"golang.org/x/exp/slices"

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

	ProtoPackage  = []string{"implantpb", "clientpb", "modulepb"}
	GlobalPlugins []*DefaultPlugin
)

type LuaVMWrapper struct {
	*lua.LState
	initialized  bool
	lastUsedTime time.Time
	lock         sync.Mutex
}

func NewLuaVMWrapper() *LuaVMWrapper {
	return &LuaVMWrapper{
		LState:      NewLuaVM(),
		initialized: false,
	}
}

func (w *LuaVMWrapper) Lock() {
	w.lock.Lock()
	w.lastUsedTime = time.Now()
}

func (w *LuaVMWrapper) Unlock() {
	w.lastUsedTime = time.Now()
	w.lock.Unlock()
}

type LuaVMPool struct {
	vms        []*LuaVMWrapper
	maxSize    int
	lock       sync.Mutex
	proto      *lua.FunctionProto
	initScript string
	plugName   string
}

func NewLuaVMPool(maxSize int, initScript string, plugName string) (*LuaVMPool, error) {
	pool := &LuaVMPool{
		maxSize:    maxSize,
		vms:        make([]*LuaVMWrapper, 0, maxSize),
		initScript: initScript,
		plugName:   plugName,
	}

	// 预编译脚本
	reader := strings.NewReader(initScript)
	chunk, err := parse.Parse(reader, "script")
	if err != nil {
		return nil, fmt.Errorf("parse script error: %v", err)
	}
	proto, err := lua.Compile(chunk, "script")
	if err != nil {
		return nil, fmt.Errorf("compile script error: %v", err)
	}
	pool.proto = proto

	return pool, nil
}

func (p *LuaVMPool) AcquireVM() (*LuaVMWrapper, error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	for _, wrapper := range p.vms {
		if wrapper.lock.TryLock() {
			return wrapper, nil
		}
	}

	if len(p.vms) < p.maxSize {
		wrapper := NewLuaVMWrapper()
		wrapper.Lock()
		p.vms = append(p.vms, wrapper)
		return wrapper, nil
	}

	logs.Log.Warnf("VM pool is full, waiting for available VM...")
	p.lock.Unlock()

	for {
		p.lock.Lock()
		for _, wrapper := range p.vms {
			if wrapper.lock.TryLock() {
				return wrapper, nil
			}
		}
		p.lock.Unlock()
		time.Sleep(100 * time.Millisecond)
	}
}

func (p *LuaVMPool) ReleaseVM(wrapper *LuaVMWrapper) {
	wrapper.Unlock()
}

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
	}

	return mal, nil
}

func (plug *LuaPlugin) Run() error {
	var err error
	plug.vmPool, err = NewLuaVMPool(10, string(plug.Content), plug.Name)
	if err != nil {
		return err
	}
	err = plug.registerLuaOnHooks()
	if err != nil {
		return err
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

	// 读取resource文件
	plug.registerLuaFunction(vm, "script_resource", func(filename string) (string, error) {
		return intermediate.GetResourceFile(plug.Name, filename)
	})

	plug.registerLuaFunction(vm, "global_resource", func(filename string) (string, error) {
		return intermediate.GetGlobalResourceFile(filename)
	})

	plug.registerLuaFunction(vm, "find_resource", func(sess *core.Session, base string, ext string) (string, error) {
		return intermediate.GetResourceFile(plug.Name, fmt.Sprintf("%s.%s.%s", base, consts.FormatArch(sess.Os.Arch), ext))
	})

	plug.registerLuaFunction(vm, "find_global_resource", func(sess *core.Session, base string, ext string) (string, error) {
		return intermediate.GetGlobalResourceFile(fmt.Sprintf("%s.%s.%s", base, consts.FormatArch(sess.Os.Arch), ext))
	})

	// 读取资源文件内容
	plug.registerLuaFunction(vm, "read_resource", func(filename string) (string, error) {
		resourcePath, _ := intermediate.GetResourceFile(plug.Name, filename)
		content, err := os.ReadFile(resourcePath)
		if err != nil {
			return "", err
		}
		return string(content), nil
	})

	plug.registerLuaFunction(vm, "read_global_resource", func(filename string) (string, error) {
		resourcePath, _ := intermediate.GetGlobalResourceFile(filename)
		content, err := os.ReadFile(resourcePath)
		if err != nil {
			return "", err
		}
		return string(content), nil
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

	plug.registerLuaFunction(vm, "opsec", func(name string, opsec int) (bool, error) {
		cmd := plug.CMDs.Find(name)
		if cmd.CMD == nil {
			return false, fmt.Errorf("command %s not found", name)
		}
		if cmd.CMD.Annotations == nil {
			cmd.CMD.Annotations = map[string]string{
				"opsec": strconv.Itoa(opsec),
			}
		} else {
			cmd.CMD.Annotations["opsec"] = strconv.Itoa(opsec)
		}
		return true, nil
	})

	plug.registerLuaFunction(vm, "command", func(name string, fn *lua.LFunction, short string, ttp string) (*cobra.Command, error) {
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
						logs.Log.Errorf("Failed to acquire VM: %v", err)
						return
					}
					defer plug.Release(wrapper)
					wrapper.Push(fn)

					for _, p := range params {
						switch p.Type {
						case LuaFlag:
							val, err := cmd.Flags().GetString(p.Name)
							if err != nil {
								logs.Log.Errorf("error getting flag %s: %s", p.Name, err.Error())
								return
							}
							wrapper.Push(lua.LString(val))
						case LuaArg:
							i, err := strconv.Atoi(p.Name)
							if err != nil {
								logs.Log.Errorf("error converting arg %s to int: %s", p.Name, err.Error())
								return
							}
							val := cmd.Flags().Arg(i)
							if val == "" {
								logs.Log.Warnf("arg %d is empty", i)
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
	})

	return nil
}

func (plug *LuaPlugin) registerLuaOnHook(name string, condition intermediate.EventCondition) {
	vm := plug.onHookVM

	if fn := vm.GetGlobal("on_" + name); fn != lua.LNil {
		plug.Events[condition] = func(event *clientpb.Event) (bool, error) {

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

func (plug *LuaPlugin) registerLuaFunction(vm *lua.LState, name string, fn interface{}) {
	wrappedFunc := mals.WrapInternalFunc(fn)
	wrappedFunc.Package = intermediate.BuiltinPackage
	wrappedFunc.Name = name
	wrappedFunc.NoCache = true
	vm.SetGlobal(name, vm.NewFunction(mals.WrapFuncForLua(wrappedFunc)))
}

func NewLuaVM() *lua.LState {
	vm := mals.NewLuaVM()
	mals.RegisterProtobufMessagesFromPackage(vm, "implantpb")
	mals.RegisterProtobufMessagesFromPackage(vm, "clientpb")
	mals.RegisterProtobufMessagesFromPackage(vm, "modulepb")
	vm.PreloadModule(intermediate.BeaconPackage, mals.PackageLoader(intermediate.InternalFunctions.Package(intermediate.BeaconPackage)))
	vm.PreloadModule(intermediate.RpcPackage, mals.PackageLoader(intermediate.InternalFunctions.Package(intermediate.RpcPackage)))
	for _, global := range GlobalPlugins {
		vm.PreloadModule(global.Name, mals.GlobalLoader(global.Name, global.Path, global.Content))
	}

	// 注册所有内置函数
	for name, fun := range intermediate.InternalFunctions.Package(intermediate.BuiltinPackage) {
		vm.SetGlobal(name, vm.NewFunction(mals.WrapFuncForLua(fun)))
	}
	return vm
}
