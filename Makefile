.PHONY: run build test lint up down migrate-up migrate-down migrate-status migrate-create

# Load .env so the migrate targets can build the DB connection string.
-include .env
export

GOOSE_DBSTRING = postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)

run:
	go run ./cmd/api

build:
	go build -o bin/finishline ./cmd/api

test:
	go test ./...

lint:
	golangci-lint run

up:
	docker compose up -d --build

down:
	docker compose down

# Versioned migrations (production schema owner). Dev still uses gorm
# AutoMigrate; these files must stay in sync with the gorm models.
migrate-up:
	goose -dir migrations postgres "$(GOOSE_DBSTRING)" up

migrate-down:
	goose -dir migrations postgres "$(GOOSE_DBSTRING)" down

migrate-status:
	goose -dir migrations postgres "$(GOOSE_DBSTRING)" status

migrate-create:
	goose -dir migrations create $(name) sql
