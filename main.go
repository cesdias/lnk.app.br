package main

import (
	"context"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	log "github.com/inconshreveable/log15"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/log/log15adapter"
	"github.com/jackc/pgx/v4/pgxpool"
)

var db *pgxpool.Pool

func getURLHandler(w http.ResponseWriter, req *http.Request) {
	var url string
	p := strings.Replace(req.URL.Path, "/", "", -1)
	err := db.QueryRow(context.Background(), "select url from shortened_urls where id=$1", p).Scan(&url)
	switch err {
	case nil:
		http.Redirect(w, req, url, http.StatusSeeOther)
	case pgx.ErrNoRows:
		http.NotFound(w, req)
	default:
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func putURLHandler(w http.ResponseWriter, req *http.Request) {
	//id := req.URL.Path
	var url string
	if body, err := ioutil.ReadAll(req.Body); err == nil {
		url = string(body)
	} else {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if _, err := db.Exec(context.Background(), `insert into shortened_urls(url) values ($1)`, url); err == nil {
		w.WriteHeader(http.StatusOK)
	} else {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func deleteURLHandler(w http.ResponseWriter, req *http.Request) {
	p := strings.Replace(req.URL.Path, "/", "", -1)
	if _, err := db.Exec(context.Background(), "delete from shortened_urls where id=$1", p); err == nil {
		w.WriteHeader(http.StatusOK)
	} else {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func urlHandler(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		getURLHandler(w, req)

	case "PUT":
		putURLHandler(w, req)

	case "DELETE":
		deleteURLHandler(w, req)

	default:
		w.Header().Add("Allow", "GET, PUT, DELETE")
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func main() {
	logger := log15adapter.NewLogger(log.New("module", "pgx"))

	poolConfig, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Crit("Unable to parse DATABASE_URL", "error", err)
		os.Exit(1)
	}

	poolConfig.ConnConfig.Logger = logger

	db, err = pgxpool.ConnectConfig(context.Background(), poolConfig)
	if err != nil {
		log.Crit("Unable to create connection pool", "error", err)
		os.Exit(1)
	}

	http.HandleFunc("/", urlHandler)

	log.Info("Starting URL shortener on localhost:8080")
	err = http.ListenAndServe("localhost:8080", nil)
	if err != nil {
		log.Crit("Unable to start web server", "error", err)
		os.Exit(1)
	}
}
