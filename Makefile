SHELL := /bin/bash

-include .env
export

.DEFAULT_GOAL := help

APP_NAME := shortener
CMD := ./cmd/shortener
ENV_SERVICES := postgres redis
MIGRATIONS_DIR := migrations
DATABASE_URL ?= $(SHORTENER_DATABASE_URL)
COVERAGE_DIR := .out/coverage
COVERAGE_PROFILE := $(COVERAGE_DIR)/coverage.out
COVERAGE_HTML := $(COVERAGE_DIR)/coverage.html
INTEGRATION_COVERAGE_PROFILE := $(COVERAGE_DIR)/integration_coverage.out
INTEGRATION_COVERAGE_HTML := $(COVERAGE_DIR)/integration_coverage.html
GOLANGCI_LINT_VERSION := v2.12.2
STATICCHECK_VERSION := 2026.1
ACTIONLINT_VERSION := v1.7.12

.PHONY: help fmt fmt-check vet lint staticcheck actionlint test test-cover test-cover-profile test-cover-func test-cover-html test-race test-integration test-integration-cover test-integration-cover-html check check-all env-up env-down env-cleanup migrate-create migrate-up migrate-down shortener-run shortener-deploy shortener-undeploy shortener-logs

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

lint: ## Run golangci-lint.
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run --allow-parallel-runners ./...

staticcheck: ## Run staticcheck.
	go run honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION) ./...

actionlint: ## Lint GitHub Actions workflows.
	go run github.com/rhysd/actionlint/cmd/actionlint@$(ACTIONLINT_VERSION)

test: ## Run unit tests.
	go test ./...

test-cover: ## Run unit tests with coverage summary.
	go test ./... -cover

test-cover-profile: ## Write unit test coverage profile.
	mkdir -p $(COVERAGE_DIR)
	go test ./... -coverprofile=$(COVERAGE_PROFILE)

test-cover-func: test-cover-profile ## Show unit test coverage by function.
	go tool cover -func=$(COVERAGE_PROFILE)

test-cover-html: test-cover-profile ## Write unit test coverage HTML report.
	go tool cover -html=$(COVERAGE_PROFILE) -o $(COVERAGE_HTML)
	@echo "Coverage HTML: $(COVERAGE_HTML)"

test-race: ## Run unit tests with race detector.
	go test -race ./...

test-integration: ## Run integration tests.
	go test -tags=integration ./...

test-integration-cover: ## Write integration coverage profile and show function coverage.
	mkdir -p $(COVERAGE_DIR)
	go test -tags=integration ./... -coverprofile=$(INTEGRATION_COVERAGE_PROFILE)
	go tool cover -func=$(INTEGRATION_COVERAGE_PROFILE)

test-integration-cover-html: ## Write integration coverage HTML report.
	mkdir -p $(COVERAGE_DIR)
	go test -tags=integration ./... -coverprofile=$(INTEGRATION_COVERAGE_PROFILE)
	go tool cover -html=$(INTEGRATION_COVERAGE_PROFILE) -o $(INTEGRATION_COVERAGE_HTML)
	@echo "Integration coverage HTML: $(INTEGRATION_COVERAGE_HTML)"

check: fmt-check vet lint actionlint test ## Run fast local checks.

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
