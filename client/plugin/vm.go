package plugin

import (
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/chainreactors/mals"
	lua "github.com/yuin/gopher-lua"
	"github.com/yuin/gopher-lua/parse"
	"strings"
	"sync"
	"time"
)

func NewLuaVM() *lua.LState {
	vm := mals.NewLuaVM()
	mals.RegisterProtobufMessagesFromPackage(vm, "implantpb")
	mals.RegisterProtobufMessagesFromPackage(vm, "clientpb")
	mals.RegisterProtobufMessagesFromPackage(vm, "modulepb")
	vm.PreloadModule(intermediate.BeaconPackage, mals.PackageLoader(intermediate.InternalFunctions.Package(intermediate.BeaconPackage)))
	vm.PreloadModule(intermediate.RpcPackage, mals.PackageLoader(intermediate.InternalFunctions.Package(intermediate.RpcPackage)))
	for _, global := range globalMalManager.globalPlugins {
		vm.PreloadModule(global.Name, mals.GlobalLoader(global.Name, global.Path, global.Content))
	}

	// 注册所有内置函数
	for name, fun := range intermediate.InternalFunctions.Package(intermediate.BuiltinPackage) {
		vm.SetGlobal(name, vm.NewFunction(mals.WrapFuncForLua(fun)))
	}
	return vm
}

type LuaVMWrapper struct {
	*lua.LState
	initialized  bool
	lastUsedTime time.Time
	lock         sync.Mutex
}

func NewLuaVMWrapper() *LuaVMWrapper {
	return &LuaVMWrapper{
		LState:      NewLuaVM(),
		initialized: false,
	}
}

func (w *LuaVMWrapper) Lock() {
	w.lock.Lock()
	w.lastUsedTime = time.Now()
}

func (w *LuaVMWrapper) Unlock() {
	w.lastUsedTime = time.Now()
	w.lock.Unlock()
}

type LuaVMPool struct {
	vms        []*LuaVMWrapper
	maxSize    int
	lock       sync.Mutex
	proto      *lua.FunctionProto
	initScript string
	plugName   string
}

func NewLuaVMPool(maxSize int, initScript string, plugName string) (*LuaVMPool, error) {
	pool := &LuaVMPool{
		maxSize:    maxSize,
		vms:        make([]*LuaVMWrapper, 0, maxSize),
		initScript: initScript,
		plugName:   plugName,
	}

	// 预编译脚本
	reader := strings.NewReader(initScript)
	chunk, err := parse.Parse(reader, "script")
	if err != nil {
		return nil, fmt.Errorf("parse script error: %v", err)
	}
	proto, err := lua.Compile(chunk, "script")
	if err != nil {
		return nil, fmt.Errorf("compile script error: %v", err)
	}
	pool.proto = proto

	return pool, nil
}

func (p *LuaVMPool) AcquireVM() (*LuaVMWrapper, error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	for _, wrapper := range p.vms {
		if wrapper.lock.TryLock() {
			return wrapper, nil
		}
	}

	if len(p.vms) < p.maxSize {
		wrapper := NewLuaVMWrapper()
		wrapper.Lock()
		p.vms = append(p.vms, wrapper)
		return wrapper, nil
	}

	logs.Log.Warnf("VM pool is full, waiting for available VM...\n")
	p.lock.Unlock()

	for {
		p.lock.Lock()
		for _, wrapper := range p.vms {
			if wrapper.lock.TryLock() {
				return wrapper, nil
			}
		}
		p.lock.Unlock()
		time.Sleep(100 * time.Millisecond)
	}
}

func (p *LuaVMPool) ReleaseVM(wrapper *LuaVMWrapper) {
	wrapper.Unlock()
}

func (p *LuaVMPool) Destroy() {
	for _, wrapper := range p.vms {
		wrapper.Close()
	}
}
