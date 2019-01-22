package main

import (
	"html/template"
	"net/http"
)

func errorHandler(w http.ResponseWriter, title, desc string, statusCode int) {
	w.WriteHeader(statusCode)
	t, _ := template.ParseFiles("template/error.html")
	t.Execute(w, map[string]string{
		"Title":       title,
		"Description": desc,
	})
}
