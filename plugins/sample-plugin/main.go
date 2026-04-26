package main

import (
	"encoding/json"
	"unsafe"
)

// ContentPayload matches the host structure.
type ContentPayload struct {
	ID      uint   `json:"id"`
	Type    string `json:"type"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Slug    string `json:"slug"`
}

// AssetPayload matches the host structure.
type AssetPayload struct {
	CSS  string `json:"css"`
	JS   string `json:"js"`
	HTML string `json:"html"`
}

// RoutePayload matches the host structure.
type RoutePayload struct {
	Pattern string `json:"pattern"`
	Method  string `json:"method"`
}

//go:wasmexport alloc
func alloc(size uint32) uint32 {
	buf := make([]byte, size)
	ptr := uint32(uintptr(unsafe.Pointer(&buf[0])))
	return ptr
}

// helper to read from memory
func ptrToBytes(ptr uint32, size uint32) []byte {
	return unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), size)
}

// helper to return bytes
func returnBytes(b []byte) uint64 {
	if len(b) == 0 {
		return 0
	}
	ptr := uint64(uintptr(unsafe.Pointer(&b[0])))
	length := uint64(len(b))
	return (ptr << 32) | (length & 0xffffffff)
}

//go:wasmexport hook_content_pre_render
func hook_content_pre_render(ptr uint32, size uint32) uint64 {
	data := ptrToBytes(ptr, size)
	
	var payload ContentPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return returnBytes([]byte(`{"error":"` + err.Error() + `"}`))
	}

	// Modify content
	payload.Content += "\n\n<p><em>*This post was enhanced by the WASM Sample Plugin.*</em></p>"

	out, _ := json.Marshal(payload)
	return returnBytes(out)
}

//go:wasmexport hook_asset_footer
func hook_asset_footer(ptr uint32, size uint32) uint64 {
	data := ptrToBytes(ptr, size)

	var payload AssetPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return returnBytes([]byte(`{"error":"` + err.Error() + `"}`))
	}

	payload.HTML += "\n<!-- Injected by WASM Sample Plugin -->\n"
	
	out, _ := json.Marshal(payload)
	return returnBytes(out)
}

//go:wasmexport registered_routes
func registered_routes(ptr uint32, size uint32) uint64 {
	routes := []RoutePayload{
		{Pattern: "/plugin/sample/status", Method: "GET"},
	}
	
	// The host expects { "routes": [...] } or just an array
	out, _ := json.Marshal(routes)
	return returnBytes(out)
}

func main() {
	// The main function is required for go build to work, but it's not actually run
	// by wazero because wazero just loads the module and calls the exported functions.
}
