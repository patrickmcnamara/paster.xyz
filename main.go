package main

import (
	"database/sql"
	"log"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	// load database configuration file
	cfg, err := loadDbConfig("db-config.json")
	if err != nil {
		log.Fatal(err)
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
	http.Handle("/", &app{db})
	log.Print("Serving HTTP")
	log.Fatal(http.ListenAndServe(":80", nil))
}
