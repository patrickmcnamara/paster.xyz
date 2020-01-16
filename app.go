package main

import (
	"archive/tar"
	"compress/gzip"
	"database/sql"
	"encoding/base64"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	pasteLength     int = 5 << 20 // 5 MiB paste length limit
	pasteListLength int = 24      // 24 rows in paste lists
)

type app struct {
	DB *sql.DB
}

func (a *app) setPaste(p paste) error {
	_, err := a.DB.Exec("INSERT INTO paste (ID, Value, Time) VALUES (?, ?, ?)", p.ID, p.Value, p.Time)
	return err
}

func (a *app) getPaste(id []byte) (p paste, err error) {
	r := a.DB.QueryRow("SELECT Value FROM paste WHERE ID = ?", id)
	err = r.Scan(&p.Value)
	return
}

func (a *app) getRecentPastes(pageNo int) (ps []paste, maxPageNo int, err error) {
	rows, err := a.DB.Query("SELECT ID, Time FROM paste ORDER BY Time DESC LIMIT ?, ?", pageNo*pasteListLength, pasteListLength)
	defer rows.Close()
	if err != nil {
		return
	}
	for rows.Next() {
		var p paste
		rows.Scan(&p.ID, &p.Time)
		ps = append(ps, p)
	}
	r := a.DB.QueryRow("SELECT COUNT(*) FROM paste")
	err = r.Scan(&maxPageNo)
	maxPageNo = (maxPageNo - 1) / pasteListLength
	return
}

func (a *app) getAllPastes() (ps []paste, err error) {
	rows, err := a.DB.Query("SELECT ID, Time, Value FROM paste ORDER BY Time DESC")
	defer rows.Close()
	if err != nil {
		return
	}
	for rows.Next() {
		var p paste
		rows.Scan(&p.ID, &p.Time, &p.Value)
		ps = append(ps, p)
	}
	return
}

func (a *app) getLatestPasteID() (id []byte, err error) {
	r := a.DB.QueryRow("SELECT ID FROM paste ORDER BY Time DESC LIMIT 1")
	err = r.Scan(&id)
	return
}

func (a *app) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	title := strings.Replace(path[1:], "-", " ", -1)
	method := r.Method

	switch method {
	case "GET":
		switch path {
		// serve homepage
		case "/":
			w.Header().Set("Cache-Control", "no-cache")
			t, _ := template.ParseFiles("template/page.tmpl", "template/index.tmpl")
			t.ExecuteTemplate(w, "index-page", pasteLength)
			log.Printf("%s - %s - homepage", method, path)

		// 404 favicon, don't log
		case "/favicon.ico":
			http.NotFound(w, r)

		// status page, don't log
		case "/status":
			fmt.Fprintln(w, "UP")

		// serve latest paste
		case "/latest", "/l":
			id, err := a.getLatestPasteID()
			if err != nil {
				errS := "could not list history"
				errorHandler(w, errS, err.Error(), http.StatusInternalServerError)
				log.Printf("%s - %s - %s", method, path, errS)
				return
			}
			w.Header().Set("Cache-Control", "no-cache")
			http.Redirect(w, r, "/"+base64.RawURLEncoding.EncodeToString(id), http.StatusSeeOther)
			log.Printf("%s - %s - redirecting to %s", method, path, "/"+base64.RawURLEncoding.EncodeToString(id))

		// list recent pastes
		case "/recent":
			currPageNo, err := strconv.Atoi(r.URL.Query().Get("p"))
			if err != nil {
				currPageNo = 0
			}
			pastes, maxPageNo, err := a.getRecentPastes(currPageNo)
			if currPageNo > maxPageNo {
				currPageNo = maxPageNo
			}
			if err != nil {
				errS := "could not list recent pastes"
				errorHandler(w, errS, err.Error(), http.StatusInternalServerError)
				log.Printf("%s - %s - %s", method, path, errS)
				return
			}
			recent := make([][2]string, len(pastes))
			for i, p := range pastes {
				recent[i][0] = base64.RawURLEncoding.EncodeToString(p.ID)
				recent[i][1] = p.Time.Format("2006-01-02 15:04:05")
			}
			prevPageNo, nextPageNo := -1, -1
			if currPageNo != 0 {
				prevPageNo = currPageNo - 1
			}
			if currPageNo != maxPageNo {
				nextPageNo = currPageNo + 1
			}
			w.Header().Set("Cache-Control", "no-cache")
			t, _ := template.ParseFiles("template/page.tmpl", "template/recent.tmpl")
			t.ExecuteTemplate(w, "recent-page", struct {
				Pastes     [][2]string
				PrevPageNo int
				NextPageNo int
			}{
				Pastes:     recent,
				PrevPageNo: prevPageNo,
				NextPageNo: nextPageNo,
			})
			log.Printf("%s - %s - listing recent pastes", method, path)

		// other stuff
		case "/other", "/contact", "/privacy-policy", "/cookie-policy":
			w.Header().Set("Cache-Control", "no-cache")
			t, _ := template.ParseFiles("template/page.tmpl", "template/"+strings.Replace(title, " ", "-", -1)+".tmpl")
			t.ExecuteTemplate(w, strings.Replace(title, " ", "-", -1)+"-page", title)
			log.Printf("%s - %s - %s", method, path, title)

		// paster.xyz backup
		case "/paster-xyz.tar.gz":
			w.Header().Set("Cache-Control", "no-cache")
			ps, _ := a.getAllPastes()
			gzw := gzip.NewWriter(w)
			defer gzw.Close()
			tgz := tar.NewWriter(gzw)
			defer tgz.Close()
			for _, paste := range ps {
				tgz.WriteHeader(&tar.Header{
					Name:    base64.RawURLEncoding.EncodeToString(paste.ID[:]),
					Size:    int64(len(paste.Value)),
					Mode:    0666,
					ModTime: paste.Time,
				})
				tgz.Write([]byte(paste.Value))
			}
			log.Printf("%s - %s - generated tar.gz archive", method, path)

		// get paste if it exists, else return a 404
		default:
			id, _ := base64.RawURLEncoding.DecodeString(path[1:])
			if p, err := a.getPaste(id); err == nil {
				// headers
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.Header().Set("Cache-Control", "immutable")
				w.Header().Set("Access-Control-Allow-Origin", "*")
				w.Header().Set("Access-Control-Allow-Methods", "GET")
				// write paste
				fmt.Fprint(w, p.Value)
				log.Printf("%s - %s - paste found", method, path)
			} else {
				errorHandler(w, "404 not found", "OOPSIE WOOPSIE!! 😳 Uwu 😚 We make a fucky wucky!! 🙅‍ 🤷🏼‍ A wittle fucko boingo! 🌈💫 The code monkeys 🙈🙉at our headquarters 🕍 💤 are working VEWY HAWD 💸💯 to fix this! ♿️", http.StatusNotFound)
				log.Printf("%s - %s - page not found", method, path)
			}
		}

	case "POST":
		// parse values
		if err := r.ParseForm(); err != nil {
			errorHandler(w, "could not set paste", err.Error(), http.StatusInternalServerError)
			log.Printf("%s - %s - could not submit paste - %v", method, path, err)
			return
		}

		// generate paste ID
		id := generateID()

		// get value and validate
		value := r.FormValue("Value")
		if len(value) == 0 {
			errorHandler(w, "value too short", "value must not be empty", http.StatusBadRequest)
			log.Printf("%s - %s - could not submit paste, too short", method, path)
			return
		} else if len(value) >= pasteLength {
			errorHandler(w, "value too long", "value must be less than 5 mebibytes", http.StatusRequestEntityTooLarge)
			log.Printf("%s - %s - could not submit paste, too long", method, path)
			return
		}

		// create and set paste
		if err := a.setPaste(paste{ID: id, Value: value, Time: time.Now().UTC()}); err != nil {
			errorHandler(w, "could not set paste", err.Error(), http.StatusInternalServerError)
			log.Printf("%s - %s - could not submit paste - %v", method, path, err)
			return
		}

		// redirect to new paste
		http.Redirect(w, r, "/"+base64.RawURLEncoding.EncodeToString(id[:]), http.StatusSeeOther)
		log.Printf("%s - %s - paste submitted, redirecting to %s", method, path, "/"+base64.RawURLEncoding.EncodeToString(id[:]))
	}
}
