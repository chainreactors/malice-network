//go:build !((linux && (386 || amd64)) || (darwin && (amd64 || arm64)) || (windows && amd64))

package traffic

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	wasi "github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// CreateTrafficEncoder - Initialize an WASM runtime using the provided module name, code, and log callback
func CreateTrafficEncoder(name string, wasm []byte, logger TrafficEncoderLogCallback) (*TrafficEncoder, error) {
	ctx := context.Background()
	wasmRuntime := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigInterpreter())

	// Build the runtime and expose helper functions
	_, err := wasmRuntime.NewHostModuleBuilder(name).

		// Rand function
		NewFunctionBuilder().WithFunc(func() uint64 {
		buf := make([]byte, 8)
		rand.Read(buf)
		return binary.LittleEndian.Uint64(buf)
	}).Export("rand").

		// Time function
		NewFunctionBuilder().WithFunc(func() int64 {
		return time.Now().UnixNano()
	}).Export("time").

		// Log function
		NewFunctionBuilder().WithFunc(func(_ context.Context, m api.Module, offset, byteCount uint32) {
		buf, ok := m.Memory().Read(offset, byteCount)
		if !ok {
			logger(fmt.Sprintf("Log error: Memory.Read(%d, %d) out of range", offset, byteCount))
		}
		logger(string(buf))
	}).Export("log").Instantiate(ctx)
	if err != nil {
		return nil, err
	}
	_, err = wasi.Instantiate(ctx, wasmRuntime)
	if err != nil {
		return nil, err
	}

	compiledMod, err := wasmRuntime.CompileModule(ctx, wasm)
	if err != nil {
		return nil, err
	}
	mod, err := wasmRuntime.InstantiateModule(ctx, compiledMod, wazero.NewModuleConfig())
	if err != nil {
		return nil, err
	}

	return &TrafficEncoder{
		ID: CalculateWasmEncoderID(wasm),
		// FileName: name, -- optionally set by caller
		Data: wasm,

		lock:    sync.Mutex{},
		ctx:     ctx,
		runtime: wasmRuntime,
		mod:     mod,

		encoder: mod.ExportedFunction("encode"),
		decoder: mod.ExportedFunction("decode"),

		// These are undocumented, but exported. See tinygo-org/tinygo#2788
		malloc: mod.ExportedFunction("malloc"),
		free:   mod.ExportedFunction("free"),
	}, nil
}
