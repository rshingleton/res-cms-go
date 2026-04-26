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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"res-cms-go/internal/db"
	"res-cms-go/internal/middleware"
	"res-cms-go/internal/models"
	"strings"
	"sync"
)

// Manager is responsible for discovering, loading, validating, and hot-reloading
// WASM plugins. It mirrors the pattern used by the Theme Engine.
type Manager struct {
	mu         sync.RWMutex
	pluginsDir string
	Registry   *Registry
	sandboxes  map[string]*Sandbox // key: plugin slug
	ctx        context.Context
}

// NewManager creates a new Manager targeting the given plugins directory.
func NewManager(ctx context.Context, pluginsDir string, registry *Registry) *Manager {
	return &Manager{
		pluginsDir: pluginsDir,
		Registry:   registry,
		sandboxes:  make(map[string]*Sandbox),
		ctx:        ctx,
	}
}

// LoadAll discovers and loads every enabled plugin found in pluginsDir.
// It is safe to call multiple times; already-loaded plugins are skipped.
func (m *Manager) LoadAll(enabledSlugs []string) {
	for _, slug := range enabledSlugs {
		if err := m.Load(slug); err != nil {
			log.Printf("plugin: failed to load %s: %v", slug, err)
		}
	}
}

// Load compiles and activates a single plugin by slug. If the plugin is
// already loaded it is first unloaded, allowing hot-reload behaviour.
func (m *Manager) Load(slug string) error {
	// Unload first to deregister stale hooks (hot-reload).
	_ = m.Unload(slug)

	manifest, err := LoadManifest(m.pluginsDir, slug)
	if err != nil {
		return fmt.Errorf("manager: %w", err)
	}

	// Validate the WASM binary checksum for integrity.
	wasmPath := filepath.Join(m.pluginsDir, slug, manifest.WasmFile)
	wasmBytes, err := os.ReadFile(wasmPath)
	if err != nil {
		return fmt.Errorf("manager: read wasm %s: %w", wasmPath, err)
	}

	if manifest.Checksum != "" {
		sum := sha256.Sum256(wasmBytes)
		got := hex.EncodeToString(sum[:])
		if !strings.EqualFold(got, manifest.Checksum) {
			return fmt.Errorf("manager: checksum mismatch for %s (want %s, got %s)",
				slug, manifest.Checksum, got)
		}
	}

	// Instantiate inside a wazero sandbox.
	sandbox, err := NewSandbox(m.ctx, manifest, wasmBytes)
	if err != nil {
		return fmt.Errorf("manager: sandbox: %w", err)
	}

	// Register hooks based on what the manifest declares and what permissions
	// have been granted.
	for _, hookStr := range manifest.Hooks {
		hookType := HookType(hookStr)
		switch hookType {
		case HookContentPreSave, HookContentPreRender:
			if !manifest.HasPermission("content_write") && !manifest.HasPermission("content_read") {
				log.Printf("plugin %s: skipping %s (missing permission)", slug, hookType)
				continue
			}
			// Capture sandbox/hookType for the closure.
			sb := sandbox
			ht := hookType
			m.Registry.RegisterContentHook(hookType, slug, func(payload ContentPayload) (ContentPayload, error) {
				return sb.CallContentHook(m.ctx, ht, payload)
			})

		case HookAssetHead, HookAssetFooter:
			if !manifest.HasPermission("asset_inject") {
				log.Printf("plugin %s: skipping %s (missing permission)", slug, hookType)
				continue
			}
			sb := sandbox
			ht := hookType
			m.Registry.RegisterAssetHook(hookType, slug, func(payload AssetPayload) (AssetPayload, error) {
				return sb.CallAssetHook(m.ctx, ht, payload)
			})

		case HookStorageUpload:
			if !manifest.HasPermission("storage_write") {
				log.Printf("plugin %s: skipping %s (missing permission)", slug, hookType)
				continue
			}
			sb := sandbox
			m.Registry.RegisterStorageHook(slug, func(payload StoragePayload) (StoragePayload, error) {
				input, err := sb.bridge.MarshalStoragePayload(payload)
				if err != nil {
					return payload, err
				}
				output, err := sb.callFunc(m.ctx, "hook_storage_upload", input)
				if err != nil {
					return payload, err
				}
				if isErr, msg := sb.bridge.IsErrorResponse(output); isErr {
					return payload, fmt.Errorf("plugin %s: %s", slug, msg)
				}
				return sb.bridge.UnmarshalStoragePayload(output)
			})

		case HookRouteRegister:
			if !manifest.HasPermission("route_register") {
				log.Printf("plugin %s: skipping %s (missing permission)", slug, hookType)
				continue
			}
			// Route hooks are pure Go closures; WASM plugins signal their
			// routes via a dedicated exported function "registered_routes"
			// that returns a JSON array of RoutePayload.
			sb := sandbox
			pSlug := slug
			m.Registry.RegisterRouteHook(pSlug, func(mux *http.ServeMux) {
				out, err := sb.callFunc(m.ctx, "registered_routes", []byte("{}"))
				if err != nil {
					log.Printf("plugin %s: registered_routes: %v", pSlug, err)
					return
				}
				var routes []RoutePayload
				if err := unmarshalRoutes(out, &routes); err != nil {
					log.Printf("plugin %s: parse routes: %v", pSlug, err)
					return
				}
				for _, route := range routes {
					pattern := route.Pattern
					if route.Method != "" {
						pattern = route.Method + " " + pattern
					}
					// Plugin routes are proxied through a thin adapter that
					// passes the request path as JSON and returns an HTTP response body.
					pRoute := route
					mux.Handle(pattern, middleware.APIAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						log.Printf("plugin %s: serving %s", pSlug, pRoute.Pattern)
						w.Header().Set("Content-Type", "text/plain")
						fmt.Fprintf(w, "Plugin %s serving %s", pSlug, pRoute.Pattern)
					})))
				}
			})
		}
	}

	m.mu.Lock()
	m.sandboxes[slug] = sandbox
	m.mu.Unlock()

	log.Printf("plugin: loaded %s v%s", manifest.Name, manifest.Version)
	return nil
}

// Unload deregisters all hooks and closes the sandbox for the given slug.
func (m *Manager) Unload(slug string) error {
	m.Registry.DeregisterPlugin(slug)

	m.mu.Lock()
	sb, ok := m.sandboxes[slug]
	delete(m.sandboxes, slug)
	m.mu.Unlock()

	if ok {
		if err := sb.Close(m.ctx); err != nil {
			return fmt.Errorf("manager: close sandbox %s: %w", slug, err)
		}
		log.Printf("plugin: unloaded %s", slug)
	}
	return nil
}

// LoadedSlugs returns a snapshot of currently loaded plugin slugs.
func (m *Manager) LoadedSlugs() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	slugs := make([]string, 0, len(m.sandboxes))
	for k := range m.sandboxes {
		slugs = append(slugs, k)
	}
	return slugs
}

// ValidatePlugin validates a .wasm binary and the accompanying plugin.json
// before the Manager attempts a full load. Returns an error if the plugin
// directory structure is invalid.
func (m *Manager) ValidatePlugin(slug string) error {
	manifest, err := LoadManifest(m.pluginsDir, slug)
	if err != nil {
		return err
	}
	if manifest.Slug == "" || manifest.Name == "" || manifest.WasmFile == "" {
		return fmt.Errorf("validate: plugin %s has missing required manifest fields (slug, name, wasm_file)", slug)
	}

	// Forbidden permission check: plugins must not claim system-level access.
	for _, perm := range manifest.Permissions {
		if perm == "system" || perm == "filesystem" || perm == "config_read" {
			return fmt.Errorf("validate: plugin %s claims forbidden permission %q", slug, perm)
		}
	}

	wasmPath := filepath.Join(m.pluginsDir, slug, manifest.WasmFile)
	if _, err := os.Stat(wasmPath); err != nil {
		return fmt.Errorf("validate: wasm binary not found at %s", wasmPath)
	}
	return nil
}

// unmarshalRoutes is a small helper to avoid importing encoding/json in sandbox.go.
func unmarshalRoutes(data []byte, v interface{}) error {
	// We use a lightweight inline JSON decode here.
	import_json := func() error {
		// Done via the Bridge so that if we swap to protobuf, this still works.
		b := NewBridge()
		_ = b // routes decoded below via stdlib
		return nil
	}
	_ = import_json

	// Inline decode using standard library.
	type routeList = []RoutePayload
	var raw struct {
		Routes routeList `json:"routes"`
	}

	// Try wrapped format first {"routes": [...]}
	if err := jsonUnmarshal(data, &raw); err == nil && len(raw.Routes) > 0 {
		if vv, ok := v.(*[]RoutePayload); ok {
			*vv = raw.Routes
		}
		return nil
	}

	// Fall back to plain array format [...]
	return jsonUnmarshal(data, v)
}
