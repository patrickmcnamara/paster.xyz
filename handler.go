package main

import (
	"html/template"
	"net/http"
)

func errorHandler(w http.ResponseWriter, title, desc string, statusCode int) {
	w.WriteHeader(statusCode)
	t, _ := template.ParseFiles("template/page.tmpl", "template/error.tmpl")
	t.ExecuteTemplate(w, "error-page", map[string]string{
		"Title":       "error - " + title,
		"Description": desc,
	})
}
