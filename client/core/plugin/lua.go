package plugin

import (
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/core/intermediate"
	"github.com/cjoudrey/gluahttp"
	"github.com/kballard/go-shellquote"
	"github.com/spf13/cobra"
	luacrypto "github.com/tengattack/gluacrypto/crypto"
	"github.com/vadv/gopher-lua-libs/argparse"
	"github.com/vadv/gopher-lua-libs/base64"
	luafilepath "github.com/vadv/gopher-lua-libs/filepath"
	"github.com/vadv/gopher-lua-libs/goos"
	"github.com/vadv/gopher-lua-libs/humanize"
	"github.com/vadv/gopher-lua-libs/inspect"
	"github.com/vadv/gopher-lua-libs/ioutil"
	"github.com/vadv/gopher-lua-libs/json"
	"github.com/vadv/gopher-lua-libs/plugin"
	"github.com/vadv/gopher-lua-libs/pprof"
	"github.com/vadv/gopher-lua-libs/regexp"
	"github.com/vadv/gopher-lua-libs/runtime"
	"github.com/vadv/gopher-lua-libs/shellescape"
	"github.com/vadv/gopher-lua-libs/storage"
	luastrings "github.com/vadv/gopher-lua-libs/strings"
	"github.com/vadv/gopher-lua-libs/tcp"
	"github.com/vadv/gopher-lua-libs/time"
	"github.com/vadv/gopher-lua-libs/xmlpath"
	"github.com/vadv/gopher-lua-libs/yaml"
	"github.com/yuin/gluare"
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
	"strings"
	"sync"
	"unicode"
)

var (
	ReservedARGS    = "args"
	ReservedCMDLINE = "cmdline"
	ReservedWords   = []string{ReservedCMDLINE, ReservedARGS}

	LuaPackages = map[string]*lua.LTable{}
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

	//globals := vm.Get(lua.GlobalsIndex).(*lua.LTable)
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

	return mal, nil
}

func (plug *LuaPlugin) Run() error {
	if err := plug.vm.DoString(string(plug.Content)); err != nil {
		return fmt.Errorf("failed to load Lua script: %w", err)
	}
	return nil
}

func (plug *LuaPlugin) RegisterLuaBuiltin() error {
	vm := plug.vm
	plugDir := filepath.Join(assets.GetMalsDir(), plug.Name)
	tempDir := assets.GetTempDir()
	vm.SetGlobal("plugin_dir", lua.LString(plugDir))
	vm.SetGlobal("plugin_resource_dir", lua.LString(filepath.Join(plugDir, "resources")))
	vm.SetGlobal("plugin_name", lua.LString(plug.Name))
	vm.SetGlobal("temp_dir", lua.LString(tempDir))
	packageMod := vm.GetGlobal("package").(*lua.LTable)
	luaPath := lua.LuaPathDefault + ";" + plugDir + "\\?.lua"
	vm.SetField(packageMod, "path", lua.LString(luaPath))
	// 读取resource文件
	plug.registerLuaFunction("script_resource", func(filename string) (string, error) {
		return intermediate.GetResourceFile(plug.Name, filename)
	})

	plug.registerLuaFunction("find_resource", func(sess *core.Session, base string, ext string) (string, error) {
		return intermediate.FindResourceFile(plug.Name, base, sess.Os.Arch, ext)
	})

	// 读取资源文件内容
	plug.registerLuaFunction("read_resource", func(filename string) (string, error) {
		return intermediate.ReadResourceFile(plug.Name, filename)
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

	plug.registerLuaFunction("command", func(name string, fn *lua.LFunction, short string, ttp string) (bool, error) {
		cmd := plug.CMDs.Find(name)

		var paramNames []string
		for _, param := range fn.Proto.DbgLocals {
			if strings.HasPrefix(param.Name, "_") {
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

func (plug *LuaPlugin) registerLuaFunction(name string, fn interface{}) {
	vm := plug.vm
	wrappedFunc := intermediate.WrapInternalFunc(fn)
	wrappedFunc.Package = intermediate.BuiltinPackage
	wrappedFunc.Name = name
	wrappedFunc.NoCache = true
	vm.SetGlobal(name, vm.NewFunction(intermediate.WrapFuncForLua(wrappedFunc)))
}

func luaLoader(L *lua.LState) int {
	// 从 LState 获取传入的包名
	packageName := L.ToString(1)

	// 创建模块表
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
	plugin.Preload(vm)
	argparse.Preload(vm)
	base64.Preload(vm)
	luafilepath.Preload(vm)
	goos.Preload(vm)
	humanize.Preload(vm)
	inspect.Preload(vm)
	ioutil.Preload(vm)
	json.Preload(vm)
	pprof.Preload(vm)
	regexp.Preload(vm)
	runtime.Preload(vm)
	shellescape.Preload(vm)
	storage.Preload(vm)
	luastrings.Preload(vm)
	tcp.Preload(vm)
	time.Preload(vm)
	xmlpath.Preload(vm)
	yaml.Preload(vm)

	vm.PreloadModule("http", gluahttp.NewHttpModule(&http.Client{}).Loader)
	vm.PreloadModule("crypto", luacrypto.Loader)
	vm.PreloadModule("re", gluare.Loader)

	// mal package
	vm.PreloadModule(intermediate.BeaconPackage, luaLoader)
	vm.PreloadModule(intermediate.RpcPackage, luaLoader)
	vm.PreloadModule(intermediate.ArmoryPackage, luaLoader)
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

// generateLuaDefinitionFile 生成 Lua 函数定义和 Protobuf class 定义文件
func GenerateLuaDefinitionFile(L *lua.LState, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// 生成 Protobuf message 的 class 定义
	generateProtobufMessageClasses(L, file)

	// 遍历所有函数签名并生成 Lua 函数定义
	for funcName, signature := range intermediate.InternalFunctions {
		// 写入函数名称注释
		if unicode.IsUpper(rune(funcName[0])) {
			continue
		}
		fmt.Fprintf(file, "--- %s\n", funcName)

		// 写入参数注释
		for i, argType := range signature.ArgTypes {
			luaType := intermediate.ConvertGoValueToLuaType(L, argType)
			fmt.Fprintf(file, "--- @param arg%d %s\n", i+1, luaType)
		}

		// 写入返回值注释
		for _, returnType := range signature.ReturnTypes {
			luaType := intermediate.ConvertGoValueToLuaType(L, returnType)
			fmt.Fprintf(file, "--- @return %s\n", luaType)
		}

		// 写入函数定义
		fmt.Fprintf(file, "function %s(", funcName)
		for i := range signature.ArgTypes {
			if i > 0 {
				fmt.Fprintf(file, ", ")
			}
			fmt.Fprintf(file, "arg%d", i+1)
		}
		fmt.Fprintf(file, ") end\n\n")
	}

	return nil
}

// generateProtobufMessageClasses 生成 Protobuf message 的 Lua class 定义
func generateProtobufMessageClasses(L *lua.LState, file *os.File) {
	// 使用 protoregistry 遍历所有注册的 Protobuf 结构体
	protoregistry.GlobalTypes.RangeMessages(func(mt protoreflect.MessageType) bool {
		// 获取结构体名称
		messageName := mt.Descriptor().FullName()

		// 只处理 clientpb 和 implantpb 下的 message
		if !(strings.HasPrefix(string(messageName), "clientpb.") || strings.HasPrefix(string(messageName), "implantpb.")) {
			return true
		}

		// 去掉前缀
		cleanName := removePrefix(string(messageName))

		// 写入 class 定义
		fmt.Fprintf(file, "--- @class %s\n", cleanName)

		// 遍历字段并写入注释
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
	if len(messageName) >= 9 && messageName[:9] == "clientpb." {
		return messageName[9:]
	}
	if len(messageName) >= 10 && messageName[:10] == "implantpb." {
		return messageName[10:]
	}
	return messageName
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
		// 去掉前缀，处理 message 类型字段
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
	// 只需调用函数，不要返回值
	RegisterProtobufMessagesFromPackage(L, "implantpb")
	RegisterProtobufMessagesFromPackage(L, "clientpb")
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
