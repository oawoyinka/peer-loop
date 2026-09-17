package handlers

import (
	"html/template"
	"log"
	"net/http"
)

// render executes the named template within the shared layout and writes
// it to w, logging (rather than half-writing broken HTML) on failure.
func render(w http.ResponseWriter, tmpl *template.Template, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("template error rendering %s: %v", name, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}
