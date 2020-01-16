package main

import (
	"html/template"
	"net/http"
)

func errorHandler(w http.ResponseWriter, title, desc string, statusCode int) {
	w.WriteHeader(statusCode)
	t, _ := template.ParseFiles("template/page.tmpl", "template/error.tmpl")
	t.ExecuteTemplate(w, "error-page", map[string]string{
		"Title":       title,
		"Description": desc,
	})
}

func notFoundHandler(w http.ResponseWriter) {
	errorHandler(w, "404 not found", "OOPSIE WOOPSIE!! 😳 Uwu 😚 We make a fucky wucky!! 🙅‍ 🤷🏼‍ A wittle fucko boingo! 🌈💫 The code monkeys 🙈🙉at our headquarters 🕍 💤 are working VEWY HAWD 💸💯 to fix this! ♿️", http.StatusNotFound)
}
