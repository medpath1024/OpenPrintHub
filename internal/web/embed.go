package web

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

//go:embed templates/*.html
var templatesFS embed.FS

//go:embed static/*
var staticFS embed.FS

// Templates holds the parsed templates
var Templates *template.Template

// titleCase converts a string to title case using the proper x/text/cases package
func titleCase(s string) string {
	return cases.Title(language.English).String(s)
}

func init() {
	// Parse all templates
	var err error
	Templates, err = template.New("").Funcs(template.FuncMap{
		"lower": strings.ToLower,
		"title": titleCase,
	}).ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		panic("failed to parse templates: " + err.Error())
	}
}

// ServeStatic serves static files from the embedded filesystem
func ServeStatic(w http.ResponseWriter, r *http.Request) {
	// Remove /static/ prefix
	path := strings.TrimPrefix(r.URL.Path, "/static/")

	// Read file from embedded FS
	content, err := fs.ReadFile(staticFS, "static/"+path)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Set content type based on extension
	contentType := "application/octet-stream"
	switch {
	case strings.HasSuffix(path, ".css"):
		contentType = "text/css; charset=utf-8"
	case strings.HasSuffix(path, ".js"):
		contentType = "application/javascript; charset=utf-8"
	case strings.HasSuffix(path, ".html"):
		contentType = "text/html; charset=utf-8"
	case strings.HasSuffix(path, ".json"):
		contentType = "application/json"
	case strings.HasSuffix(path, ".png"):
		contentType = "image/png"
	case strings.HasSuffix(path, ".svg"):
		contentType = "image/svg+xml"
	case strings.HasSuffix(path, ".ico"):
		contentType = "image/x-icon"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=31536000")
	_, _ = w.Write(content)
}

// GetStaticFS returns the static filesystem for direct access
func GetStaticFS() embed.FS {
	return staticFS
}

// GetTemplatesFS returns the templates filesystem for direct access
func GetTemplatesFS() embed.FS {
	return templatesFS
}
