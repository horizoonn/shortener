SHELL := /bin/bash

-include .env
export

.DEFAULT_GOAL := help

APP_NAME := shortener
CMD := ./cmd/shortener
ENV_SERVICES := postgres redis
MIGRATIONS_DIR := migrations
DATABASE_URL ?= $(SHORTENER_DATABASE_URL)

.PHONY: help fmt fmt-check vet test test-race test-integration check check-all env-up env-down env-cleanup migrate-create migrate-up migrate-down shortener-run shortener-deploy shortener-undeploy shortener-logs

help:
	@awk 'BEGIN {FS = ":.*##"; printf "Available targets:\n"} /^[a-zA-Z0-9_-]+:.*##/ {printf "  %-22s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

fmt: ## Format Go files.
	gofmt -w $$(find . -path './.out' -prune -o -name '*.go' -print)

fmt-check: ## Check Go formatting.
	@files="$$(gofmt -l $$(find . -path './.out' -prune -o -name '*.go' -print))"; \
	if [ -n "$$files" ]; then \
		echo "Files require gofmt:"; \
		echo "$$files"; \
		exit 1; \
	fi

vet: ## Run go vet.
	go vet ./...

test: ## Run unit tests.
	go test ./...

test-race: ## Run unit tests with race detector.
	go test -race ./...

test-integration: ## Run integration tests.
	go test -tags=integration ./...

check: fmt-check vet test ## Run fast local checks.

check-all: check test-race test-integration ## Run all checks.

env-up: ## Start PostgreSQL and Redis.
	docker compose up -d $(ENV_SERVICES)

env-down: ## Stop local environment.
	docker compose down

env-cleanup: ## Stop local environment and remove volumes.
	docker compose down -v --remove-orphans

migrate-create: ## Create a migration: make migrate-create name=create_links.
	@test -n "$(name)" || (echo "usage: make migrate-create name=create_links" && exit 1)
	migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq $(name)

migrate-up: ## Apply migrations.
	@test -n "$(DATABASE_URL)" || (echo "SHORTENER_DATABASE_URL or DATABASE_URL is required" && exit 1)
	migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" up

migrate-down: ## Roll back one migration.
	@test -n "$(DATABASE_URL)" || (echo "SHORTENER_DATABASE_URL or DATABASE_URL is required" && exit 1)
	migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" down 1

shortener-run: ## Run the service locally.
	go run $(CMD)

shortener-deploy: ## Build and start the service with Docker Compose.
	docker compose up -d --build $(APP_NAME)

shortener-undeploy: ## Stop the service.
	docker compose stop $(APP_NAME)

shortener-logs: ## Tail service logs.
	docker compose logs -f $(APP_NAME)
