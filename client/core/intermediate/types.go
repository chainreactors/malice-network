package intermediate

import "fmt"

var InternalFunctions = make(map[string]InternalFunc)

type InternalFunc func(...interface{}) (interface{}, error)

func RegisterInternalFunc(name string, fn InternalFunc) error {
	if _, ok := InternalFunctions[name]; ok {
		return fmt.Errorf("function %s already registered", name)
	}

	InternalFunctions[name] = fn
	return nil
}
