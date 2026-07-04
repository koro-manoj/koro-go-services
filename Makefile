.PHONY: help deps tidy test build run-api run-worker docker-up docker-down migrate lint

GO ?= go
BIN_DIR := bin

help:
	@echo "Targets: deps tidy test build run-api run-worker docker-up docker-down migrate"

deps:
	$(GO) mod download

tidy:
	$(GO) mod tidy

test:
	$(GO) test ./...

build: tidy
	mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/api ./cmd/api
	$(GO) build -o $(BIN_DIR)/worker ./cmd/worker

run-api: build
	$(BIN_DIR)/api

run-worker: build
	$(BIN_DIR)/worker

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down

migrate:
	docker compose exec -T postgres psql -U koro -d koro -f /docker-entrypoint-initdb.d/001_initial.sql

lint:
	$(GO) vet ./...
