package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

/*
Endpoints:
- GET /
- POST /api/login
- POST /api/logout
- GET/POST/PATCH/DELETE /api/room/:id
- WS /api/room/:id/connect
- POST /api/room/:id/join
*/

var db *sql.DB
var secureCookies bool

func main() {
	log.SetOutput(os.Stderr)
	// TODO: use environment variables or config
	secureCookies = false
	connStr := "dbname=concinnity user=postgres host=localhost password=postgres sslmode=disable"
	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Panicln("Failed to open connection to database!", err)
	}
	CreateSqlTables()
	PrepareSqlStatements()

	// Endpoints
	http.HandleFunc("/", StatusEndpoint)
	http.HandleFunc("/api/login", LoginEndpoint)
	http.HandleFunc("/api/logout", LogoutEndpoint)
	http.HandleFunc("/api/room", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			http.Error(w, errorJson("Not Implemented!"), http.StatusNotImplemented) // TODO
		} else if r.Method == "POST" {
			http.Error(w, errorJson("Not Implemented!"), http.StatusNotImplemented) // TODO
		} else if r.Method == "PATCH" {
			http.Error(w, errorJson("Not Implemented!"), http.StatusNotImplemented) // TODO
		} else if r.Method == "DELETE" {
			http.Error(w, errorJson("Not Implemented!"), http.StatusNotImplemented) // TODO
		} else {
			http.Error(w, errorJson("Method Not Allowed!"), http.StatusMethodNotAllowed)
		}
	})

	port := "8000"
	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}
	log.SetOutput(os.Stdout)
	log.Println("Listening to port " + port)
	log.SetOutput(os.Stderr)
	http.ListenAndServe(":"+port, nil)
}
