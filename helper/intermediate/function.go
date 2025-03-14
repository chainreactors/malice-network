package intermediate

import (
	"errors"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/mals"
	"reflect"
	"strings"
)

var (
	ErrFunctionNotFound = errors.New("function not found")
	WarnArgsMismatch    = errors.New("arguments mismatch")
	WarnReturnMismatch  = errors.New("return values mismatch")
)

type InternalFunc struct {
	*mals.MalFunction
	FinishCallback ImplantCallback // implant callback
	DoneCallback   ImplantCallback
}

// callback to callee, like lua or go, return string
type ImplantCallback func(content *clientpb.TaskContext) (string, error)

var InternalFunctions = make(internalFuncs)

type internalFuncs map[string]*InternalFunc

func (fns internalFuncs) All() map[string]*mals.MalFunction {
	ret := make(map[string]*mals.MalFunction)
	for k, v := range fns {
		ret[k] = v.MalFunction
	}
	return ret
}

// package
func (fns internalFuncs) Package(pkg string) map[string]*mals.MalFunction {
	ret := make(map[string]*mals.MalFunction)
	for k, v := range fns {
		if v.Package == pkg {
			ret[k] = v.MalFunction
		}
	}
	return ret
}

// RegisterInternalFunc 注册并生成 Lua 定义文件
func RegisterInternalFunc(pkg, name string, fn *mals.MalFunction, callback ImplantCallback) error {
	fn.Package = pkg
	name = strings.ReplaceAll(name, "-", "_")
	fn.Name = name
	ifn := &InternalFunc{
		MalFunction: fn,
	}
	if callback != nil {
		ifn.FinishCallback = callback
	}
	if _, ok := InternalFunctions[name]; ok {
		return fmt.Errorf("function %s already registered", name)
	}
	InternalFunctions[name] = ifn
	return nil
}

func AddHelper(name string, helper *mals.Helper) error {
	name = strings.ReplaceAll(name, "-", "_")
	if fn, ok := InternalFunctions[name]; ok {
		if helper.Input != nil && len(helper.Input) != len(fn.ArgTypes) {
			logs.Log.Warnf("function %s %s\n", name, WarnArgsMismatch.Error())
		}
		if helper.Output != nil && len(helper.Output) != len(fn.ReturnTypes) {
			logs.Log.Warnf("function %s %s\n", name, WarnReturnMismatch.Error())
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
	err := RegisterInternalFunc(BuiltinPackage, name, mals.WrapInternalFunc(fn), nil)
	if err != nil {
		return
	}
}

func WrapFunctionReturn(fn interface{}) func(args ...interface{}) (interface{}, error) {
	fnValue := reflect.ValueOf(fn)
	fnType := fnValue.Type()

	return func(args ...interface{}) (interface{}, error) {
		// 参数检查
		numIn := fnType.NumIn()
		isVariadic := fnType.IsVariadic()
		minArgs := numIn
		if isVariadic {
			minArgs--
		}
		if len(args) < minArgs || (!isVariadic && len(args) != numIn) {
			return nil, fmt.Errorf("expected %d arguments, got %d", numIn, len(args))
		}
		for i := 0; i < len(args); i++ {
			var expectedType reflect.Type
			if isVariadic && i >= minArgs {
				expectedType = fnType.In(numIn - 1).Elem()
			} else {
				expectedType = fnType.In(i)
			}
			if reflect.TypeOf(args[i]) != expectedType {
				return nil, fmt.Errorf("argument %d has wrong type: expected %v, got %v", i, expectedType, reflect.TypeOf(args[i]))
			}
		}

		// 调用原函数
		inValues := make([]reflect.Value, len(args))
		for i, arg := range args {
			inValues[i] = reflect.ValueOf(arg)
		}
		results := fnValue.Call(inValues)

		// 处理返回值
		numOut := fnType.NumOut()
		if numOut == 0 {
			// 无返回值，返回 (true, nil)
			return true, nil
		}

		// 检查最后一个返回值是否为 error
		var err error
		hasError := numOut > 0 && fnType.Out(numOut-1) == reflect.TypeOf((*error)(nil)).Elem()
		if hasError {
			err, _ = results[numOut-1].Interface().(error)
		}

		// 根据返回值数量处理
		switch numOut {
		case 1:
			if hasError {
				// 单个 error 返回值，返回 (bool, error)
				return err == nil, err
			}
			// 单个非 error 返回值，返回 (value, nil)
			return results[0].Interface(), nil
		case 2:
			if hasError {
				// 两个返回值，第二个是 error，返回 (第一个值, error)
				return results[0].Interface(), err
			}
			// 两个非 error 返回值，打包成 []interface{}
			return []interface{}{results[0].Interface(), results[1].Interface()}, nil
		default:
			// 超过两个返回值，将非 error 值打包成 []interface{}
			count := numOut
			if hasError {
				count--
			}
			values := make([]interface{}, count)
			for i := 0; i < count; i++ {
				values[i] = results[i].Interface()
			}
			return values, err
		}
	}
}
