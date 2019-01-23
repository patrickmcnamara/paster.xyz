package main

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"
)

const pasteLimit int = 24

type app struct {
	DB *sql.DB
}

func (a *app) getPaste(pasteID id) (*paste, error) {
	var p paste
	r := a.DB.QueryRow("SELECT Value FROM paste WHERE ID = ?", pasteID)
	err := r.Scan(&p.Value)
	return &p, err
}

func (a *app) setPaste(p *paste) error {
	_, err := a.DB.Exec("INSERT INTO paste VALUES (?, ?, ?, ?)", p.ID, p.Value, p.Time, p.User)
	return err
}

func (a *app) getHistoryPastes(user id) (ps []*paste, err error) {
	rows, err := a.DB.Query("SELECT ID, Time FROM paste WHERE User = ? ORDER by Time DESC LIMIT ?", user, pasteLimit)
	defer rows.Close()
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var p paste
		p.User = user
		rows.Scan(&p.ID, &p.Time)
		ps = append(ps, &p)
	}
	return ps, nil
}

func (a *app) getRecentPastes() (ps []*paste, err error) {
	rows, err := a.DB.Query("SELECT ID, Time FROM paste WHERE Time IS NOT NULL ORDER BY Time DESC LIMIT ?", pasteLimit)
	defer rows.Close()
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var p paste
		rows.Scan(&p.ID, &p.Time)
		ps = append(ps, &p)
	}
	return ps, nil
}

func (a *app) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	title := path[1:]
	method := r.Method

	switch method {
	case "GET":
		switch path {
		// serve homepage
		case "/":
			t, _ := template.ParseFiles("template/page.tmpl", "template/index.tmpl")
			t.ExecuteTemplate(w, "index-page", "paster")
			log.Printf("%s - %s - homepage", method, path)

		// don't serve favicon and don't log
		case "/favicon.ico":
			http.NotFound(w, r)

		// list history pastes
		case "/history":
			var pastes []*paste
			t, _ := template.ParseFiles("template/page.tmpl", "template/list.tmpl")
			if c, err := r.Cookie("user"); err == nil {
				user, _ := base64.RawURLEncoding.DecodeString(c.Value)
				pastes, err := a.getHistoryPastes(user)
				if err != nil {
					errS := "could not list history"
					errorHandler(w, errS, err.Error(), http.StatusInternalServerError)
					log.Printf("%s - %s - %s", method, path, errS)
					return
				}
				t.ExecuteTemplate(w, "list-page", map[string]interface{}{
					"Title":  title,
					"Pastes": pastes,
				})
				log.Printf("%s - %s - user cookie found, listing history", method, path)
			} else {
				t.ExecuteTemplate(w, "list-page", map[string]interface{}{
					"Title":  title,
					"Pastes": pastes,
				})
				log.Printf("%s - %s - user cookie not found, no history", method, path)
			}

		// list recent pastes
		case "/recent":
			var pastes []*paste
			t, _ := template.ParseFiles("template/page.tmpl", "template/list.tmpl")
			pastes, err := a.getRecentPastes()
			if err != nil {
				errS := "could not list recent pastes"
				errorHandler(w, errS, err.Error(), http.StatusInternalServerError)
				log.Printf("%s - %s - %s", method, path, errS)
				return
			}
			t.ExecuteTemplate(w, "list-page", map[string]interface{}{
				"Title":  title,
				"Pastes": pastes,
			})
			log.Printf("%s - %s - listing recent pastes", method, path)

		// get paste if it exists, else return a 404
		default:
			id, _ := base64.RawURLEncoding.DecodeString(path[1:])
			if p, err := a.getPaste(id); err == nil {
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				fmt.Fprint(w, p.Value)
				log.Printf("%s - %s - paste found", method, path)
			} else {
				errorHandler(w, "404 not found", "OOPSIE WOOPSIE!! 😳 Uwu 😚 We make a fucky wucky!! 🙅‍ 🤷🏼‍ A wittle fucko boingo! 🌈💫 The code monkeys 🙈🙉at our headquarters 🕍 💤 are working VEWY HAWD 💸💯 to fix this! ♿️", http.StatusNotFound)
				log.Printf("%s - %s - paste not found", method, path)
			}
		}

	case "POST":
		// parse values
		err := r.ParseForm()
		if err != nil {
			errorHandler(w, "could not set paste", err.Error(), http.StatusInternalServerError)
			log.Printf("%s - %s - could not submit paste - %v", method, path, err)
			return
		}

		// get paste ID and value
		pasteID := generateID()
		value := r.FormValue("paste")
		fmt.Println("paste =", len(value))
		if len(value) == 0 {
			errorHandler(w, "paste too short", "paste needs to not be empty", http.StatusBadRequest)
			log.Printf("%s - %s - could not submit paste, too short", method, path)
			return
		} else if len([]byte(value)) >= 65536 {
			errorHandler(w, "paste too long", "paste needs to be shorter than 65536 bytes", http.StatusRequestEntityTooLarge)
			log.Printf("%s - %s - could not submit paste, too long", method, path)
			return
		}

		// assign to user
		var user id
		if c, err := r.Cookie("user"); err == nil {
			user, _ = base64.RawURLEncoding.DecodeString(c.Value)
			http.SetCookie(w, &http.Cookie{
				Name:    "user",
				Expires: time.Now().Add(time.Hour * 24 * 365),
				Value:   c.Value,
			})
			log.Printf("%s - %s - user cookie found, refreshing cookie", method, path)
		} else {
			user = generateID()
			http.SetCookie(w, &http.Cookie{
				Name:    "user",
				Expires: time.Now().Add(time.Hour * 24 * 365),
				Value:   base64.RawURLEncoding.EncodeToString(user),
			})
			log.Printf("%s - %s - user cookie not found, setting cookie", method, path)
		}

		// create paste
		err = a.setPaste(&paste{
			ID:    pasteID,
			Value: value,
			Time:  time.Now().UTC(),
			User:  user,
		})
		if err != nil {
			errorHandler(w, "could not set paste", err.Error(), http.StatusInternalServerError)
			log.Printf("%s - %s - could not submit paste - %v", method, path, err)
			return
		}

		// redirect to new paste
		http.Redirect(w, r, "/"+pasteID.String(), http.StatusSeeOther)
		log.Printf("%s - %s - paste submitted, redirecting to %s", method, path, "/"+pasteID.String())
	}
}
