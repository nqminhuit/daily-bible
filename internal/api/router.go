package api

import (
	"database/sql"
	"net/http"
)

func NewRouter(db *sql.DB) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/gospel/", makeGetGospelHandler(db))
	mux.HandleFunc("/api/v1/search", makeSearchHandler(db))
	mux.HandleFunc("/api/v1/random", makeRandomHandler(db))
	return mux
}
