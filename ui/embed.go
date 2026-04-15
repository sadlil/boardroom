package ui

import (
	"bytes"
	"embed"
	"html/template"
	"io"
	"io/fs"
	"net/http"
)

//go:embed *.html styles.css
var content embed.FS

//go:embed templates/*.html
var templateContent embed.FS

// Assets is the HTTP filesystem for the embedded static content
var Assets http.FileSystem

// Templates holds all parsed HTML templates
var Templates *template.Template

func init() {
	// Static asset filesystem
	sys, err := fs.Sub(content, ".")
	if err != nil {
		panic(err)
	}
	Assets = http.FS(sys)

	// Parse all HTML templates with custom functions
	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
	}
	Templates, err = template.New("").Funcs(funcMap).ParseFS(templateContent, "templates/*.html")
	if err != nil {
		panic("failed to parse templates: " + err.Error())
	}
}

// Render executes a named template with the given data and writes the result to w.
func Render(w io.Writer, name string, data any) error {
	return Templates.ExecuteTemplate(w, name, data)
}

// RenderToString executes a named template and returns the result as a string.
func RenderToString(name string, data any) (string, error) {
	var buf bytes.Buffer
	if err := Templates.ExecuteTemplate(&buf, name, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
