package intermediate

import (
	"fmt"
	"os"
	"reflect"
)

type InternalFunc func(...interface{}) (interface{}, error)
type FunctionSignature struct {
	ArgTypes    []reflect.Type
	ReturnTypes []reflect.Type
}

var InternalFunctions = make(map[string]InternalFunc)
var InternalFunctionSignatures = make(map[string]*FunctionSignature)

// RegisterInternalFunc 注册并生成 Lua 定义文件
func RegisterInternalFunc(name string, fn InternalFunc, sig *FunctionSignature) error {
	if _, ok := InternalFunctions[name]; ok {
		return fmt.Errorf("function %s already registered", name)
	}

	InternalFunctions[name] = fn
	InternalFunctionSignatures[name] = sig
	fmt.Printf("%s %v\n", name, sig)
	return nil
}

// 获取函数的参数和返回值类型
func GetInternalFuncSignature(fn reflect.Value) *FunctionSignature {
	fnType := fn.Type()

	// 获取参数类型
	numArgs := fnType.NumIn()
	argTypes := make([]reflect.Type, numArgs)
	for i := 0; i < numArgs; i++ {
		argTypes[i] = fnType.In(i)
	}

	// 获取返回值类型
	numReturns := fnType.NumOut()
	// 如果最后一个返回值是 error 类型，忽略它
	if numReturns > 0 && fnType.Out(numReturns-1) == reflect.TypeOf((*error)(nil)).Elem() {
		numReturns--
	}
	returnTypes := make([]reflect.Type, numReturns)
	for i := 0; i < numReturns; i++ {
		returnTypes[i] = fnType.Out(i)
	}

	return &FunctionSignature{
		ArgTypes:    argTypes,
		ReturnTypes: returnTypes,
	}
}

// 将函数签名写入定义文件
func appendToLuaDefinitionFile(name string, argTypes, returnTypes []reflect.Type) error {
	// 打开文件以追加内容
	file, err := os.OpenFile("builtin.lua", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// 写入函数签名
	signature := fmt.Sprintf("function %s(", name)

	// 写入参数类型
	for i, argType := range argTypes {
		if i > 0 {
			signature += ", "
		}
		signature += argType.String()
	}

	signature += ")"

	// 写入返回值类型
	if len(returnTypes) > 0 {
		signature += " -> "
		for i, returnType := range returnTypes {
			if i > 0 {
				signature += ", "
			}
			signature += returnType.String()
		}
	}

	signature += " end\n"

	// 写入文件
	if _, err := file.WriteString(signature); err != nil {
		return err
	}

	return nil
}
