package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/minh/daily-bible/internal/api"
	dbpkg "github.com/minh/daily-bible/internal/db"
)

func main() {
	dbPath := flag.String("db", "data/readings.db", "path to sqlite db")
	addr := flag.String("addr", ":8080", "server address")
	flag.Parse()

	db, err := dbpkg.Open(*dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := dbpkg.InitDB(db, "data/schema.sql"); err != nil {
		log.Printf("warning: init schema: %v", err)
	}

	mux := api.NewRouter(db)

	srv := &http.Server{
		Addr:         *addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Printf("listening on %s", *addr)
	log.Fatal(srv.ListenAndServe())
}
