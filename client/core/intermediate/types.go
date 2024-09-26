package intermediate

import (
	"fmt"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
)

type InternalFunc struct {
	Name           string
	RawName        string
	Raw            interface{}
	Func           func(...interface{}) (interface{}, error)
	FinishCallback ImplantCallback // implant callback
	DoneCallback   ImplantCallback
	ArgTypes       []reflect.Type
	ReturnTypes    []reflect.Type
}

type ImplantCallback func(content *clientpb.TaskContext) (string, error)

var InternalFunctions = make(map[string]*InternalFunc)

// RegisterInternalFunc 注册并生成 Lua 定义文件
func RegisterInternalFunc(name string, fn *InternalFunc, callback ImplantCallback) error {
	name = strings.ReplaceAll(name, "-", "_")
	if callback != nil {
		fn.FinishCallback = callback
	}
	if _, ok := InternalFunctions[name]; ok {
		return fmt.Errorf("function %s already registered", name)
	}
	fn.Name = name
	InternalFunctions[name] = fn
	return nil
}

func RegisterInternalDoneCallback(name string, callback ImplantCallback) error {
	name = strings.ReplaceAll(name, "-", "_")
	if _, ok := InternalFunctions[name]; !ok {
		return fmt.Errorf("function %s not found", name)
	}
	InternalFunctions[name].DoneCallback = callback
	return nil
}

func RegisterFunction(name string, fn interface{}) {
	wrappedFunc := WrapInternalFunc(fn)
	err := RegisterInternalFunc(name, wrappedFunc, nil)
	if err != nil {
		return
	}
}

// 获取函数的参数和返回值类型
func GetInternalFuncSignature(fn interface{}) *InternalFunc {
	fnType := reflect.TypeOf(fn)

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
	return &InternalFunc{
		Raw:         fn,
		RawName:     filepath.Base(runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()),
		ArgTypes:    argTypes,
		ReturnTypes: returnTypes,
	}
}

func WrapInternalFunc(fun interface{}) *InternalFunc {
	internalFunc := GetInternalFuncSignature(fun)

	internalFunc.Func = func(params ...interface{}) (interface{}, error) {
		funcValue := reflect.ValueOf(fun)
		funcType := funcValue.Type()

		// 检查函数的参数数量是否匹配
		if funcType.NumIn() != len(params) {
			return nil, fmt.Errorf("expected %d arguments, got %d", funcType.NumIn(), len(params))
		}

		// 构建参数切片并检查参数类型
		in := make([]reflect.Value, len(params))
		for i, param := range params {
			expectedType := funcType.In(i)
			if reflect.TypeOf(param) != expectedType {
				return nil, fmt.Errorf("argument %d should be %v, got %v", i+1, expectedType, reflect.TypeOf(param))
			}
			in[i] = reflect.ValueOf(param)
		}

		// 调用原始函数并获取返回值
		results := funcValue.Call(in)

		// 处理返回值
		var result interface{}
		if len(results) > 0 {
			result = results[0].Interface()
		}

		var err error
		// 如果函数返回了多个值，最后一个值通常是 error
		if len(results) > 1 {
			if e, ok := results[len(results)-1].Interface().(error); ok {
				err = e
			}
		}

		return result, err
	}
	return internalFunc
}
