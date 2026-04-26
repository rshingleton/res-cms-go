# WASM Plugin Specification (ResCMS)

ResCMS supports a robust, sandboxed plugin system based on **WebAssembly (WASM)**. This allows the core CMS to be extended with multi-language plugins (Go, Rust, AssemblyScript, etc.) while maintaining a single-binary architecture and high security.

## Core Runtime
The framework uses [wazero](https://github.com/tetratelabs/wazero), a zero-dependency WebAssembly runtime for Go.

- **Isolation**: Each plugin runs in its own isolated sandbox.
- **WASI Support**: Plugins are compiled as WASI (WebAssembly System Interface) reactors.
- **Memory Safety**: Plugins communicate via a JSON-based memory bridge.

## Developing a Plugin (Go)

### 1. Requirements
- Go 1.24 or higher
- The plugin must be a `main` package.

### 2. ABI Contract
Plugins communicate with the host via several exported functions. Because standard Go WASM currently supports a single return value for exports, ResCMS uses a **Packed i64 Return** strategy:
- `i64 return value` = `(pointer << 32) | length`

#### Required Export: `alloc`
The host calls `alloc` to reserve memory in the guest sandbox for passing JSON payloads.
```go
//go:wasmexport alloc
func alloc(size uint32) uint32 {
	buf := make([]byte, size)
	return uint32(uintptr(unsafe.Pointer(&buf[0])))
}
```

### 3. Hook Types
Plugins register for specific lifecycle hooks in their `plugin.json`.

| Hook Type | WASM Export Name | Purpose |
|-----------|------------------|---------|
| `content.pre_save` | `hook_content_pre_save` | Modify post/page content before DB save. |
| `content.pre_render` | `hook_content_pre_render` | Modify content before it reaches the template. |
| `asset.head` | `hook_asset_head` | Inject CSS/JS into the `<head>`. |
| `asset.footer` | `hook_asset_footer` | Inject HTML/JS before `</body>`. |
| `route.register` | `registered_routes` | Register custom HTTP API endpoints. |

### 4. Sample Hook Implementation
```go
//go:wasmexport hook_content_pre_render
func hook_content_pre_render(ptr uint32, size uint32) uint64 {
	// 1. Read input JSON from memory
	data := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), size)
	
	// 2. Parse payload
	var payload ContentPayload
	json.Unmarshal(data, &payload)

	// 3. Modify data
	payload.Content += "\n<p>Enhanced by WASM</p>"

	// 4. Return packed pointer/length
	out, _ := json.Marshal(payload)
	return (uint64(uintptr(unsafe.Pointer(&out[0]))) << 32) | uint64(len(out))
}
```

## Plugin Manifest (`plugin.json`)
Every plugin must reside in its own folder under `/plugins` and contain a `plugin.json`.

```json
{
    "slug": "my-plugin",
    "name": "My Extension",
    "version": "1.0.0",
    "hooks": ["content.pre_render", "asset.footer"],
    "permissions": ["content_read", "content_write", "asset_inject"],
    "wasm_file": "plugin.wasm",
    "checksum": "sha256_of_the_wasm_file"
}
```

## Security & Permissions
Plugins must explicitly declare permissions in their manifest. The `PluginManager` validates these during load:
- `content_read` / `content_write`
- `asset_inject`
- `route_register`
- `storage_write`

Any attempt to use an undeclared hook or perform unauthorized operations will be blocked by the host.
