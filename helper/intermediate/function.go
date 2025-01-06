package intermediate

import (
	"errors"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/mals"
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
