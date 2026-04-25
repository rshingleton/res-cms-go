package theme

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type ThemeConfig struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Author      string `json:"author"`
	Description string `json:"description"`
	Config      struct {
		Colors     map[string]string `json:"colors"`
		Typography map[string]string `json:"typography"`
		BorderWidth string           `json:"border_width"`
	} `json:"config"`
}

type Engine struct {
	ThemesPath string
	Active     string
	Templates  map[string]*template.Template
	Funcs      template.FuncMap
}

func NewEngine(themesPath string) *Engine {
	return &Engine{
		ThemesPath: themesPath,
		Templates:  make(map[string]*template.Template),
	}
}

func (e *Engine) LoadTheme(name string, funcs template.FuncMap) error {
	e.Funcs = funcs
	themeDir := filepath.Join(e.ThemesPath, name)
	if _, err := os.Stat(themeDir); os.IsNotExist(err) {
		return fmt.Errorf("theme %s does not exist", name)
	}

	// Load manifest
	manifestPath := filepath.Join(themeDir, "theme.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read theme.json: %v", err)
	}

	var config ThemeConfig
	if err := json.Unmarshal(manifestData, &config); err != nil {
		return fmt.Errorf("failed to parse theme.json: %v", err)
	}

	// Base template for the theme
	base := template.New("").Funcs(funcs)

	// Load partials
	partialsDir := filepath.Join(themeDir, "partials")
	err = filepath.Walk(partialsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".html") {
			return err
		}
		name := "partials/" + filepath.Base(path)
		content, _ := os.ReadFile(path)
		_, err = base.New(name).Parse(string(content))
		return err
	})
	if err != nil {
		return err
	}

	// Load layouts
	layoutsDir := filepath.Join(themeDir, "layouts")
	
	// Check if a master layout exists
	masterPath := filepath.Join(layoutsDir, "main.html")
	if _, err := os.Stat(masterPath); err == nil {
		content, _ := os.ReadFile(masterPath)
		_, err = base.New("layouts/main.html").Parse(string(content))
		if err != nil {
			return err
		}
	}

	err = filepath.Walk(layoutsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".html") {
			return err
		}
		
		pageName := filepath.Base(path)
		if pageName == "main.html" {
			return nil // Already handled
		}

		content, _ := os.ReadFile(path)
		
		t, err := base.Clone()
		if err != nil {
			return err
		}
		
		_, err = t.New(pageName).Parse(string(content))
		if err != nil {
			return err
		}
		
		e.Templates[pageName] = t
		return nil
	})

	e.Active = name
	return nil
}

func (e *Engine) Reload() error {
	if e.Active == "" {
		return errors.New("no active theme to reload")
	}
	return e.LoadTheme(e.Active, e.Funcs)
}

func (e *Engine) ValidateTheme(path string) error {
	// Check for required files
	required := []string{"theme.json", "layouts/index.html", "layouts/post.html"}
	for _, f := range required {
		if _, err := os.Stat(filepath.Join(path, f)); os.IsNotExist(err) {
			return fmt.Errorf("missing required file: %s", f)
		}
	}
	return nil
}

func (e *Engine) ExtractTheme(zipPath string) (string, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	// Find the theme name (assume the first directory or a common prefix)
	var themeName string
	if len(r.File) > 0 {
		parts := strings.Split(r.File[0].Name, "/")
		if len(parts) > 0 {
			themeName = parts[0]
		}
	}

	if themeName == "" {
		return "", errors.New("could not determine theme name from zip")
	}

	extractPath := filepath.Join(e.ThemesPath, themeName)
	os.RemoveAll(extractPath) // Clean up if exists

	for _, f := range r.File {
		fpath := filepath.Join(e.ThemesPath, f.Name)
		if !strings.HasPrefix(fpath, filepath.Clean(e.ThemesPath)+string(os.PathSeparator)) {
			return "", fmt.Errorf("illegal file path: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return "", err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return "", err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return "", err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return "", err
		}
	}

	// Validate after extraction
	if err := e.ValidateTheme(extractPath); err != nil {
		os.RemoveAll(extractPath)
		return "", err
	}

	return themeName, nil
}

func (e *Engine) ExportTheme(name string, target io.Writer) error {
	themeDir := filepath.Join(e.ThemesPath, name)
	if _, err := os.Stat(themeDir); os.IsNotExist(err) {
		return fmt.Errorf("theme %s does not exist", name)
	}

	zw := zip.NewWriter(target)
	defer zw.Close()

	return filepath.Walk(themeDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(e.ThemesPath, path)
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = relPath

		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})
}
