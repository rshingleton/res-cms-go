package theme

import (
	"html/template"
	"os"
	"path/filepath"
	"testing"
)

func TestThemeEngine_LoadTheme(t *testing.T) {
	// Create a temporary themes directory
	tmpDir, err := os.MkdirTemp("", "themes-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	themeName := "test-theme"
	themeDir := filepath.Join(tmpDir, themeName)
	os.MkdirAll(filepath.Join(themeDir, "layouts"), 0755)
	os.MkdirAll(filepath.Join(themeDir, "partials"), 0755)

	// Create a dummy theme.json
	manifest := `{"name": "Test Theme", "version": "1.0.0"}`
	os.WriteFile(filepath.Join(themeDir, "theme.json"), []byte(manifest), 0644)

	// Create a dummy layout
	layout := `{{define "content"}}TestContent{{end}}`
	os.WriteFile(filepath.Join(themeDir, "layouts", "index.html"), []byte(layout), 0644)

	engine := NewEngine(tmpDir)
	funcs := template.FuncMap{}

	err = engine.LoadTheme(themeName, funcs)
	if err != nil {
		t.Fatalf("LoadTheme failed: %v", err)
	}

	if engine.Active != themeName {
		t.Errorf("Expected active theme %s, got %s", themeName, engine.Active)
	}

	if _, ok := engine.Templates["index.html"]; !ok {
		t.Errorf("index.html template not loaded")
	}
}

func TestThemeEngine_ValidateTheme(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "themes-val-*")
	defer os.RemoveAll(tmpDir)

	engine := NewEngine(tmpDir)
	themeDir := filepath.Join(tmpDir, "invalid-theme")
	os.MkdirAll(themeDir, 0755)

	err := engine.ValidateTheme(themeDir)
	if err == nil {
		t.Errorf("Expected validation error for missing files, got nil")
	}

	// Add missing files
	os.WriteFile(filepath.Join(themeDir, "theme.json"), []byte("{}"), 0644)
	os.MkdirAll(filepath.Join(themeDir, "layouts"), 0755)
	os.WriteFile(filepath.Join(themeDir, "layouts", "main.html"), []byte(""), 0644)
	os.WriteFile(filepath.Join(themeDir, "layouts", "index.html"), []byte(""), 0644)
	os.WriteFile(filepath.Join(themeDir, "layouts", "post.html"), []byte(""), 0644)
	os.WriteFile(filepath.Join(themeDir, "layouts", "page.html"), []byte(""), 0644)
	os.WriteFile(filepath.Join(themeDir, "layouts", "login.html"), []byte(""), 0644)
	os.WriteFile(filepath.Join(themeDir, "layouts", "profile.html"), []byte(""), 0644)

	err = engine.ValidateTheme(themeDir)
	if err != nil {
		t.Errorf("Expected validation to pass, got error: %v", err)
	}
}
