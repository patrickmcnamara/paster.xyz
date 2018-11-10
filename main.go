package main

import (
	"database/sql"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	// load config file
	cfg, err := loadConfig("config.json")
	if err != nil {
		log.Fatal(err)
	}

	// sort logfile out
	switch cfg.LogFile {
	case "":
		log.SetOutput(ioutil.Discard)
	case "stdout":
		log.SetOutput(os.Stdout)
	case "stderr":
		log.SetOutput(os.Stderr)
	default:
		f, err := os.OpenFile(cfg.LogFile, os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			log.Fatalln(err)
		}
		defer f.Close()
		log.SetOutput(f)
	}

	// open database and test connection
	log.Println("Setting up database...")
	dataSourceName := cfg.getDataSourceName()
	db, err := sql.Open("mysql", dataSourceName)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	log.Println("Testing database connection...")
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	// start server
	log.Println("Starting server...")
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))
	http.Handle("/", &app{db, cfg})
	// serve HTTP and redirect to HTTPS
	log.Print("Serving HTTP")
	go func() {
		log.Fatal(http.ListenAndServe(":80", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "https://"+r.Host+r.URL.Path, http.StatusMovedPermanently)
			log.Printf("%s - redirected HTTP to HTTPS", r.URL.Path)
		})))
	}()
	// serve HTTPS
	log.Print("Serving HTTPS")
	func() {
		log.Fatal(http.ListenAndServeTLS(":443", cfg.CertFile, cfg.KeyFile, nil))
	}()
}
