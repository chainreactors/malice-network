package plugin

import (
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/core/intermediate"
	lua "github.com/yuin/gopher-lua"
	"os"
	"path/filepath"
)

const (
	LuaScript = "lua"
	TCLScript = "tcl"
	CNAScript = "cna"
	GoPlugin  = "go"
)

func NewPlugin(manifest *MalManiFest) (*Plugin, error) {
	path := filepath.Join(assets.GetMalsDir(), manifest.Name)
	content, err := os.ReadFile(filepath.Join(path, manifest.EntryFile))
	if err != nil {
		return nil, err
	}

	plug := &Plugin{
		MalManiFest: manifest,
		Enable:      true,
		Content:     content,
		Path:        path,
	}

	return plug, nil
}

type Plugin struct {
	*MalManiFest
	Enable  bool
	Content []byte
	Path    string
}

func (plug *Plugin) RegisterLuaBuiltin(vm *lua.LState) error {
	plugDir := filepath.Join(assets.GetMalsDir(), plug.Name)
	vm.SetGlobal("plugin_dir", lua.LString(plugDir))
	vm.SetGlobal("plugin_resource_dir", lua.LString(filepath.Join(plugDir, "resources")))
	vm.SetGlobal("plugin_name", lua.LString(plug.Name))
	packageMod := vm.GetGlobal("package").(*lua.LTable)
	luaPath := lua.LuaPathDefault + ";" + plugDir + "\\?.lua"
	vm.SetField(packageMod, "path", lua.LString(luaPath))
	// 读取resource文件
	plug.registerLuaFunction(vm, "script_resource", func(filename string) (string, error) {
		return intermediate.GetResourceFile(plug.Name, filename)
	})

	plug.registerLuaFunction(vm, "find_resource", func(sess *core.Session, base string, ext string) (string, error) {
		return intermediate.FindResourceFile(plug.Name, base, sess.Os.Arch, ext)
	})

	// 读取资源文件内容
	plug.registerLuaFunction(vm, "read_resource", func(filename string) (string, error) {
		return intermediate.ReadResourceFile(plug.Name, filename)
	})
	return nil
}

func (plug *Plugin) registerLuaFunction(vm *lua.LState, name string, fn interface{}) {
	wrappedFunc := intermediate.WrapInternalFunc(fn)
	vm.SetGlobal(name, vm.NewFunction(intermediate.WrapFuncForLua(wrappedFunc)))
}
