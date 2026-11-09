# Makefile for daily-bible

PHONY_TARGETS := $(shell grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | cut -d: -f1)
.PHONY: $(PHONY_TARGETS)

DB=resources/bible.db
PORT=:8090

GOFLAGS=-tags "fts5"

.DEFAULT_GOAL := help

help: ## (0) Show all available commands and their descriptions
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | \
	awk 'BEGIN {FS=":.*?## "}; {printf "%-23s %s\n", $$1, $$2}'

test-with-race-detector: ## (0) Run tests with race detector, better run on CI
	go test $(GOFLAGS) -race -cover ./...

compile: ## (0) Compile the project, but do not build the binary files
	go build $(GOFLAGS) ./...

fmt: ## (0) Format the code using gofmt
	go fmt ./...

test: ## (1) Run all unit tests
	go test $(GOFLAGS) -cover ./...

crawler: ## (2) Crawl the 3 latest Bible readings and save them to build/gospels.tsv
	@mkdir -p build
	go run ./tools/crawler -totalUrls=3

crawler-all-urls: ## (2) Crawl the Bible verses and save them to build/gospels.tsv
	@mkdir -p build
	go run ./tools/crawler

tsv: build/gospels.txt ## (3) Convert the crawled data to TSV format
	@mkdir -p build
	go run ./tools/tsv

import-db: data/migrations build/gospels.tsv ## (4) Import data into SQLite database, requires sqlite3 to be installed
	@mkdir -p resources
	@rm -f $(DB)

	sqlite3 $(DB) < data/migrations/001_initial.sql

	printf "%s\n" \
	"PRAGMA journal_mode=OFF;" \
	"PRAGMA synchronous=OFF;" \
	".mode tabs" \
	".import build/gospels.tsv verses" \
	| sqlite3 $(DB)

	sqlite3 $(DB) "INSERT INTO verses_fts(verses_fts) VALUES('rebuild');"
	sqlite3 $(DB) < data/migrations/002_daily_readings.sql
	sqlite3 $(DB) < data/migrations/003_reading_sources.sql
	sqlite3 $(DB) < data/migrations/004_reading_cache.sql
	@echo "✅ Database imported successfully to $(DB)"

crawl-lectionary: ## Generate calendar estimates and verify exact-date readings (YEARS=2026,2027)
	go run $(GOFLAGS) ./tools/lectionarycrawler -years="$(YEARS)"

verify-readings: ## Verify a bounded date window from Vatican News (FROM=2026-11-09 TO=2026-11-11)
	go run $(GOFLAGS) ./tools/lectionarycrawler -from="$(FROM)" -to="$(TO)"

setup-lectionary: ## Full lectionary setup (YEARS=2026,2027)
	$(MAKE) crawl-lectionary YEARS="$(YEARS)"

build: ## (5) Build the server binary
	@mkdir -p build
	go build $(GOFLAGS) -ldflags="-s -w" -o build/daily-bible-server ./cmd/server

build-cli: ## Build the CLI binary
	@mkdir -p build
	go build $(GOFLAGS) -ldflags="-s -w" -o build/daily-bible ./cmd/cli

dev: ## (6) Run the server in development mode (PORT=:8090)
	@mkdir -p build
	go run $(GOFLAGS) ./cmd/server -port $(PORT)

clean: ## (99) Clean the build artifacts
	rm -rf build
