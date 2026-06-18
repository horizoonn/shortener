SHELL := /bin/bash

-include .env
export

.DEFAULT_GOAL := help

APP_NAME := shortener
CMD := ./cmd/shortener
GO_PACKAGES := ./cmd/... ./internal/...
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

.PHONY: help fmt fmt-check vet lint staticcheck actionlint test test-cover test-cover-profile test-cover-func test-cover-html test-race test-integration test-integration-cover test-integration-cover-html check check-all env-up env-down env-cleanup migrate-create migrate-up migrate-down shortener-run shortener-deploy shortener-undeploy shortener-logs observability-up observability-down observability-logs web-install web-dev web-build web-lint web-audit web-check dev-up dev-down dev-logs

help:
	@awk 'BEGIN {FS = ":.*##"; printf "Available targets:\n"} /^[a-zA-Z0-9_-]+:.*##/ {printf "  %-22s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

fmt: ## Format Go files.
	gofmt -w $$(find cmd internal -name '*.go' -print)

fmt-check: ## Check Go formatting.
	@files="$$(gofmt -l $$(find cmd internal -name '*.go' -print))"; \
	if [ -n "$$files" ]; then \
		echo "Files require gofmt:"; \
		echo "$$files"; \
		exit 1; \
	fi

vet: ## Run go vet.
	go vet $(GO_PACKAGES)

lint: ## Run golangci-lint.
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run --allow-parallel-runners $(GO_PACKAGES)

staticcheck: ## Run staticcheck.
	go run honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION) $(GO_PACKAGES)

actionlint: ## Lint GitHub Actions workflows.
	go run github.com/rhysd/actionlint/cmd/actionlint@$(ACTIONLINT_VERSION)

test: ## Run unit tests.
	go test $(GO_PACKAGES)

test-cover: ## Run unit tests with coverage summary.
	go test $(GO_PACKAGES) -cover

test-cover-profile: ## Write unit test coverage profile.
	mkdir -p $(COVERAGE_DIR)
	go test $(GO_PACKAGES) -coverprofile=$(COVERAGE_PROFILE)

test-cover-func: test-cover-profile ## Show unit test coverage by function.
	go tool cover -func=$(COVERAGE_PROFILE)

test-cover-html: test-cover-profile ## Write unit test coverage HTML report.
	go tool cover -html=$(COVERAGE_PROFILE) -o $(COVERAGE_HTML)
	@echo "Coverage HTML: $(COVERAGE_HTML)"

test-race: ## Run unit tests with race detector.
	go test -race $(GO_PACKAGES)

test-integration: ## Run integration tests.
	go test -tags=integration $(GO_PACKAGES)

test-integration-cover: ## Write integration coverage profile and show function coverage.
	mkdir -p $(COVERAGE_DIR)
	go test -tags=integration $(GO_PACKAGES) -coverprofile=$(INTEGRATION_COVERAGE_PROFILE)
	go tool cover -func=$(INTEGRATION_COVERAGE_PROFILE)

test-integration-cover-html: ## Write integration coverage HTML report.
	mkdir -p $(COVERAGE_DIR)
	go test -tags=integration $(GO_PACKAGES) -coverprofile=$(INTEGRATION_COVERAGE_PROFILE)
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

shortener-deploy: web-build ## Build and start the service with Docker Compose.
	docker compose up -d --build $(APP_NAME)

shortener-undeploy: ## Stop the service.
	docker compose stop $(APP_NAME)

shortener-logs: ## Tail service logs.
	docker compose logs -f $(APP_NAME)

observability-up: ## Start backend, Prometheus, and Grafana.
	docker compose --profile observability up -d --build shortener prometheus grafana

observability-down: ## Stop Prometheus and Grafana.
	docker compose --profile observability stop prometheus grafana

observability-logs: ## Tail Prometheus and Grafana logs.
	docker compose --profile observability logs -f prometheus grafana

web-install: ## Install frontend dependencies.
	cd web && npm ci

web-dev: ## Run frontend dev server locally.
	cd web && npm run dev

web-build: ## Build frontend assets into web/public.
	cd web && npm run build

web-lint: ## Lint frontend.
	cd web && npm run lint

web-audit: ## Audit frontend dependencies.
	cd web && npm audit

web-check: web-lint web-build web-audit ## Run frontend checks.

dev-up: ## Start backend and frontend with Docker Compose.
	@test -d web/node_modules || $(MAKE) web-install
	docker compose up --build shortener frontend

dev-down: ## Stop backend and frontend Docker Compose environment.
	docker compose down

dev-logs: ## Tail backend and frontend logs.
	docker compose logs -f shortener frontend
