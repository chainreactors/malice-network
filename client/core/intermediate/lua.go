package intermediate

import (
	"fmt"
	lua "github.com/yuin/gopher-lua"
	"google.golang.org/protobuf/proto"
	luar "layeh.com/gopher-luar"
	"reflect"
)

func WrapFuncForLua(fn *InternalFunc) lua.LGFunction {
	return func(L *lua.LState) int {
		var args []interface{}

		// 将 Lua 参数转换为 Go 参数
		for i := 1; i <= L.GetTop(); i++ {
			args = append(args, ConvertLuaValueToGo(L, L.Get(i)))
		}

		// 调用 Go 函数
		result, err := fn.Func(args...)
		if err != nil {
			L.Push(lua.LString(fmt.Sprintf("Error: %v", err)))
			return 1
		}

		// 将结果推回 Lua
		L.Push(ConvertGoValueToLua(L, result))

		return 1
	}
}

// 将 Lua 表转换为 Go 的 map[string]interface{}
func luaTableToMap(L *lua.LState, tbl *lua.LTable) map[string]interface{} {
	result := make(map[string]interface{})
	tbl.ForEach(func(key, value lua.LValue) {
		switch keyStr := key.(type) {
		case lua.LString:
			result[string(keyStr)] = ConvertLuaValueToGo(L, value)
		}
	})
	return result
}

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

func ConvertLuaValueToGo(L *lua.LState, value lua.LValue) interface{} {
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
func ConvertGoValueToLua(L *lua.LState, value interface{}) lua.LValue {
	switch v := value.(type) {
	case proto.Message:
		// 如果是 proto.Message 类型，将其封装为 LUserData 并设置元表
		ud := L.NewUserData()
		ud.Value = v
		L.SetMetatable(ud, L.GetTypeMetatable("ProtobufMessage"))
		return ud
	case []string:
		// 如果是 []string 类型，将其转换为 Lua 表
		luaTable := L.NewTable()
		for _, str := range v {
			luaTable.Append(lua.LString(str)) // 将每个 string 添加到表中
		}
		return luaTable
	default:
		return luar.New(L, value)
	}
}

func ConvertGoValueToLuaType(L *lua.LState, t reflect.Type) string {
	// 判断是否是 Protobuf 消息类型
	if t.Implements(reflect.TypeOf((*proto.Message)(nil)).Elem()) {
		// 返回具体的 Protobuf 类型名
		return t.Elem().Name()
	}

	// 处理其他类型
	switch t.Kind() {
	case reflect.Int, reflect.Int32, reflect.Int64, reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Bool:
		return "boolean"
	case reflect.String:
		return "string"
	case reflect.Slice:
		if t.Elem().Kind() == reflect.String {
			return "table<string>"
		}
		return "table"
	case reflect.Ptr:
		if t.Elem().Kind() == reflect.Struct {
			return "table"
		}
		return ConvertGoValueToLuaType(L, t.Elem()) // 递归处理指针类型
	default:
		return "any"
	}
}

func ConvertNumericType(value int64, kind reflect.Kind) interface{} {
	switch kind {
	case reflect.Int:
		return int(value)
	case reflect.Int8:
		return int8(value)
	case reflect.Int16:
		return int16(value)
	case reflect.Int32:
		return int32(value)
	case reflect.Int64:
		return int64(value)
	case reflect.Uint:
		return uint(value)
	case reflect.Uint8:
		return uint8(value)
	case reflect.Uint16:
		return uint16(value)
	case reflect.Uint32:
		return uint32(value)
	case reflect.Uint64:
		return uint64(value)
	case reflect.Float32:
		return float32(value)
	case reflect.Float64:
		return value
	default:
		return value // 其他类型，保持不变
	}
}
