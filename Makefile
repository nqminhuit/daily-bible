# Makefile for daily-bible

.PHONY: test compile import-sqlite build

test:
	go test -tags "fts5" -cover ./...

compile:
	go build -tags "fts5" ./...

import-sqlite:
	@go run -tags "fts5" ./cmd/importer -db data/readings.db -schema data/schema.sql

build:
	@mkdir -p build
	@go build -tags "fts5" -ldflags="-s -w" -o build/daily-bible ./cmd/server

dev:
	@go run -tags "fts5" ./cmd/server

clean:
	@rm -rf build
