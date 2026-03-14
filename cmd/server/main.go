package main

import (
	"log"
	"net/http"
	"time"

	"github.com/minh/daily-bible/internal/api"
	"github.com/minh/daily-bible/internal/constants"
	dbpkg "github.com/minh/daily-bible/internal/db"
)

func main() {
	db, err := dbpkg.Open(constants.DBPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	mux := api.NewRouter(db)

	srv := &http.Server{
		Addr:         constants.ServerAddr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Printf("listening on %s", constants.ServerAddr)
	log.Fatal(srv.ListenAndServe())
}
