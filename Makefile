# Makefile for daily-bible

PHONY_TARGETS := $(shell grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | cut -d: -f1)
.PHONY: $(PHONY_TARGETS)

DB=build/bible.db

GOFLAGS=-tags "fts5"

.DEFAULT_GOAL := help

help: ## (0) Show all available commands and their descriptions
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | \
	awk 'BEGIN {FS=":.*?## "}; {printf "%-23s %s\n", $$1, $$2}'

test-with-race-detector: ## (0) Run tests with race detector, better run on CI
	go test $(GOFLAGS) -race -cover ./...

compile: ## (0) Compile the project, but do not build the binary files
	go build $(GOFLAGS) ./...

test: ## (1) Run all unit tests
	go test $(GOFLAGS) -cover ./...

links: ## (2) Crawl the Bible links and save them to build/bible-links.txt
	@mkdir -p build
	go run ./tools/biblelinks

crawler: build/bible-links.txt ## (3) Crawl the Bible verses and save them to build/gospels.tsv
	@mkdir -p build
	go run ./tools/crawler

tsv: build/gospels.txt ## (4) Convert the crawled data to TSV format
	@mkdir -p build
	go run ./tools/tsv

import-db: data/schema.sql build/gospels.tsv ## (5) Import data into SQLite database, requires sqlite3 to be installed
	@mkdir -p build
	@rm -f $(DB)

	sqlite3 $(DB) < data/schema.sql

	printf "%s\n" \
	"PRAGMA journal_mode=OFF;" \
	"PRAGMA synchronous=OFF;" \
	".mode tabs" \
	".import build/gospels.tsv verses" \
	| sqlite3 $(DB)

	sqlite3 $(DB) < data/fts.sql
	sqlite3 $(DB) "INSERT INTO verses_fts(verses_fts) VALUES('rebuild');"
	sqlite3 $(DB) < data/triggers.sql
	@echo "✅ Database imported successfully to $(DB)"

build: ## (6) Build the binary server file
	@mkdir -p build
	go build $(GOFLAGS) -ldflags="-s -w" -o build/daily-bible ./cmd/server

dev: ## (7) Run the server in development mode
	@mkdir -p build
	go run $(GOFLAGS) ./cmd/server

clean: ## (99) Clean the build artifacts
	rm -rf build
