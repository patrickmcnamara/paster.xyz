package main

import (
	"html/template"
	"io/ioutil"
	"net/http"
)

func notFoundHandler(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	data, _ := ioutil.ReadFile("static/not-found.html")
	w.Write(data)
}

func errorHandler(w http.ResponseWriter, title, desc string, statusCode int) {
	w.WriteHeader(statusCode)
	t, _ := template.ParseFiles("static/error.html")
	t.Execute(w, map[string]string{
		"Title":       title,
		"Description": desc,
	})
}
