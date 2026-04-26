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

// Package plugin provides the WASM plugin framework for ResCMS.
// It implements a sandboxed execution environment for WebAssembly plugins,
// allowing secure, multi-language extensibility while maintaining a
// single-binary architecture.
package plugin

import (
	"net/http"
	"sync"
)

// HookType identifies which lifecycle event a hook targets.
type HookType string

const (
	// HookContentPreSave is fired before post/page content is persisted.
	HookContentPreSave HookType = "content.pre_save"
	// HookContentPreRender is fired before content is sent to the template engine.
	HookContentPreRender HookType = "content.pre_render"
	// HookAssetHead is fired to inject CSS/JS into the <head>.
	HookAssetHead HookType = "asset.head"
	// HookAssetFooter is fired to inject assets before </body>.
	HookAssetFooter HookType = "asset.footer"
	// HookStorageUpload is fired when a file is about to be stored.
	HookStorageUpload HookType = "storage.upload"
	// HookRouteRegister signals that a plugin wants to register HTTP routes.
	HookRouteRegister HookType = "route.register"
)

// ContentPayload carries post/page data through content hooks.
type ContentPayload struct {
	ID      uint   `json:"id"`
	Type    string `json:"type"` // "post" or "page"
	Title   string `json:"title"`
	Content string `json:"content"`
	Slug    string `json:"slug"`
}

// AssetPayload carries injected asset strings through asset hooks.
type AssetPayload struct {
	CSS  string `json:"css"`
	JS   string `json:"js"`
	HTML string `json:"html"`
}

// StoragePayload carries file information through storage hooks.
type StoragePayload struct {
	OriginalName string `json:"original_name"`
	MimeType     string `json:"mime_type"`
	Data         []byte `json:"data"`
	// URL is populated by plugins that redirect storage to a cloud provider.
	URL string `json:"url,omitempty"`
}

// RoutePayload carries route registration information from plugins.
type RoutePayload struct {
	// Pattern is the URL path pattern to register (e.g. "/plugin/my-plugin/dashboard").
	Pattern string `json:"pattern"`
	// Method is the HTTP method (GET, POST, etc.). Empty means all methods.
	Method string `json:"method"`
}

// ContentHookFn is the function signature for content lifecycle hooks.
// The function receives a payload, may modify it, and returns the (possibly
// mutated) payload.
type ContentHookFn func(payload ContentPayload) (ContentPayload, error)

// AssetHookFn appends CSS/JS/HTML fragments to the site output.
type AssetHookFn func(payload AssetPayload) (AssetPayload, error)

// StorageHookFn intercepts file storage. Return a non-empty URL to redirect
// storage to a cloud provider instead of local disk.
type StorageHookFn func(payload StoragePayload) (StoragePayload, error)

// RouteHookFn registers HTTP handler functions via the provided mux.
type RouteHookFn func(mux *http.ServeMux)

// hookEntry wraps a hook function together with its owning plugin slug.
type contentHookEntry struct {
	pluginSlug string
	fn         ContentHookFn
}

type assetHookEntry struct {
	pluginSlug string
	fn         AssetHookFn
}

type storageHookEntry struct {
	pluginSlug string
	fn         StorageHookFn
}

type routeHookEntry struct {
	pluginSlug string
	fn         RouteHookFn
}

// Registry is the central hook registry for all loaded plugins.
// It is safe for concurrent use.
type Registry struct {
	mu           sync.RWMutex
	contentHooks map[HookType][]contentHookEntry
	assetHooks   map[HookType][]assetHookEntry
	storageHooks []storageHookEntry
	routeHooks   []routeHookEntry
}

// NewRegistry creates and returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		contentHooks: make(map[HookType][]contentHookEntry),
		assetHooks:   make(map[HookType][]assetHookEntry),
	}
}

// RegisterContentHook subscribes a function to a content lifecycle hook point.
func (r *Registry) RegisterContentHook(hookType HookType, pluginSlug string, fn ContentHookFn) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.contentHooks[hookType] = append(r.contentHooks[hookType], contentHookEntry{
		pluginSlug: pluginSlug,
		fn:         fn,
	})
}

// RegisterAssetHook subscribes a function to an asset injection hook point.
func (r *Registry) RegisterAssetHook(hookType HookType, pluginSlug string, fn AssetHookFn) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.assetHooks[hookType] = append(r.assetHooks[hookType], assetHookEntry{
		pluginSlug: pluginSlug,
		fn:         fn,
	})
}

// RegisterStorageHook subscribes a function to the storage upload hook.
func (r *Registry) RegisterStorageHook(pluginSlug string, fn StorageHookFn) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.storageHooks = append(r.storageHooks, storageHookEntry{
		pluginSlug: pluginSlug,
		fn:         fn,
	})
}

// RegisterRouteHook subscribes a function that will register HTTP routes.
func (r *Registry) RegisterRouteHook(pluginSlug string, fn RouteHookFn) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.routeHooks = append(r.routeHooks, routeHookEntry{
		pluginSlug: pluginSlug,
		fn:         fn,
	})
}

// DeregisterPlugin removes all hooks registered by the given plugin slug.
// This is called when a plugin is deactivated or unloaded.
func (r *Registry) DeregisterPlugin(slug string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for hookType, entries := range r.contentHooks {
		filtered := entries[:0]
		for _, e := range entries {
			if e.pluginSlug != slug {
				filtered = append(filtered, e)
			}
		}
		r.contentHooks[hookType] = filtered
	}

	for hookType, entries := range r.assetHooks {
		filtered := entries[:0]
		for _, e := range entries {
			if e.pluginSlug != slug {
				filtered = append(filtered, e)
			}
		}
		r.assetHooks[hookType] = filtered
	}

	storageFiltered := r.storageHooks[:0]
	for _, e := range r.storageHooks {
		if e.pluginSlug != slug {
			storageFiltered = append(storageFiltered, e)
		}
	}
	r.storageHooks = storageFiltered

	routeFiltered := r.routeHooks[:0]
	for _, e := range r.routeHooks {
		if e.pluginSlug != slug {
			routeFiltered = append(routeFiltered, e)
		}
	}
	r.routeHooks = routeFiltered
}

// FireContentHook executes all registered hooks for the given HookType in
// registration order, piping the payload through each handler.
func (r *Registry) FireContentHook(hookType HookType, payload ContentPayload) (ContentPayload, error) {
	r.mu.RLock()
	entries := make([]contentHookEntry, len(r.contentHooks[hookType]))
	copy(entries, r.contentHooks[hookType])
	r.mu.RUnlock()

	var err error
	for _, entry := range entries {
		payload, err = entry.fn(payload)
		if err != nil {
			return payload, err
		}
	}
	return payload, nil
}

// FireAssetHook executes all registered asset hooks, collecting injections.
func (r *Registry) FireAssetHook(hookType HookType, payload AssetPayload) (AssetPayload, error) {
	r.mu.RLock()
	entries := make([]assetHookEntry, len(r.assetHooks[hookType]))
	copy(entries, r.assetHooks[hookType])
	r.mu.RUnlock()

	var err error
	for _, entry := range entries {
		payload, err = entry.fn(payload)
		if err != nil {
			return payload, err
		}
	}
	return payload, nil
}

// FireStorageHook runs the first registered storage hook. Storage hooks are
// exclusive: only the first hook that sets a non-empty URL is used, allowing
// a single cloud-storage provider to intercept uploads.
func (r *Registry) FireStorageHook(payload StoragePayload) (StoragePayload, error) {
	r.mu.RLock()
	entries := make([]storageHookEntry, len(r.storageHooks))
	copy(entries, r.storageHooks)
	r.mu.RUnlock()

	for _, entry := range entries {
		result, err := entry.fn(payload)
		if err != nil {
			return payload, err
		}
		if result.URL != "" {
			return result, nil
		}
		payload = result
	}
	return payload, nil
}

// ApplyRouteHooks calls all registered route hook functions so that plugins
// can mount their own HTTP handlers on the provided mux.
func (r *Registry) ApplyRouteHooks(mux *http.ServeMux) {
	r.mu.RLock()
	entries := make([]routeHookEntry, len(r.routeHooks))
	copy(entries, r.routeHooks)
	r.mu.RUnlock()

	for _, entry := range entries {
		entry.fn(mux)
	}
}
