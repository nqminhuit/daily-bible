# Makefile for daily-bible

.PHONY: test compile import-sqlite build

test:
	go test -cover ./...

compile:
	go build ./...

import-sqlite:
	@test -f data/schema.sql || (echo "data/schema.sql not found" && exit 1)
	@test -f data/gospel.json || (echo "data/gospel.json not found" && exit 1)
	@mkdir -p data
	@go run ./cmd/importer -in data/gospel.json -db data/readings.db -schema data/schema.sql

build:
	@mkdir -p build
	@go build -ldflags="-s -w" -o build/daily-bible ./cmd/server
