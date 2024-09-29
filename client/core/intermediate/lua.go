package intermediate

import (
	"fmt"
	"github.com/chainreactors/utils/iutils"
	lua "github.com/yuin/gopher-lua"
	"google.golang.org/protobuf/proto"
	luar "layeh.com/gopher-luar"
	"reflect"
)

var (
	luaFunctionCache = map[string]lua.LGFunction{}
)

const (
	BeaconPackage  = "beacon"
	RpcPackage     = "rpc"
	ArmoryPackage  = "armory"
	BuiltinPackage = "builtin"
)

func WrapFuncForLua(fn *InternalFunc) lua.LGFunction {
	if luaFn, ok := luaFunctionCache[fn.String()]; !fn.NoCache && ok {
		return luaFn
	}

	luaFn := func(vm *lua.LState) int {
		var args []interface{}
		top := vm.GetTop()

		// 检查最后一个参数是否为回调函数
		var callback *lua.LFunction
		if top > 0 && fn.HasLuaCallback {
			if vm.Get(top).Type() == lua.LTFunction {
				callback = vm.Get(top).(*lua.LFunction)
				top-- // 去掉回调函数，调整参数数量
			}
		}

		// 将 Lua 参数转换为 Go 参数
		for i := 1; i <= top; i++ {
			args = append(args, ConvertLuaValueToGo(vm.Get(i)))
		}
		args, err := ConvertArgsToExpectedTypes(args, fn.ArgTypes)
		if err != nil {
			vm.Error(lua.LString(fmt.Sprintf("Error: %v", err)), 1)
			return 0
		}
		// 调用 Go 函数
		result, err := fn.Func(args...)
		if err != nil {
			vm.Error(lua.LString(fmt.Sprintf("Error: %v", err)), 1)
			return 0
		}

		// 如果有回调，调用回调函数
		if callback != nil {
			vm.Push(callback)
			vm.Push(ConvertGoValueToLua(vm, result))
			// 可以推送需要传递给回调的参数，如果需要的话
			if err := vm.PCall(1, 1, nil); err != nil { // 期待一个返回值
				vm.Error(lua.LString(fmt.Sprintf("Callback Error: %v", err)), 1)
				return 0
			}

			return 1
		} else {
			vm.Push(ConvertGoValueToLua(vm, result))
			return 1
		}
	}
	luaFunctionCache[fn.String()] = luaFn
	return luaFn
}

// Convert the []interface{} and map[string]interface{} to the expected types defined in ArgTypes
func ConvertArgsToExpectedTypes(args []interface{}, argTypes []reflect.Type) ([]interface{}, error) {
	if len(args) != len(argTypes) {
		return nil, fmt.Errorf("argument count mismatch: expected %d, got %d", len(argTypes), len(args))
	}

	convertedArgs := make([]interface{}, len(args))

	for i, arg := range args {
		expectedType := argTypes[i]
		val := reflect.ValueOf(arg)

		// Skip conversion if types are already identical
		if val.Type() == expectedType {
			convertedArgs[i] = arg
			continue
		}

		// Handle string conversion with ToString
		if expectedType.Kind() == reflect.String {
			convertedArgs[i] = iutils.ToString(arg)
			continue
		}

		// Handle slice conversion
		if expectedType.Kind() == reflect.Slice && val.Kind() == reflect.Slice {
			elemType := expectedType.Elem()
			sliceVal := reflect.MakeSlice(expectedType, val.Len(), val.Len())
			for j := 0; j < val.Len(); j++ {
				elem := val.Index(j)
				convertedElem, err := convertValueToExpectedType(elem.Interface(), elemType)
				if err != nil {
					return nil, fmt.Errorf("cannot convert slice element at index %d: %v", j, err)
				}
				sliceVal.Index(j).Set(reflect.ValueOf(convertedElem))
			}
			convertedArgs[i] = sliceVal.Interface()
			continue
		}

		// Handle map conversion
		if expectedType.Kind() == reflect.Map && val.Kind() == reflect.Map {
			keyType := expectedType.Key()
			elemType := expectedType.Elem()
			mapVal := reflect.MakeMap(expectedType)
			for _, key := range val.MapKeys() {
				convertedKey, err := convertValueToExpectedType(key.Interface(), keyType)
				if err != nil {
					return nil, fmt.Errorf("cannot convert map key %v: %v", key, err)
				}
				convertedValue, err := convertValueToExpectedType(val.MapIndex(key).Interface(), elemType)
				if err != nil {
					return nil, fmt.Errorf("cannot convert map value for key %v: %v", key, err)
				}
				mapVal.SetMapIndex(reflect.ValueOf(convertedKey), reflect.ValueOf(convertedValue))
			}
			convertedArgs[i] = mapVal.Interface()
			continue
		}

		// Default conversion using reflect.Convert
		if val.Type().ConvertibleTo(expectedType) {
			convertedArgs[i] = val.Convert(expectedType).Interface()
		} else {
			return nil, fmt.Errorf("cannot convert argument %d to %s", i+1, expectedType)
		}
	}
	return convertedArgs, nil
}

// Helper function to convert individual values to the expected type
func convertValueToExpectedType(value interface{}, expectedType reflect.Type) (interface{}, error) {
	val := reflect.ValueOf(value)

	// Skip conversion if types are already identical
	if val.Type() == expectedType {
		return value, nil
	}

	// Handle string conversion
	if expectedType.Kind() == reflect.String {
		return iutils.ToString(value), nil
	}

	// Handle slice conversion
	if expectedType.Kind() == reflect.Slice && val.Kind() == reflect.Slice {
		elemType := expectedType.Elem()
		sliceVal := reflect.MakeSlice(expectedType, val.Len(), val.Len())
		for j := 0; j < val.Len(); j++ {
			convertedElem, err := convertValueToExpectedType(val.Index(j).Interface(), elemType)
			if err != nil {
				return nil, fmt.Errorf("cannot convert slice element at index %d: %v", j, err)
			}
			sliceVal.Index(j).Set(reflect.ValueOf(convertedElem))
		}
		return sliceVal.Interface(), nil
	}

	// Handle map conversion
	if expectedType.Kind() == reflect.Map && val.Kind() == reflect.Map {
		keyType := expectedType.Key()
		elemType := expectedType.Elem()
		mapVal := reflect.MakeMap(expectedType)
		for _, key := range val.MapKeys() {
			convertedKey, err := convertValueToExpectedType(key.Interface(), keyType)
			if err != nil {
				return nil, fmt.Errorf("cannot convert map key %v: %v", key, err)
			}
			convertedValue, err := convertValueToExpectedType(val.MapIndex(key).Interface(), elemType)
			if err != nil {
				return nil, fmt.Errorf("cannot convert map value for key %v: %v", key, err)
			}
			mapVal.SetMapIndex(reflect.ValueOf(convertedKey), reflect.ValueOf(convertedValue))
		}
		return mapVal.Interface(), nil
	}

	// Default conversion
	if val.Type().ConvertibleTo(expectedType) {
		return val.Convert(expectedType).Interface(), nil
	}

	return nil, fmt.Errorf("cannot convert value to %s", expectedType)
}

func isArray(tbl *lua.LTable) bool {
	length := tbl.Len() // Length of the array part
	count := 0
	isSequential := true
	tbl.ForEach(func(key, val lua.LValue) {
		if k, ok := key.(lua.LNumber); ok {
			index := int(k)
			if index != count+1 {
				isSequential = false
			}
			count++
		} else {
			isSequential = false
		}
	})
	return isSequential && count == length
}

// ConvertLuaTableToGo takes a Lua table and converts it into a Go slice or map
func ConvertLuaTableToGo(tbl *lua.LTable) interface{} {
	// Check if the Lua table is an array (keys are sequential integers starting from 1)
	if isArray(tbl) {
		// Convert to Go slice
		var array []interface{}
		tbl.ForEach(func(key, val lua.LValue) {
			array = append(array, ConvertLuaValueToGo(val))
		})
		return array
	}

	// Otherwise, convert to Go map
	m := make(map[string]interface{})
	tbl.ForEach(func(key, val lua.LValue) {
		m[key.String()] = ConvertLuaValueToGo(val)
	})
	return m
}

func ConvertLuaValueToGo(value lua.LValue) interface{} {
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
		return ConvertLuaTableToGo(v)
	case *lua.LUserData:
		if protoMsg, ok := v.Value.(proto.Message); ok {
			return protoMsg
		}
		return v.Value
	case *lua.LNilType:
		return nil
	case *lua.LFunction:
		return v
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
