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

const (
	pasteLength     int = 64 << 10 // 64 KiB paste length limit
	pasteListLength int = 24       // 24 rows in paste lists
)

type app struct {
	DB *sql.DB
}

func (a *app) getPaste(pasteID id) (*paste, error) {
	var p paste
	r := a.DB.QueryRow("SELECT Value FROM paste WHERE ID = ? AND (Expiry IS NULL OR Expiry > NOW())", pasteID)
	err := r.Scan(&p.Value)
	return &p, err
}

func (a *app) setPaste(p *paste) error {
	_, err := a.DB.Exec("INSERT INTO paste (ID, Value, Time, Expiry, User, List) VALUES (?, ?, ?, ?, ?, ?)", p.ID, p.Value, p.Time, p.Expiry, p.User, p.List)
	return err
}

func (a *app) getHistoryPastes(user id) (ps []*paste, err error) {
	rows, err := a.DB.Query("SELECT ID, Time, Expiry FROM paste WHERE User = ? AND (Expiry IS NULL OR Expiry > NOW()) ORDER by Time DESC LIMIT ?", user, pasteListLength)
	defer rows.Close()
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var p paste
		p.User = user
		rows.Scan(&p.ID, &p.Time, &p.Expiry)
		ps = append(ps, &p)
	}
	return ps, nil
}

func (a *app) getRecentPastes() (ps []*paste, err error) {
	rows, err := a.DB.Query("SELECT ID, Time, Expiry FROM paste WHERE List AND (Expiry IS NULL OR Expiry > NOW()) ORDER BY Time DESC LIMIT ?", pasteListLength)
	defer rows.Close()
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var p paste
		rows.Scan(&p.ID, &p.Time, &p.Expiry)
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

		// get paste ID, Value, List and Expiry
		pasteID := generateID()
		value := r.FormValue("Value")
		if len(value) == 0 {
			errorHandler(w, "value too short", "value must not be empty", http.StatusBadRequest)
			log.Printf("%s - %s - could not submit paste, too short", method, path)
			return
		} else if len([]byte(value)) >= pasteLength {
			errorHandler(w, "value too long", "value must be less than 64 kibibytes", http.StatusRequestEntityTooLarge)
			log.Printf("%s - %s - could not submit paste, too long", method, path)
			return
		}
		list := r.FormValue("List") == "list"
		expiryValue := r.FormValue("Expiry")
		expiryTime, err := time.Parse(time.RFC3339[:16], expiryValue)
		var expiry nullTime
		if expiryValue == "" {
			expiry = nullTime{expiryTime, false}
		} else if err != nil {
			errorHandler(w, "invalid expiry", "expiry must be after the current time", http.StatusBadRequest)
			log.Printf("%s - %s - could not submit paste, invalid expiry", method, path)
			return
		} else if !expiryTime.After(time.Now()) {
			errorHandler(w, "expiry before current time", "expiry must be after the current time", http.StatusBadRequest)
			log.Printf("%s - %s - could not submit paste, expiry before current time", method, path)
			return
		} else {
			expiry = nullTime{expiryTime, true}
		}

		// assign to User
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

		// create and set paste
		err = a.setPaste(&paste{
			ID:     pasteID,
			Value:  value,
			Time:   time.Now().UTC(),
			Expiry: expiry,
			User:   user,
			List:   list,
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
