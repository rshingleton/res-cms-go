package main

import (
	"os"
	"strings"
)

func main() {
	files := []string{
		"cmd/res-cms/main.go",
		"internal/handlers/admin.go",
	}

	for _, file := range files {
		content, _ := os.ReadFile(file)
		str := string(content)
		str = strings.ReplaceAll(str, "\"toUpper\": func(s string) string { return strings.ToUpper(s) },", "\"toUpper\": func(s string) string { return strings.ToUpper(s) },\n\t\t\"safeHTML\": func(s string) template.HTML { return template.HTML(s) },")
		os.WriteFile(file, []byte(str), 0644)
	}
}
