package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/minh/daily-bible/internal/api"
	"github.com/minh/daily-bible/internal/constants"
	dbpkg "github.com/minh/daily-bible/internal/db"
)

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = constants.DBPath
	}

	db, err := dbpkg.Open(dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	mux, err := api.NewRouter(db)
	if err != nil {
		log.Fatalf("init router: %v", err)
	}

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
