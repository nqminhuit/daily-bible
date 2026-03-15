# Makefile for daily-bible

.PHONY: test compile import-sqlite build

DB=build/bible.db

test-with-race-detection:
	go test -tags "fts5" -race -cover ./...

test:
	go test -tags "fts5" -cover ./...

compile:
	go build -tags "fts5" ./...

import-db: data/schema.sql build/gospels.tsv
	mkdir -p build
	rm -f $(DB)

	sqlite3 $(DB) < data/schema.sql

	sqlite3 $(DB) <<EOF
	PRAGMA journal_mode=OFF;
	PRAGMA synchronous=OFF;
	.mode tabs
	.import build/gospels.tsv verses
	EOF

	sqlite3 $(DB) < data/indexes.sql
	sqlite3 $(DB) < data/fts.sql
	sqlite3 $(DB) "INSERT INTO verses_fts(verses_fts) VALUES('rebuild');"
	sqlite3 $(DB) < data/triggers.sql

build:
	@mkdir -p build
	@go build -tags "fts5" -ldflags="-s -w" -o build/daily-bible ./cmd/server

links:
	go run ./tools/biblelinks

crawler: build/bible-links.txt
	@go run ./tools/crawler

dev:
	@mkdir -p build
	@go run -tags "fts5" ./cmd/server

clean:
	@rm -rf build
