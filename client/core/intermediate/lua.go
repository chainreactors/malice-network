package intermediate

import (
	"fmt"
	lua "github.com/yuin/gopher-lua"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	luar "layeh.com/gopher-luar"
	"reflect"
)

var InternalFunctions = make(map[string]InternalFunc)

type InternalFunc func(...interface{}) (interface{}, error)

func WrapFuncForLua(fn InternalFunc, expectedArgs []reflect.Type) lua.LGFunction {
	return func(L *lua.LState) int {
		var args []interface{}
		top := L.GetTop()

		// 检查参数数量
		if len(expectedArgs) != top {
			L.Push(lua.LString(fmt.Sprintf("Error: expected %d arguments, got %d", len(expectedArgs), top)))
			return 1
		}

		// 将 Lua 参数转换为 Go 参数并检查类型
		for i := 1; i <= top; i++ {
			arg := convertLuaValueToGo(L, L.Get(i))
			if reflect.TypeOf(arg) != expectedArgs[i-1] {
				L.Push(lua.LString(fmt.Sprintf("Error: argument %d expected type %v, got %v", i, expectedArgs[i-1], reflect.TypeOf(arg))))
				return 1
			}
			args = append(args, arg)
		}

		// 调用 Go 函数
		result, err := fn(args...)
		if err != nil {
			L.Push(lua.LString(fmt.Sprintf("Error: %v", err)))
			return 1
		}

		// 将结果推回 Lua
		L.Push(convertGoValueToLua(L, result))

		return 1
	}
}

func WrapDynamicFuncForLua(fn InternalFunc) lua.LGFunction {
	return func(L *lua.LState) int {
		var args []interface{}

		// 将 Lua 参数转换为 Go 参数
		for i := 1; i <= L.GetTop(); i++ {
			args = append(args, convertLuaValueToGo(L, L.Get(i)))
		}

		// 调用 Go 函数
		result, err := fn(args...)
		if err != nil {
			L.Push(lua.LString(fmt.Sprintf("Error: %v", err)))
			return 1
		}

		// 将结果推回 Lua
		L.Push(convertGoValueToLua(L, result))

		return 1
	}
}

// 注册 Protobuf Message 的类型和方法
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

// __tostring 元方法：将 Protobuf 消息转换为字符串
func protoToString(L *lua.LState) int {
	ud := L.CheckUserData(1)
	if msg, ok := ud.Value.(proto.Message); ok {
		// 使用 protojson 将 Protobuf 消息转换为 JSON 字符串
		marshaler := protojson.MarshalOptions{
			Indent: "  ", // 美化输出
		}
		jsonStr, err := marshaler.Marshal(msg)
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
			fieldValue := convertLuaValueToGo(L, value)
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
		L.Push(convertGoValueToLua(L, val))
		return 1
	}
	return 0
}

// __newindex 元方法：设置 Protobuf 消息的字段值
func protoNewIndex(L *lua.LState) int {
	ud := L.CheckUserData(1)
	fieldName := L.CheckString(2)
	newValue := convertLuaValueToGo(L, L.Get(3))

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

// 将 Lua 表转换为 Go 的 map[string]interface{}
func luaTableToMap(L *lua.LState, tbl *lua.LTable) map[string]interface{} {
	result := make(map[string]interface{})
	tbl.ForEach(func(key, value lua.LValue) {
		switch keyStr := key.(type) {
		case lua.LString:
			result[string(keyStr)] = convertLuaValueToGo(L, value)
		}
	})
	return result
}

//// 将 Go 的 interface{} 值转换为 Lua 值
//func convertGoValueToLua(L *lua.LState, value interface{}) lua.LValue {
//	switch v := value.(type) {
//	case string:
//		return lua.LString(v)
//	case int, int8, int16, int32, int64:
//		return lua.LNumber(reflect.ValueOf(v).Int())
//	case uint, uint8, uint16, uint32, uint64:
//		return lua.LNumber(reflect.ValueOf(v).Uint())
//	case float32, float64:
//		return lua.LNumber(reflect.ValueOf(v).Float())
//	case bool:
//		return lua.LBool(v)
//	case proto.Message:
//		// 如果是 proto.Message 类型，将其封装为 LUserData
//		ud := L.NewUserData()
//		ud.Value = v
//		L.SetMetatable(ud, L.GetTypeMetatable("ProtobufMessage"))
//		return ud
//	case map[string]interface{}:
//		return mapToLuaTable(L, v)
//	case []interface{}:
//		luaArray := L.NewTable()
//		for _, item := range v {
//			luaArray.Append(convertGoValueToLua(L, item))
//		}
//		return luaArray
//	case nil:
//		return lua.LNil
//	default:
//		return lua.LString(fmt.Sprintf("%v", v))
//	}
//}

func isArrayTable(L *lua.LState, tbl *lua.LTable) bool {
	maxKey := 0
	isArray := true
	tbl.ForEach(func(key lua.LValue, value lua.LValue) {
		if key.Type() != lua.LTNumber {
			isArray = false
		} else {
			if int(lua.LVAsNumber(key)) > maxKey {
				maxKey = int(lua.LVAsNumber(key))
			}
		}
	})
	return isArray && maxKey == tbl.Len()
}

func luaTableToStringSlice(L *lua.LState, tbl *lua.LTable) []string {
	var result []string
	tbl.ForEach(func(key lua.LValue, value lua.LValue) {
		if str, ok := value.(lua.LString); ok {
			result = append(result, string(str))
		}
	})
	return result
}

func convertLuaValueToGo(L *lua.LState, value lua.LValue) interface{} {
	switch v := value.(type) {
	case lua.LString:
		return string(v)
	case lua.LNumber:
		if v == lua.LNumber(int64(v)) {
			return int64(v)
		}
		return float64(v)
	case lua.LBool:
		return bool(v)
	case *lua.LTable:
		if isArrayTable(L, v) {
			return luaTableToStringSlice(L, v)
		}
		return luaTableToMap(L, v)
	case *lua.LUserData:
		if protoMsg, ok := v.Value.(proto.Message); ok {
			return protoMsg
		}
		return v.Value
	case *lua.LNilType:
		return nil
	default:
		return v.String()
	}
}

// 将 Lua 的 lua.LValue 转换为 Go 的 interface{}
func convertGoValueToLua(L *lua.LState, value interface{}) lua.LValue {
	switch v := value.(type) {
	case proto.Message:
		// 如果是 proto.Message 类型，将其封装为 LUserData 并设置元表
		ud := L.NewUserData()
		ud.Value = v
		L.SetMetatable(ud, L.GetTypeMetatable("ProtobufMessage"))
		return ud
	default:
		// 对于其他类型，使用 luar.New 进行处理
		return luar.New(L, value)
	}
}
