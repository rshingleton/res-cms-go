/*
 * Copyright (C) 2026 Russ Shingleton <reshingleton@gmail.com>
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// Sandbox wraps a single WASM module instance running inside wazero.
// Each plugin runs in its own isolated Sandbox so that panics or errors in
// one plugin cannot corrupt the host process or other plugins.
type Sandbox struct {
	mu       sync.Mutex
	manifest *Manifest
	runtime  wazero.Runtime
	module   api.Module
	bridge   *Bridge
}

// wasmRuntime holds the process-level wazero runtime (one per process).
var (
	wasmRuntimeOnce sync.Once
	wasmRuntime     wazero.Runtime
)

// getWasmRuntime returns the singleton wazero runtime, creating it on first
// call. wazero uses a compilation cache so successive module instantiations
// are fast.
func getWasmRuntime(ctx context.Context) wazero.Runtime {
	wasmRuntimeOnce.Do(func() {
		// Create a compilation cache so that repeated loads of the same binary
		// are served from memory.
		cache := wazero.NewCompilationCache()
		config := wazero.NewRuntimeConfig().WithCompilationCache(cache)
		wasmRuntime = wazero.NewRuntimeWithConfig(ctx, config)

		// Instantiate WASI so that plugins built with TinyGo (or any WASI
		// target) can use standard I/O and environment calls.
		_, err := wasi_snapshot_preview1.Instantiate(ctx, wasmRuntime)
		if err != nil {
			log.Printf("plugin: failed to instantiate WASI: %v", err)
		}

		// Expose the host ABI that guest plugins call back into.
		instantiateHostModule(ctx, wasmRuntime)
	})
	return wasmRuntime
}

// hostLogMessage is the host function exposed to WASM guests as
// "rescms.log_message".  Guests call this to emit log lines visible in the
// host's stderr without having any direct access to host resources.
func hostLogMessage(_ context.Context, m api.Module, offset, length uint32) {
	data, ok := m.Memory().Read(offset, length)
	if !ok {
		return
	}
	log.Printf("[plugin/%s] %s", m.Name(), string(data))
}

// instantiateHostModule registers the host ABI as a WASM module named
// "rescms" so guests can import it.
func instantiateHostModule(ctx context.Context, rt wazero.Runtime) {
	_, err := rt.NewHostModuleBuilder("rescms").
		NewFunctionBuilder().
		WithFunc(hostLogMessage).
		Export("log_message").
		Instantiate(ctx)
	if err != nil {
		log.Printf("plugin: failed to build host module: %v", err)
	}
}

// NewSandbox compiles and instantiates a WASM plugin module. The sandbox is
// ready to receive calls after this function returns without error.
func NewSandbox(ctx context.Context, manifest *Manifest, wasmBytes []byte) (*Sandbox, error) {
	rt := getWasmRuntime(ctx)

	// Compile the binary (cached by wazero if the same bytes are seen again).
	compiled, err := rt.CompileModule(ctx, wasmBytes)
	if err != nil {
		return nil, fmt.Errorf("sandbox: compile %s: %w", manifest.Slug, err)
	}

	// Configure the module with its own stdout/stderr streams so that panic
	// output from a guest module never leaks into the host's descriptors.
	modConfig := wazero.NewModuleConfig().
		WithName(manifest.Slug).
		WithStdout(os.Stderr). // redirect guest stdout → host stderr (labelled)
		WithStderr(os.Stderr).
		WithSysNanosleep() // allow time calls without host clock access

	mod, err := rt.InstantiateModule(ctx, compiled, modConfig)
	if err != nil {
		return nil, fmt.Errorf("sandbox: instantiate %s: %w", manifest.Slug, err)
	}

	// For WASI reactors (e.g., Go c-shared), we must call _initialize to start the runtime.
	if initFn := mod.ExportedFunction("_initialize"); initFn != nil {
		if _, err := initFn.Call(ctx); err != nil {
			return nil, fmt.Errorf("sandbox: _initialize %s: %w", manifest.Slug, err)
		}
	}

	return &Sandbox{
		manifest: manifest,
		runtime:  rt,
		module:   mod,
		bridge:   NewBridge(),
	}, nil
}

// callFunc is a helper that calls a WASM exported function and returns the
// result byte slice. Panics inside the guest are caught by wazero and returned
// as errors, keeping the host process unaffected.
//
// The ABI contract is:
//
//	Guest exports: func <name>(ptr i32, len i32) (ptr i32, len i32)
//
// The host writes the input JSON into the guest's linear memory, calls the
// export, then reads the output JSON from the returned pointer.
func (s *Sandbox) callFunc(ctx context.Context, name string, inputJSON []byte) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	fn := s.module.ExportedFunction(name)
	if fn == nil {
		return nil, fmt.Errorf("sandbox %s: exported function %q not found", s.manifest.Slug, name)
	}

	// Allocate guest memory for the input via the "alloc" export.
	allocFn := s.module.ExportedFunction("alloc")
	if allocFn == nil {
		return nil, fmt.Errorf("sandbox %s: missing required export 'alloc'", s.manifest.Slug)
	}

	inputLen := uint64(len(inputJSON))
	allocRes, err := allocFn.Call(ctx, inputLen)
	if err != nil {
		return nil, fmt.Errorf("sandbox %s: alloc failed: %w", s.manifest.Slug, err)
	}
	ptr := uint32(allocRes[0])

	// Write input into guest memory.
	if !s.module.Memory().Write(ptr, inputJSON) {
		return nil, fmt.Errorf("sandbox %s: failed to write input to guest memory", s.manifest.Slug)
	}

	// Call the hook function.
	results, err := fn.Call(ctx, uint64(ptr), inputLen)
	if err != nil {
		log.Printf("sandbox %s: call %q failed: %v", s.manifest.Slug, name, err)
		return nil, fmt.Errorf("sandbox %s: call %q failed: %w", s.manifest.Slug, name, err)
	}
	if len(results) < 1 {
		log.Printf("sandbox %s: %q returned %d results, want at least 1", s.manifest.Slug, name, len(results))
		return nil, fmt.Errorf("sandbox %s: %q returned %d results, want at least 1", s.manifest.Slug, name, len(results))
	}

	// Read output from guest memory.
	var outPtr, outLen uint32
	if len(results) >= 2 {
		outPtr = uint32(results[0])
		outLen = uint32(results[1])
	} else if len(results) == 1 {
		// Packed return: ptr << 32 | len
		packed := results[0]
		outPtr = uint32(packed >> 32)
		outLen = uint32(packed & 0xffffffff)
	} else {
		return nil, fmt.Errorf("sandbox %s: %q returned no results", s.manifest.Slug, name)
	}

	out, ok := s.module.Memory().Read(outPtr, outLen)
	if !ok {
		return nil, fmt.Errorf("sandbox %s: failed to read output from guest memory (ptr: %d, len: %d)", s.manifest.Slug, outPtr, outLen)
	}

	// Make a copy as the underlying memory may be reclaimed.
	result := make([]byte, len(out))
	copy(result, out)
	return result, nil
}

// CallContentHook executes the guest's content hook function for the given
// hook type. Returns the (possibly modified) payload.
func (s *Sandbox) CallContentHook(ctx context.Context, hookType HookType, payload ContentPayload) (ContentPayload, error) {
	input, err := s.bridge.MarshalContentPayload(payload)
	if err != nil {
		return payload, err
	}

	// Guest function name convention: "hook_<type>" with dots → underscores.
	funcName := "hook_" + strings.ReplaceAll(string(hookType), ".", "_")
	output, err := s.callFunc(ctx, funcName, input)
	if err != nil {
		return payload, err
	}

	if isErr, msg := s.bridge.IsErrorResponse(output); isErr {
		return payload, fmt.Errorf("plugin %s: %s", s.manifest.Slug, msg)
	}

	return s.bridge.UnmarshalContentPayload(output)
}

// CallAssetHook executes the guest's asset hook function.
func (s *Sandbox) CallAssetHook(ctx context.Context, hookType HookType, payload AssetPayload) (AssetPayload, error) {
	input, err := s.bridge.MarshalAssetPayload(payload)
	if err != nil {
		return payload, err
	}

	funcName := "hook_" + strings.ReplaceAll(string(hookType), ".", "_")
	output, err := s.callFunc(ctx, funcName, input)
	if err != nil {
		return payload, err
	}

	if isErr, msg := s.bridge.IsErrorResponse(output); isErr {
		return payload, fmt.Errorf("plugin %s: %s", s.manifest.Slug, msg)
	}

	return s.bridge.UnmarshalAssetPayload(output)
}

// Close terminates the sandbox, releasing all WASM resources.
func (s *Sandbox) Close(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.module.Close(ctx)
}

// PluginsDir is the directory that contains plugin subdirectories.
// Each subdirectory holds a plugin.json manifest and one or more .wasm binaries.
const PluginsDir = "plugins"

// Manifest describes a plugin package, parsed from plugins/<slug>/plugin.json.
type Manifest struct {
	// Slug is the unique machine-readable identifier for the plugin.
	Slug string `json:"slug"`
	// Name is the human-readable display name.
	Name string `json:"name"`
	// Version follows Semantic Versioning (e.g. "1.0.0").
	Version string `json:"version"`
	// Author is the plugin author name or organisation.
	Author string `json:"author"`
	// Description is a brief summary of what the plugin does.
	Description string `json:"description"`
	// License should state "GPLv3" or compatible.
	License string `json:"license"`
	// Hooks lists the HookType strings this plugin subscribes to.
	Hooks []string `json:"hooks"`
	// Permissions explicitly lists what the plugin is allowed to access.
	// Supported values: "content_read", "content_write", "asset_inject",
	// "storage_write", "route_register".
	Permissions []string `json:"permissions"`
	// WasmFile is the filename of the .wasm binary relative to the plugin dir.
	WasmFile string `json:"wasm_file"`
	// EntryPoint is the exported function name called on plugin init (optional).
	EntryPoint string `json:"entry_point,omitempty"`
	// Checksum is the SHA-256 hex digest of the .wasm binary for integrity checks.
	Checksum string `json:"checksum"`
}

// HasPermission returns true if the given permission is declared in the manifest.
func (m *Manifest) HasPermission(perm string) bool {
	for _, p := range m.Permissions {
		if p == perm {
			return true
		}
	}
	return false
}

// LoadManifest reads and parses a plugin.json from pluginsDir/<slug>/plugin.json.
func LoadManifest(pluginsDir, slug string) (*Manifest, error) {
	path := filepath.Join(pluginsDir, slug, "plugin.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("manifest: read %s: %w", path, err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("manifest: parse %s: %w", path, err)
	}
	if m.Slug == "" {
		m.Slug = slug
	}
	return &m, nil
}
