package intermediate

import (
	"errors"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
)

var (
	ErrFunctionNotFound = errors.New("function not found")
	WarnArgsMismatch    = errors.New("arguments mismatch")
	WarnReturnMismatch  = errors.New("return values mismatch")
)

type Helper struct {
	Group   string
	Short   string
	Long    string
	Input   []string
	Output  []string
	Example string
	CMDName string
}

func (help *Helper) FormatInput() ([]string, []string) {
	var keys, values []string
	if help.Input == nil {
		return keys, values
	}

	for _, input := range help.Input {
		i := strings.Index(input, ":")
		if i == -1 {
			keys = append(keys, input)
			values = append(values, "")
		} else {
			keys = append(keys, input[:i])
			values = append(values, input[i+1:])
		}
	}
	return keys, values
}

func (help *Helper) FormatOutput() ([]string, []string) {
	var keys, values []string
	if help.Output == nil {
		return keys, values
	}

	for _, output := range help.Output {
		i := strings.Index(output, ":")
		if i == -1 {
			keys = append(keys, output)
			values = append(values, "")
		} else {
			keys = append(keys, output[:i])
			values = append(values, output[i+1:])
		}
	}
	return keys, values
}

type InternalFunc struct {
	Name           string
	Package        string
	RawName        string
	Raw            interface{}
	Func           func(...interface{}) (interface{}, error)
	HasLuaCallback bool
	NoCache        bool
	FinishCallback ImplantCallback // implant callback
	DoneCallback   ImplantCallback
	ArgTypes       []reflect.Type
	ReturnTypes    []reflect.Type
	*Helper
}

func (fn *InternalFunc) String() string {
	return fmt.Sprintf("%s.%s", fn.Package, fn.Name)
}

// callback to callee, like lua or go, return string
type ImplantCallback func(content *clientpb.TaskContext) (string, error)

var InternalFunctions = make(map[string]*InternalFunc)

// RegisterInternalFunc 注册并生成 Lua 定义文件
func RegisterInternalFunc(pkg, name string, fn *InternalFunc, callback ImplantCallback) error {
	fn.Package = pkg
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

func AddHelper(name string, helper *Helper) error {
	name = strings.ReplaceAll(name, "-", "_")
	if fn, ok := InternalFunctions[name]; ok {
		if helper.Input != nil && len(helper.Input) != len(fn.ArgTypes) {
			logs.Log.Warnf("function %s %s", name, WarnArgsMismatch.Error())
		}
		if helper.Output != nil && len(helper.Output) != len(fn.ReturnTypes) {
			logs.Log.Warnf("function %s %s", name, WarnReturnMismatch.Error())
		}
		fn.Helper = helper
		return nil
	} else {
		return fmt.Errorf("%s %w", name, ErrFunctionNotFound)
	}
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
	err := RegisterInternalFunc(BuiltinPackage, name, wrappedFunc, nil)
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
