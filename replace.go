package main

import (
	"os"
	"strings"
)

func main() {
	files := []string{
		"internal/handlers/root.go",
		"internal/handlers/api.go",
		"internal/handlers/admin.go",
	}

	for _, file := range files {
		content, _ := os.ReadFile(file)
		str := string(content)
		str = strings.ReplaceAll(str, "Preload(\"Categories\")", "Preload(\"Pages\")")
		str = strings.ReplaceAll(str, "Association(\"Categories\")", "Association(\"Pages\")")
		os.WriteFile(file, []byte(str), 0644)
	}
}
