package plugin

import (
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/core/intermediate"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/cjoudrey/gluahttp"
	"github.com/kballard/go-shellquote"
	"github.com/spf13/cobra"
	luacrypto "github.com/tengattack/gluacrypto/crypto"
	"github.com/vadv/gopher-lua-libs/argparse"
	"github.com/vadv/gopher-lua-libs/base64"
	"github.com/vadv/gopher-lua-libs/cmd"
	"github.com/vadv/gopher-lua-libs/db"
	luafilepath "github.com/vadv/gopher-lua-libs/filepath"
	"github.com/vadv/gopher-lua-libs/goos"
	"github.com/vadv/gopher-lua-libs/humanize"
	"github.com/vadv/gopher-lua-libs/inspect"
	"github.com/vadv/gopher-lua-libs/ioutil"
	"github.com/vadv/gopher-lua-libs/json"
	"github.com/vadv/gopher-lua-libs/log"
	"github.com/vadv/gopher-lua-libs/plugin"
	"github.com/vadv/gopher-lua-libs/regexp"
	"github.com/vadv/gopher-lua-libs/shellescape"
	"github.com/vadv/gopher-lua-libs/stats"
	"github.com/vadv/gopher-lua-libs/storage"
	luastrings "github.com/vadv/gopher-lua-libs/strings"
	"github.com/vadv/gopher-lua-libs/tcp"
	"github.com/vadv/gopher-lua-libs/template"
	"github.com/vadv/gopher-lua-libs/time"
	luayaml "github.com/vadv/gopher-lua-libs/yaml"
	lua "github.com/yuin/gopher-lua"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"
)

var (
	ReservedARGS    = "args"
	ReservedCMDLINE = "cmdline"
	ReservedWords   = []string{ReservedCMDLINE, ReservedARGS}

	LuaPackages = map[string]*lua.LTable{}

	ProtoPackage  = []string{"implantpb", "clientpb", "modulepb"}
	GlobalPlugins []*DefaultPlugin
)

type LuaPlugin struct {
	*DefaultPlugin
	vm   *lua.LState
	lock *sync.Mutex
}

func NewLuaMalPlugin(manifest *MalManiFest) (*LuaPlugin, error) {
	plug, err := NewPlugin(manifest)
	if err != nil {
		return nil, err
	}
	mal := &LuaPlugin{
		DefaultPlugin: plug,
		vm:            NewLuaVM(),
		lock:          &sync.Mutex{},
	}
	err = mal.RegisterLuaBuiltin()
	if err != nil {
		return nil, err
	}

	return mal, nil
}

func (plug *LuaPlugin) Run() error {
	if err := plug.vm.DoString(string(plug.Content)); err != nil {
		return fmt.Errorf("failed to load Lua script: %w", err)
	}
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

func (plug *LuaPlugin) RegisterLuaBuiltin() error {
	vm := plug.vm
	plugDir := filepath.Join(assets.GetMalsDir(), plug.Name)
	vm.SetGlobal("plugin_dir", lua.LString(plugDir))
	vm.SetGlobal("plugin_resource_dir", lua.LString(filepath.Join(plugDir, "resources")))
	vm.SetGlobal("plugin_name", lua.LString(plug.Name))
	vm.SetGlobal("temp_dir", lua.LString(assets.GetTempDir()))
	vm.SetGlobal("resource_dir", lua.LString(assets.GetResourceDir()))
	packageMod := vm.GetGlobal("package").(*lua.LTable)
	luaPath := lua.LuaPathDefault + ";" + plugDir + "\\?.lua"
	vm.SetField(packageMod, "path", lua.LString(luaPath))
	// 读取resource文件
	plug.registerLuaFunction("script_resource", func(filename string) (string, error) {
		return intermediate.GetResourceFile(plug.Name, filename)
	})

	plug.registerLuaFunction("global_resource", func(filename string) (string, error) {
		return intermediate.GetGlobalResourceFile(filename)
	})

	plug.registerLuaFunction("find_resource", func(sess *core.Session, base string, ext string) (string, error) {
		return intermediate.GetResourceFile(plug.Name, fmt.Sprintf("%s.%s.%s", base, consts.FormatArch(sess.Os.Arch), ext))
	})

	plug.registerLuaFunction("find_global_resource", func(sess *core.Session, base string, ext string) (string, error) {
		return intermediate.GetGlobalResourceFile(fmt.Sprintf("%s.%s.%s", base, consts.FormatArch(sess.Os.Arch), ext))
	})

	// 读取资源文件内容
	plug.registerLuaFunction("read_resource", func(filename string) (string, error) {
		resourcePath, _ := intermediate.GetResourceFile(plug.Name, filename)
		content, err := os.ReadFile(resourcePath)
		if err != nil {
			return "", err
		}
		return string(content), nil
	})

	plug.registerLuaFunction("read_global_resource", func(filename string) (string, error) {
		resourcePath, _ := intermediate.GetGlobalResourceFile(filename)
		content, err := os.ReadFile(resourcePath)
		if err != nil {
			return "", err
		}
		return string(content), nil
	})

	plug.registerLuaFunction("help", func(name string, long string) (bool, error) {
		cmd := plug.CMDs.Find(name)
		cmd.Long = long
		return true, nil
	})

	plug.registerLuaFunction("example", func(name string, example string) (bool, error) {
		cmd := plug.CMDs.Find(name)
		cmd.Example = example
		return true, nil
	})

	plug.registerLuaFunction("opsec", func(name string, opsec int) (bool, error) {
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

	plug.registerLuaFunction("command", func(name string, fn *lua.LFunction, short string, ttp string) (bool, error) {
		cmd := plug.CMDs.Find(name)

		var paramNames []string
		for _, param := range fn.Proto.DbgLocals {
			if !strings.HasPrefix(param.Name, "flag_") && param.Name != "args" {
				continue
			}
			paramNames = append(paramNames, param.Name)
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
					plug.lock.Lock()
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
								return
							}
							vm.Push(lua.LString(val))
						}
					}

					var outFunc intermediate.BuiltinCallback
					if outFile, _ := cmd.Flags().GetString("file"); outFile == "" {
						outFunc = func(content interface{}) (bool, error) {
							logs.Log.Consolef("%v", content)
							return true, nil
						}
					} else {
						outFunc = func(content interface{}) (bool, error) {
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
					go func() {
						defer plug.lock.Unlock()
						if err := vm.PCall(len(paramNames), lua.MultRet, nil); err != nil {
							logs.Log.Errorf("error calling Lua %s:\n%s", fn.String(), err.Error())
							return
						}

						resultCount := vm.GetTop()
						for i := 1; i <= resultCount; i++ {
							// 从栈顶依次弹出返回值
							result := vm.Get(-resultCount + i - 1)
							_, err := outFunc(intermediate.ConvertLuaValueToGo(result))
							if err != nil {
								logs.Log.Errorf("error calling outFunc:\n%s", err.Error())
								return
							}
						}
						vm.Pop(resultCount)
					}()
				}()
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

func (plug *LuaPlugin) registerLuaOnHook(name string, condition intermediate.EventCondition) {
	vm := plug.vm
	if fn := vm.GetGlobal("on_" + name); fn != lua.LNil {
		plug.Events[condition] = func(event *clientpb.Event) (bool, error) {
			plug.lock.Lock()
			defer plug.lock.Unlock()
			vm.Push(fn)
			vm.Push(intermediate.ConvertGoValueToLua(vm, event))

			if err := vm.PCall(1, lua.MultRet, nil); err != nil {
				return false, fmt.Errorf("error calling Lua function %s: %w", name, err)
			}

			vm.Pop(vm.GetTop())
			return true, nil
		}
	}
}

func (plug *LuaPlugin) registerLuaFunction(name string, fn interface{}) {
	vm := plug.vm
	wrappedFunc := intermediate.WrapInternalFunc(fn)
	wrappedFunc.Package = intermediate.BuiltinPackage
	wrappedFunc.Name = name
	wrappedFunc.NoCache = true
	vm.SetGlobal(name, vm.NewFunction(intermediate.WrapFuncForLua(wrappedFunc)))
}

func globalLoader(plug *DefaultPlugin) func(L *lua.LState) int {
	return func(L *lua.LState) int {
		if err := L.DoString(string(plug.Content)); err != nil {
			logs.Log.Errorf("error loading Lua global script: %s", err.Error())
		}
		mod := L.Get(-1)
		L.Pop(1)

		if mod.Type() != lua.LTTable {
			mod = L.NewTable()
		}
		L.SetField(mod, "_NAME", lua.LString(plug.Name))
		L.Push(mod)
		return 1
	}
}

func luaLoader(L *lua.LState) int {
	// 从 LState 获取传入的包名
	packageName := L.ToString(1)

	mod := L.NewTable()
	L.SetField(mod, "_NAME", lua.LString(packageName))
	// 查找 InternalFunctions 中属于该包的函数并注册
	for _, fn := range intermediate.InternalFunctions {
		if fn.Package == packageName {
			mod.RawSetString(fn.Name, L.NewFunction(intermediate.WrapFuncForLua(fn)))
		}
	}

	// 如果没有找到函数，则返回空表
	L.Push(mod)
	return 1
}

func LoadLib(vm *lua.LState) {
	vm.OpenLibs()

	// https://github.com/vadv/gopher-lua-libs
	plugin.Preload(vm)
	argparse.Preload(vm)
	base64.Preload(vm)
	luafilepath.Preload(vm)
	goos.Preload(vm)
	humanize.Preload(vm)
	inspect.Preload(vm)
	ioutil.Preload(vm)
	json.Preload(vm)
	//pprof.Preload(vm)
	regexp.Preload(vm)
	//runtime.Preload(vm)
	shellescape.Preload(vm)
	storage.Preload(vm)
	luastrings.Preload(vm)
	tcp.Preload(vm)
	time.Preload(vm)
	stats.Loader(vm)
	//xmlpath.Preload(vm)
	luayaml.Preload(vm)
	db.Loader(vm)
	template.Loader(vm)
	log.Loader(vm)
	cmd.Loader(vm)

	vm.PreloadModule("http", gluahttp.NewHttpModule(&http.Client{}).Loader)
	vm.PreloadModule("crypto", luacrypto.Loader)

	// mal package
	vm.PreloadModule(intermediate.BeaconPackage, luaLoader)
	vm.PreloadModule(intermediate.RpcPackage, luaLoader)
	vm.PreloadModule(intermediate.ArmoryPackage, luaLoader)

	for _, global := range GlobalPlugins {
		vm.PreloadModule(global.Name, globalLoader(global))
	}
}

func NewLuaVM() *lua.LState {
	vm := lua.NewState()
	LoadLib(vm)
	RegisterProtobufMessageType(vm)
	RegisterAllProtobufMessages(vm)

	for name, fun := range intermediate.InternalFunctions {
		if fun.Package != intermediate.BuiltinPackage {
			continue
		}
		vm.SetGlobal(name, vm.NewFunction(intermediate.WrapFuncForLua(fun)))
	}
	return vm
}

// // 注册 Protobuf Message 的类型和方法
func RegisterProtobufMessageType(L *lua.LState) {
	mt := L.NewTypeMetatable("ProtobufMessage")
	L.SetGlobal("ProtobufMessage", mt)

	// 注册 __index 和 __newindex 元方法
	L.SetField(mt, "__index", L.NewFunction(protoIndex))
	L.SetField(mt, "__newindex", L.NewFunction(protoNewIndex))

	// 注册 __tostring 元方法
	L.SetField(mt, "__tostring", L.NewFunction(protoToString))

	L.SetField(mt, "New", L.NewFunction(protoNew))
}

func GenerateLuaDefinitionFile(L *lua.LState, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	generateProtobufMessageClasses(L, file)

	// 按 package 分组，然后在每个分组内按 funcName 排序
	groupedFunctions := make(map[string][]string)
	for funcName, signature := range intermediate.InternalFunctions {
		if unicode.IsUpper(rune(funcName[0])) {
			continue
		}
		groupedFunctions[signature.Package] = append(groupedFunctions[signature.Package], funcName)
	}

	// 排序每个 package 内的函数名
	for _, funcs := range groupedFunctions {
		sort.Strings(funcs)
	}

	// 生成 Lua 定义文件
	for group, funcs := range groupedFunctions {
		fmt.Fprintf(file, "-- Group: %s\n\n", group)
		for _, funcName := range funcs {
			signature := intermediate.InternalFunctions[funcName]

			fmt.Fprintf(file, "--- %s\n", funcName)

			// Short, Long, Example 描述
			if signature.Helper != nil {
				if signature.Helper.Short != "" {
					for _, line := range strings.Split(signature.Helper.Short, "\n") {
						fmt.Fprintf(file, "--- %s\n", line)
					}
					fmt.Fprintf(file, "---\n")
				}
				if signature.Helper.Long != "" {
					for _, line := range strings.Split(signature.Helper.Long, "\n") {
						fmt.Fprintf(file, "--- %s\n", line)
					}
					fmt.Fprintf(file, "---\n")
				}
				if signature.Helper.Example != "" {
					fmt.Fprintf(file, "--- @example\n")
					for _, line := range strings.Split(signature.Helper.Example, "\n") {
						fmt.Fprintf(file, "--- %s\n", line)
					}
					fmt.Fprintf(file, "---\n")
				}
			}

			// 参数和返回值描述
			var paramsName []string
			for i, argType := range signature.ArgTypes {
				luaType := intermediate.ConvertGoValueToLuaType(L, argType)
				if signature.Helper == nil {
					paramsName = append(paramsName, fmt.Sprintf("arg%d", i+1))
					fmt.Fprintf(file, "--- @param arg%d %s\n", i+1, luaType)
				} else {
					keys, values := signature.Helper.FormatInput()
					paramsName = append(paramsName, keys[i])
					fmt.Fprintf(file, "--- @param %s %s %s\n", keys[i], luaType, values[i])
				}
			}
			for _, returnType := range signature.ReturnTypes {
				luaType := intermediate.ConvertGoValueToLuaType(L, returnType)
				if signature.Helper == nil {
					fmt.Fprintf(file, "--- @return %s\n", luaType)
				} else {
					keys, values := signature.Helper.FormatOutput()
					for i := range keys {
						fmt.Fprintf(file, "--- @return %s %s %s\n", keys[i], luaType, values[i])
					}
				}
			}

			// 函数定义
			fmt.Fprintf(file, "function %s(", funcName)
			for i := range signature.ArgTypes {
				if i > 0 {
					fmt.Fprintf(file, ", ")
				}
				fmt.Fprintf(file, paramsName[i])
			}
			fmt.Fprintf(file, ") end\n\n")
		}
	}

	return nil
}

func GenerateMarkdownDefinitionFile(L *lua.LState, pkg, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// 按 package 分组，然后在每个分组内按 funcName 排序
	groupedFunctions := make(map[string][]string)
	for funcName, iFunc := range intermediate.InternalFunctions {
		if iFunc.Package != pkg {
			continue
		}
		group := "base"
		if iFunc.Helper != nil {
			group = iFunc.Helper.Group
		}
		groupedFunctions[group] = append(groupedFunctions[group], funcName)
	}

	// 排序每个 package 内的函数名
	for _, funcs := range groupedFunctions {
		sort.Strings(funcs)
	}

	// 生成 Markdown 文档
	for pkg, funcs := range groupedFunctions {
		// Package 名称作为二级标题
		fmt.Fprintf(file, "## %s\n\n", pkg)
		for _, funcName := range funcs {
			iFunc := intermediate.InternalFunctions[funcName]

			// 函数名作为三级标题
			fmt.Fprintf(file, "### %s\n\n", funcName)

			// 写入 Short 描述
			if iFunc.Helper != nil && iFunc.Helper.Short != "" {
				fmt.Fprintf(file, "%s\n\n", iFunc.Helper.Short)
			}

			// 写入 Long 描述
			if iFunc.Helper != nil && iFunc.Helper.Long != "" {
				for _, line := range strings.Split(iFunc.Helper.Long, "\n") {
					fmt.Fprintf(file, "%s\n", line)
				}
				fmt.Fprintf(file, "\n")
			}

			// 写入参数描述
			fmt.Fprintf(file, "**Arguments**\n\n")
			for i, argType := range iFunc.ArgTypes {
				luaType := intermediate.ConvertGoValueToLuaType(L, argType)
				if iFunc.Helper == nil {
					fmt.Fprintf(file, "- `$%d` [%s] \n", i+1, luaType)
				} else {
					keys, values := iFunc.Helper.FormatInput()
					paramName := fmt.Sprintf("$%d", i+1)
					if i < len(keys) && keys[i] != "" {
						paramName = keys[i]
					}
					description := ""
					if i < len(values) {
						description = values[i]
					}
					fmt.Fprintf(file, "- `%s` [%s] - %s\n", paramName, luaType, description)
				}
			}
			fmt.Fprintf(file, "\n")

			// Example
			if iFunc.Helper != nil && iFunc.Helper.Example != "" {
				fmt.Fprintf(file, "**Example**\n\n```\n")
				for _, line := range strings.Split(iFunc.Helper.Example, "\n") {
					fmt.Fprintf(file, "%s\n", line)
				}
				fmt.Fprintf(file, "```\n\n")
			}
		}
	}

	return nil
}

// generateProtobufMessageClasses 生成 Protobuf message 的 Lua class 定义
func generateProtobufMessageClasses(L *lua.LState, file *os.File) {
	// 使用 protoregistry 遍历所有注册的 Protobuf 结构体
	protoregistry.GlobalTypes.RangeMessages(func(mt protoreflect.MessageType) bool {
		// 获取结构体名称
		messageName := mt.Descriptor().FullName()
		var contains bool
		for _, pkg := range ProtoPackage {
			if strings.HasPrefix(string(messageName), pkg) {
				contains = true
			}
		}
		if !contains {
			return true
		}

		// 去掉前缀
		cleanName := removePrefix(string(messageName))

		// 写入 class 定义
		fmt.Fprintf(file, "--- @class %s\n", cleanName)

		fields := mt.Descriptor().Fields()
		for i := 0; i < fields.Len(); i++ {
			field := fields.Get(i)
			luaType := protoFieldToLuaType(field)
			fmt.Fprintf(file, "--- @field %s %s\n", field.Name(), luaType)
		}

		fmt.Fprintf(file, "\n")
		return true
	})
}

// 移除前缀 clientpb 或 implantpb
func removePrefix(messageName string) string {
	i := strings.Index(messageName, ".")
	if i == -1 {
		return messageName
	} else {
		return messageName[i+1:]
	}
}

// protoFieldToLuaType 将 Protobuf 字段映射为 Lua 类型
func protoFieldToLuaType(field protoreflect.FieldDescriptor) string {
	switch field.Kind() {
	case protoreflect.BoolKind:
		return "boolean"
	case protoreflect.Int32Kind, protoreflect.Int64Kind, protoreflect.Uint32Kind, protoreflect.Uint64Kind, protoreflect.FloatKind, protoreflect.DoubleKind:
		return "number"
	case protoreflect.StringKind:
		return "string"
	case protoreflect.BytesKind:
		return "string" // Lua 中处理为 string
	case protoreflect.MessageKind:
		if field.Cardinality() == protoreflect.Repeated {
			return "table"
		}
		return removePrefix(string(field.Message().FullName()))
	case protoreflect.EnumKind:
		return "string" // 枚举可以映射为字符串
	default:
		return "any"
	}
}

// RegisterProtobufMessagesFromPackage 注册指定包中所有的 Protobuf Message
func RegisterProtobufMessagesFromPackage(L *lua.LState, pkg string) {
	// 通过 protoregistry 获取所有注册的消息
	protoregistry.GlobalTypes.RangeMessages(func(mt protoreflect.MessageType) bool {
		messageName := string(mt.Descriptor().FullName())

		// 检查 message 是否属于指定包
		if len(pkg) == 0 || messageName == pkg || (len(messageName) >= len(pkg) && messageName[:len(pkg)] == pkg) {
			// 将每个 message 注册为 Lua 类型
			RegisterProtobufMessage(L, messageName, mt.New().Interface().(proto.Message))
		}
		return true
	})
}

// RegisterAllProtobufMessages 注册 implantpb 和 clientpb 中的所有 Protobuf Message
func RegisterAllProtobufMessages(L *lua.LState) {
	RegisterProtobufMessagesFromPackage(L, "implantpb")
	RegisterProtobufMessagesFromPackage(L, "clientpb")
	RegisterProtobufMessagesFromPackage(L, "modulepb")
}

// RegisterProtobufMessage 注册 Protobuf message 类型到 Lua
func RegisterProtobufMessage(L *lua.LState, msgType string, msg proto.Message) {
	mt := L.NewTypeMetatable(msgType)
	L.SetGlobal(msgType, mt)

	// 注册 Protobuf 操作
	L.SetField(mt, "__index", L.NewFunction(protoIndex))
	L.SetField(mt, "__newindex", L.NewFunction(protoNewIndex))
	L.SetField(mt, "__tostring", L.NewFunction(protoToString))

	// 新增 New 方法，用于创建该消息的空实例
	L.SetField(mt, "New", L.NewFunction(func(L *lua.LState) int {
		// 创建一个该消息的空实例
		newMsg := proto.Clone(msg).(proto.Message)

		// 将新创建的消息封装为 UserData
		ud := L.NewUserData()
		ud.Value = newMsg
		L.SetMetatable(ud, L.GetTypeMetatable(msgType))
		L.Push(ud)

		return 1 // 返回新建的消息实例
	}))
}

// __tostring 元方法：将 Protobuf 消息转换为字符串
func protoToString(L *lua.LState) int {
	ud := L.CheckUserData(1)
	if msg, ok := ud.Value.(proto.Message); ok {
		// 使用反射遍历并处理 Protobuf 消息的字段
		truncatedMsg := truncateMessageFields(msg)

		// 使用 protojson 将处理后的 Protobuf 消息转换为 JSON 字符串
		marshaler := protojson.MarshalOptions{
			Indent: "  ",
		}
		jsonStr, err := marshaler.Marshal(truncatedMsg)
		if err != nil {
			L.Push(lua.LString(fmt.Sprintf("Error: %v", err)))
		} else {
			L.Push(lua.LString(fmt.Sprintf("<ProtobufMessage: %s> %s", proto.MessageName(msg), string(jsonStr))))
		}
		return 1
	}
	L.Push(lua.LString("<invalid ProtobufMessage>"))
	return 1
}

// truncateLongFields 递归处理 map 中的字符串字段，截断长度超过 1024 的字符串
func truncateMessageFields(msg proto.Message) proto.Message {
	// 创建消息的深拷贝，以避免修改原始消息
	copyMsg := proto.Clone(msg)

	msgValue := reflect.ValueOf(copyMsg).Elem()
	msgType := msgValue.Type()

	for i := 0; i < msgType.NumField(); i++ {
		fieldValue := msgValue.Field(i)

		// 处理字符串类型字段
		if fieldValue.Kind() == reflect.String && fieldValue.Len() > 1024 {
			truncatedStr := fieldValue.String()[:1024] + "......"
			fieldValue.SetString(truncatedStr)
		}

		// 处理字节数组（[]byte）类型字段
		if fieldValue.Kind() == reflect.Slice && fieldValue.Type().Elem().Kind() == reflect.Uint8 {
			// 如果字节数组长度大于 1024，则截断
			if fieldValue.Len() > 1024 {
				truncatedBytes := append(fieldValue.Slice(0, 1024).Bytes(), []byte("......")...)
				fieldValue.SetBytes(truncatedBytes)
			}
		}

		// 处理嵌套的消息类型字段
		if fieldValue.Kind() == reflect.Ptr && !fieldValue.IsNil() && fieldValue.Elem().Kind() == reflect.Struct {
			nestedMsg, ok := fieldValue.Interface().(proto.Message)
			if ok {
				truncateMessageFields(nestedMsg)
			}
		}

		// 处理 repeated 字段（slice 类型）
		if fieldValue.Kind() == reflect.Slice && fieldValue.Type().Elem().Kind() == reflect.Ptr {
			for j := 0; j < fieldValue.Len(); j++ {
				item := fieldValue.Index(j)
				if item.Kind() == reflect.Ptr && item.Elem().Kind() == reflect.Struct {
					nestedMsg, ok := item.Interface().(proto.Message)
					if ok {
						truncateMessageFields(nestedMsg)
					}
				}
			}
		}
	}

	return copyMsg
}

func protoNew(L *lua.LState) int {
	// 获取消息类型名称
	msgTypeName := L.CheckString(2) // 这里确保第一个参数是字符串类型

	// 查找消息类型
	msgType, err := protoregistry.GlobalTypes.FindMessageByName(protoreflect.FullName(msgTypeName))
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString("invalid message type: " + msgTypeName))
		return 2
	}

	// 创建消息实例
	msg := msgType.New().Interface()

	// 初始化字段
	if L.GetTop() > 1 {
		initTable := L.CheckTable(3)
		initTable.ForEach(func(key lua.LValue, value lua.LValue) {
			fieldName := key.String()
			fieldValue := intermediate.ConvertLuaValueToGo(value)
			setFieldByName(msg, fieldName, fieldValue)
		})
	}

	// 将消息实例返回给 Lua
	ud := L.NewUserData()
	ud.Value = msg
	L.SetMetatable(ud, L.GetTypeMetatable("ProtobufMessage"))
	L.Push(ud)
	return 1
}

// __index 元方法：获取 Protobuf 消息的字段值
func protoIndex(L *lua.LState) int {
	ud := L.CheckUserData(1)
	fieldName := L.CheckString(2)

	if msg, ok := ud.Value.(proto.Message); ok {
		val := getFieldByName(msg, fieldName)
		L.Push(intermediate.ConvertGoValueToLua(L, val))
		return 1
	}
	return 0
}

// __newindex 元方法：设置 Protobuf 消息的字段值
func protoNewIndex(L *lua.LState) int {
	ud := L.CheckUserData(1)
	fieldName := L.CheckString(2)
	newValue := intermediate.ConvertLuaValueToGo(L.Get(3))

	if msg, ok := ud.Value.(proto.Message); ok {
		setFieldByName(msg, fieldName, newValue)
	}
	return 0
}

// 使用反射获取字段值
func getFieldByName(msg proto.Message, fieldName string) interface{} {
	val := reflect.ValueOf(msg).Elem().FieldByName(fieldName)
	if val.IsValid() {
		return val.Interface()
	}
	return nil
}

// 使用反射设置字段值
func setFieldByName(msg proto.Message, fieldName string, newValue interface{}) {
	val := reflect.ValueOf(msg).Elem().FieldByName(fieldName)
	if val.IsValid() && val.CanSet() {
		// 将 Lua 值转换为 Go 值并直接设置
		newVal := reflect.ValueOf(newValue)

		// 特别处理 []byte 类型
		if val.Kind() == reflect.Slice && val.Type().Elem().Kind() == reflect.Uint8 {
			if str, ok := newValue.(string); ok {
				newVal = reflect.ValueOf([]byte(str))
			}
		}

		// 检查是否可以直接设置值
		if newVal.Type().ConvertibleTo(val.Type()) {
			val.Set(newVal.Convert(val.Type()))
		}
	}
}
