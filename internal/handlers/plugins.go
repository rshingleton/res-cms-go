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

package handlers

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"res-cms-go/internal/db"
	"res-cms-go/internal/middleware"
	"res-cms-go/internal/models"
	"res-cms-go/internal/plugin"
	"strings"
)

// PluginManager is the process-wide plugin.Manager instance, initialised in main.go.
var PluginManager *plugin.Manager

// AdminListPluginsHandler renders the Plugin Manager dashboard page.
func AdminListPluginsHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/manage/plugins" {
		http.NotFound(w, r)
		return
	}

	user := middleware.RequireUser(r)
	if user == nil || !user.IsAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var plugins []models.Plugin
	db.DB.Order("name ASC").Find(&plugins)

	// Collect slugs of plugins that are currently live in the WASM runtime.
	loadedSet := make(map[string]bool)
	if PluginManager != nil {
		for _, s := range PluginManager.LoadedSlugs() {
			loadedSet[s] = true
		}
	}

	type PluginView struct {
		models.Plugin
		IsLoaded    bool
		Permissions []string
		Hooks       []string
	}

	var views []PluginView
	for _, p := range plugins {
		var perms, hooks []string
		_ = json.Unmarshal([]byte(p.Permissions), &perms)
		_ = json.Unmarshal([]byte(p.Hooks), &hooks)
		views = append(views, PluginView{
			Plugin:      p,
			IsLoaded:    loadedSet[p.Slug],
			Permissions: perms,
			Hooks:       hooks,
		})
	}

	data := map[string]interface{}{
		"BlogName":  getBlogName(),
		"User":      user,
		"Plugins":   views,
		"ActiveTab": "plugins",
	}

	if err := renderTemplate(w, r, "admin/plugins/list.html", data); err != nil {
		log.Printf("Error rendering plugins template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// AdminUploadPluginHandler handles .wasm or .zip plugin uploads.
// Uploaded archives must contain:
//
//	<slug>/
//	  plugin.json
//	  <wasm_file>.wasm
func AdminUploadPluginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	user := middleware.RequireUser(r)
	if user == nil || !user.IsAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("plugin_file")
	if err != nil {
		middleware.GenerateFlashCookie(w, "No file uploaded")
		http.Redirect(w, r, "/manage/plugins", http.StatusFound)
		return
	}
	defer file.Close()

	// Create a temp file to buffer the upload.
	tmp, err := os.CreateTemp("", "rescms-plugin-*.zip")
	if err != nil {
		middleware.GenerateFlashCookie(w, "Server error: could not create temp file")
		http.Redirect(w, r, "/manage/plugins", http.StatusFound)
		return
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	if _, err := io.Copy(tmp, file); err != nil {
		middleware.GenerateFlashCookie(w, "Upload failed")
		http.Redirect(w, r, "/manage/plugins", http.StatusFound)
		return
	}

	pluginsDir := plugin.PluginsDir
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		middleware.GenerateFlashCookie(w, "Server error: cannot create plugins directory")
		http.Redirect(w, r, "/manage/plugins", http.StatusFound)
		return
	}

	var slug string

	ext := strings.ToLower(filepath.Ext(header.Filename))
	switch ext {
	case ".zip":
		slug, err = extractPluginZip(tmp.Name(), pluginsDir)
		if err != nil {
			middleware.GenerateFlashCookie(w, "Invalid plugin archive: "+err.Error())
			http.Redirect(w, r, "/manage/plugins", http.StatusFound)
			return
		}
	default:
		middleware.GenerateFlashCookie(w, "Unsupported file type. Please upload a .zip plugin archive.")
		http.Redirect(w, r, "/manage/plugins", http.StatusFound)
		return
	}

	// Validate the extracted plugin structure.
	if PluginManager == nil {
		middleware.GenerateFlashCookie(w, "Plugin manager not initialised")
		http.Redirect(w, r, "/manage/plugins", http.StatusFound)
		return
	}

	if err := PluginManager.ValidatePlugin(slug); err != nil {
		os.RemoveAll(filepath.Join(pluginsDir, slug))
		middleware.GenerateFlashCookie(w, "Validation failed: "+err.Error())
		http.Redirect(w, r, "/manage/plugins", http.StatusFound)
		return
	}

	// Load and parse the manifest to populate the DB record.
	manifest, err := plugin.LoadManifest(pluginsDir, slug)
	if err != nil {
		middleware.GenerateFlashCookie(w, "Cannot read manifest: "+err.Error())
		http.Redirect(w, r, "/manage/plugins", http.StatusFound)
		return
	}

	// Compute SHA-256 checksum of the WASM binary.
	wasmPath := filepath.Join(pluginsDir, slug, manifest.WasmFile)
	checksum, err := sha256File(wasmPath)
	if err != nil {
		middleware.GenerateFlashCookie(w, "Checksum error: "+err.Error())
		http.Redirect(w, r, "/manage/plugins", http.StatusFound)
		return
	}

	permsJSON, _ := json.Marshal(manifest.Permissions)
	hooksJSON, _ := json.Marshal(manifest.Hooks)

	record := models.Plugin{
		Slug:        manifest.Slug,
		Name:        manifest.Name,
		Version:     manifest.Version,
		Author:      manifest.Author,
		Description: manifest.Description,
		License:     manifest.License,
		WasmFile:    manifest.WasmFile,
		Checksum:    checksum,
		Permissions: string(permsJSON),
		Hooks:       string(hooksJSON),
		Enabled:     false,
	}

	// Upsert the plugin record.
	result := db.DB.Where(models.Plugin{Slug: slug}).FirstOrCreate(&record)
	if result.Error != nil {
		middleware.GenerateFlashCookie(w, "Database error: "+result.Error.Error())
		http.Redirect(w, r, "/manage/plugins", http.StatusFound)
		return
	}

	// If it already existed, update the fields from the new manifest.
	if result.RowsAffected == 0 {
		db.DB.Model(&record).Updates(map[string]interface{}{
			"name":        manifest.Name,
			"version":     manifest.Version,
			"author":      manifest.Author,
			"description": manifest.Description,
			"wasm_file":   manifest.WasmFile,
			"checksum":    checksum,
			"permissions": string(permsJSON),
			"hooks":       string(hooksJSON),
		})
	}

	middleware.GenerateFlashCookie(w, fmt.Sprintf("Plugin '%s' v%s installed. Enable it to activate.", manifest.Name, manifest.Version))
	http.Redirect(w, r, "/manage/plugins", http.StatusFound)
}

// AdminTogglePluginHandler enables or disables a plugin by slug.
// Enabling a plugin loads it into the WASM runtime immediately (hot-load).
// Disabling unloads it without a server restart.
func AdminTogglePluginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	user := middleware.RequireUser(r)
	if user == nil || !user.IsAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	slug := strings.TrimPrefix(r.URL.Path, "/manage/plugins/toggle/")
	if slug == "" {
		http.NotFound(w, r)
		return
	}

	var record models.Plugin
	if err := db.DB.Where("slug = ?", slug).First(&record).Error; err != nil {
		middleware.GenerateFlashCookie(w, "Plugin not found")
		http.Redirect(w, r, "/manage/plugins", http.StatusFound)
		return
	}

	newEnabled := !record.Enabled
	if err := db.DB.Model(&record).Update("enabled", newEnabled).Error; err != nil {
		middleware.GenerateFlashCookie(w, "Database error: "+err.Error())
		http.Redirect(w, r, "/manage/plugins", http.StatusFound)
		return
	}

	if PluginManager != nil {
		if newEnabled {
			if err := PluginManager.Load(slug); err != nil {
				middleware.GenerateFlashCookie(w, "Plugin enabled in DB but WASM load failed: "+err.Error())
				http.Redirect(w, r, "/manage/plugins", http.StatusFound)
				return
			}
			middleware.GenerateFlashCookie(w, "Plugin '"+record.Name+"' enabled and loaded.")
		} else {
			_ = PluginManager.Unload(slug)
			middleware.GenerateFlashCookie(w, "Plugin '"+record.Name+"' disabled and unloaded.")
		}
	}

	http.Redirect(w, r, "/manage/plugins", http.StatusFound)
}

// AdminDeletePluginHandler removes a plugin's files and database record.
func AdminDeletePluginHandler(w http.ResponseWriter, r *http.Request) {
	user := middleware.RequireUser(r)
	if user == nil || !user.IsAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	slug := strings.TrimPrefix(r.URL.Path, "/manage/plugins/delete/")
	if slug == "" {
		http.NotFound(w, r)
		return
	}

	// Unload from runtime first.
	if PluginManager != nil {
		_ = PluginManager.Unload(slug)
	}

	// Remove files.
	pluginDir := filepath.Join(plugin.PluginsDir, slug)
	os.RemoveAll(pluginDir)

	// Remove DB record.
	if err := db.DB.Where("slug = ?", slug).Delete(&models.Plugin{}).Error; err != nil {
		middleware.GenerateFlashCookie(w, "Failed to delete plugin record: "+err.Error())
	} else {
		middleware.GenerateFlashCookie(w, "Plugin '"+slug+"' removed.")
	}

	http.Redirect(w, r, "/manage/plugins", http.StatusFound)
}

// AdminReloadPluginHandler hot-reloads a plugin without a server restart.
func AdminReloadPluginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	user := middleware.RequireUser(r)
	if user == nil || !user.IsAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	slug := strings.TrimPrefix(r.URL.Path, "/manage/plugins/reload/")
	if slug == "" {
		http.NotFound(w, r)
		return
	}

	if PluginManager == nil {
		middleware.GenerateFlashCookie(w, "Plugin manager not initialised")
		http.Redirect(w, r, "/manage/plugins", http.StatusFound)
		return
	}

	if err := PluginManager.Load(slug); err != nil {
		middleware.GenerateFlashCookie(w, "Reload failed: "+err.Error())
	} else {
		middleware.GenerateFlashCookie(w, "Plugin '"+slug+"' reloaded.")
	}

	http.Redirect(w, r, "/manage/plugins", http.StatusFound)
}

// ─── helpers ────────────────────────────────────────────────────────────────

// extractPluginZip extracts a plugin zip archive to pluginsDir, validates the
// directory structure, and returns the plugin slug.
func extractPluginZip(zipPath, pluginsDir string) (string, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", fmt.Errorf("cannot open zip: %w", err)
	}
	defer r.Close()

	if len(r.File) == 0 {
		return "", fmt.Errorf("empty archive")
	}

	// Detect slug from the first directory entry.
	var slug string
	for _, f := range r.File {
		parts := strings.SplitN(filepath.ToSlash(f.Name), "/", 2)
		if parts[0] != "" {
			slug = parts[0]
			break
		}
	}
	if slug == "" {
		return "", fmt.Errorf("cannot determine plugin slug from archive")
	}

	dest := filepath.Join(pluginsDir, slug)
	os.RemoveAll(dest) // clean up any previous version

	for _, f := range r.File {
		fpath := filepath.Join(pluginsDir, filepath.FromSlash(f.Name))
		// Path traversal guard.
		if !strings.HasPrefix(filepath.Clean(fpath), filepath.Clean(pluginsDir)+string(os.PathSeparator)) {
			return "", fmt.Errorf("illegal path in archive: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, 0755)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			return "", err
		}

		out, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return "", err
		}
		rc, err := f.Open()
		if err != nil {
			out.Close()
			return "", err
		}
		_, copyErr := io.Copy(out, rc)
		out.Close()
		rc.Close()
		if copyErr != nil {
			return "", copyErr
		}
	}

	return slug, nil
}

// sha256File returns the hex-encoded SHA-256 digest of the file at path.
func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
