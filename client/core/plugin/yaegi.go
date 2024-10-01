package plugin

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/core/intermediate"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
	"reflect"
)

func NewGoMalPlugin(manifest *MalManiFest) (*GoMalPlugin, error) {
	plug, err := NewPlugin(manifest)
	if err != nil {
		return nil, err
	}
	mal := &GoMalPlugin{
		DefaultPlugin: plug,
		Interpreter:   NewYaegiInterpreter(),
	}
	return mal, nil

}

func NewYaegiInterpreter() *interp.Interpreter {
	i := interp.New(interp.Options{})
	i.Use(stdlib.Symbols)

	functionMap := make(map[string]map[string]reflect.Value)

	// 遍历所有 InternalFunctions
	for _, fun := range intermediate.InternalFunctions {
		var packageName string
		if fun.Package == intermediate.BuiltinPackage {
			packageName = fmt.Sprintf("iom/builtin")
		} else {
			packageName = fmt.Sprintf("iom/builtin/%s", fun.Package)
		}

		if _, exists := functionMap[packageName]; !exists {
			functionMap[packageName] = make(map[string]reflect.Value)
		}

		functionMap[packageName][fun.Name] = reflect.ValueOf(fun.Func)
	}

	return i
}

type GoMalPlugin struct {
	*DefaultPlugin
	Interpreter *interp.Interpreter
}

func (plug *GoMalPlugin) Run() error {
	_, err := plug.Interpreter.Eval(string(plug.Content))
	if err != nil {
		return err
	}
	return nil
}
