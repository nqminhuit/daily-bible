package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/minh/daily-bible/internal/api"
	"github.com/minh/daily-bible/internal/constants"
	dbpkg "github.com/minh/daily-bible/internal/db"
)

func main() {
	port := flag.String("port", constants.ServerAddr, "port to listen on (e.g. :8090)")
	flag.Parse()

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = constants.DBPath
	}

	log.Printf("using database at %s", dbPath)

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
		Addr:         *port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Printf("listening on %s", *port)
	log.Fatal(srv.ListenAndServe())
}
