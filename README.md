# Shortener

Shortener is a production-style URL shortener with a Go backend, PostgreSQL storage, Redis cache-aside link resolution, click analytics, and a React frontend.

## Stack

- Go with standard `net/http`
- PostgreSQL as the source of truth
- Redis as cache-aside storage for link resolution
- `github.com/kelseyhightower/envconfig` for configuration
- `go.uber.org/zap` for logging
- Prometheus metrics and Grafana dashboards for local observability
- `golang-migrate` for migrations
- Vite, React, TypeScript, Tailwind CSS, and shadcn/ui for the frontend
- Docker Compose for local infrastructure

## Architecture

```text
cmd/shortener              service entrypoint and Dockerfile
internal/app               dependency wiring and lifecycle
internal/config            environment config
internal/logger            zap logger setup
internal/errors            sentinel domain errors
internal/httpapi           HTTP helpers, middleware, router
internal/links             link domain packages
internal/analytics         analytics domain packages
migrations                 database migrations
web                         Vite React frontend
web/public                  built frontend assets served by the backend image
docs/openapi.yaml          OpenAPI contract
observability              Prometheus and Grafana provisioning
```

## Quick Start

For the full app in Docker Compose:

```bash
cp .env.example .env
make dev-up
```

Open:

```text
http://localhost:5173
```

The Vite dev server proxies API and redirect requests to the backend container.
If `web/node_modules` is missing, `make dev-up` installs frontend dependencies with `npm ci` before starting Compose.

To stop the environment:

```bash
make dev-down
```

For backend-only local development:

```bash
cp .env.example .env
make env-up
make migrate-up
make shortener-run
curl http://localhost:8080/healthz
```

For a fully containerized run, use:

```bash
cp .env.example .env
make shortener-deploy
```

Then open:

```text
http://localhost:8080
```

`make shortener-deploy` builds the React assets first and then packages `web/public` into the Go service image.

To start the backend with Prometheus and Grafana:

```bash
cp .env.example .env
make observability-up
```

Open:

```text
Prometheus: http://localhost:9090
Grafana:    http://localhost:3000
```

Default Grafana credentials are `admin` / `admin`. The Prometheus datasource and `Shortener Overview` dashboard are provisioned from the repository. The backend exposes metrics at `GET /metrics`; keep this endpoint internal or protected in real deployments.

## Makefile Commands

| Command | Description |
| --- | --- |
| `make help` | Show available commands. |
| `make fmt` | Format Go files. |
| `make fmt-check` | Check Go formatting. |
| `make vet` | Run `go vet ./...`. |
| `make lint` | Run `golangci-lint`. |
| `make staticcheck` | Run Staticcheck. |
| `make actionlint` | Lint GitHub Actions workflows. |
| `make test` | Run `go test ./...`. |
| `make test-cover` | Run unit tests with coverage summary. |
| `make test-cover-func` | Show unit test coverage by function. |
| `make test-cover-html` | Write unit test coverage HTML report. |
| `make test-race` | Run tests with the race detector. |
| `make test-integration` | Run integration tests with the `integration` tag. |
| `make test-integration-cover` | Show integration coverage by function. |
| `make test-integration-cover-html` | Write integration coverage HTML report. |
| `make check` | Run formatting, vet, golangci-lint, actionlint, and unit tests. |
| `make check-all` | Run all checks. |
| `make env-up` | Start PostgreSQL and Redis. |
| `make env-down` | Stop Docker Compose services. |
| `make env-cleanup` | Stop services and remove volumes. |
| `make migrate-create name=...` | Create a migration. |
| `make migrate-up` | Apply migrations. |
| `make migrate-down` | Roll back one migration. |
| `make shortener-run` | Run the service locally. |
| `make shortener-deploy` | Build and run the service in Docker Compose. |
| `make shortener-undeploy` | Stop the service container. |
| `make shortener-logs` | Tail service logs. |
| `make observability-up` | Start backend, Prometheus, and Grafana. |
| `make observability-down` | Stop Prometheus and Grafana. |
| `make observability-logs` | Tail Prometheus and Grafana logs. |
| `make web-install` | Install frontend dependencies. |
| `make web-dev` | Run the Vite dev server locally. |
| `make web-build` | Build frontend assets into `web/public`. |
| `make web-lint` | Lint frontend code. |
| `make web-audit` | Audit frontend dependencies. |
| `make web-check` | Run frontend lint, build, and audit. |
| `make dev-up` | Start backend and frontend with Docker Compose. |
| `make dev-down` | Stop the Docker Compose development environment. |
| `make dev-logs` | Tail backend and frontend logs. |

## CI

GitHub Actions runs frontend checks, Go checks, race tests, and integration tests as separate jobs. The workflow uses the Go version from `go.mod`, Node.js 20 for the frontend, dependency caches for Go and npm, and keeps the default token limited to read-only repository contents.

## API

| Method | Path | Status |
| --- | --- | --- |
| `GET` | `/healthz` | Implemented. |
| `GET` | `/readyz` | Implemented. |
| `GET` | `/metrics` | Implemented. Prometheus/OpenMetrics exposition. |
| `POST` | `/api/v1/shorten` | Implemented. |
| `GET` | `/s/{code}` | Implemented. |
| `GET` | `/api/v1/analytics/{code}` | Implemented. |
| `GET` | `/api/v1/links/{code}/qr` | Implemented. Returns a PNG QR code for the short URL. |
| `DELETE` | `/api/v1/links/{code}` | Implemented. Soft-disables a short link and invalidates its cache entry. |

Analytics returns exact `total_clicks` for the requested filter. If `from` and `to` are omitted, bucketed aggregations and recent clicks default to the last 90 days; user-agent aggregation is capped to the top 50 values. Daily and monthly buckets use `SHORTENER_TIME_ZONE`.

## Examples

```bash
curl -i http://localhost:8080/healthz
```

```bash
curl -i http://localhost:8080/metrics
```

```bash
curl -X POST http://localhost:8080/api/v1/shorten \
  -H 'Content-Type: application/json' \
  -d '{"original_url":"https://example.com"}'
```

```bash
curl -X POST http://localhost:8080/api/v1/shorten \
  -H 'Content-Type: application/json' \
  -d '{"original_url":"https://example.com","custom_alias":"my-link"}'
```

```bash
curl -i http://localhost:8080/s/abc1234
```

```bash
curl 'http://localhost:8080/api/v1/analytics/abc1234?from=2026-06-01&to=2026-06-12&recent_limit=20'
```

```bash
curl -o qr.png 'http://localhost:8080/api/v1/links/abc1234/qr?size=256'
```

```bash
curl -i -X DELETE http://localhost:8080/api/v1/links/abc1234
```

## Environment Variables

| Variable | Default | Description |
| --- | --- | --- |
| `SHORTENER_ENVIRONMENT` | `development` | Runtime environment. |
| `SHORTENER_TIME_ZONE` | `UTC` | Process time zone. |
| `SHORTENER_LOG_LEVEL` | `DEBUG` | Zap log level. |
| `SHORTENER_LOG_FOLDER` | `.out/logs` | Local log file directory. |
| `SHORTENER_HTTP_ADDR` | `:8080` | HTTP bind address. |
| `SHORTENER_HTTP_PUBLIC_BASE_URL` | `http://localhost:8080` | Public base URL used to build short URLs. |
| `SHORTENER_HTTP_SHUTDOWN_TIMEOUT` | `10s` | Graceful HTTP shutdown timeout. |
| `SHORTENER_HTTP_READ_HEADER_TIMEOUT` | `5s` | Read header timeout. |
| `SHORTENER_HTTP_READ_TIMEOUT` | `10s` | Read timeout. |
| `SHORTENER_HTTP_WRITE_TIMEOUT` | `10s` | Write timeout. |
| `SHORTENER_HTTP_IDLE_TIMEOUT` | `60s` | Idle timeout. |
| `SHORTENER_HTTP_ALLOWED_ORIGINS` | `*` | Comma-separated CORS origins. |
| `SHORTENER_HTTP_ALLOWED_METHODS` | `GET,POST,DELETE,OPTIONS` | Comma-separated CORS methods. |
| `SHORTENER_DATABASE_URL` | local PostgreSQL URL | PostgreSQL connection string. |
| `SHORTENER_POSTGRES_TIMEOUT` | `5s` | PostgreSQL operation timeout. |
| `SHORTENER_POSTGRES_MAX_CONNS` | `10` | PostgreSQL pool max connections. |
| `SHORTENER_POSTGRES_MIN_CONNS` | `2` | PostgreSQL pool min connections. |
| `SHORTENER_POSTGRES_MAX_CONN_IDLE_TIME` | `5m` | PostgreSQL pool idle lifetime. |
| `SHORTENER_REDIS_ADDR` | `localhost:6379` | Redis address. |
| `SHORTENER_REDIS_PASSWORD` | empty | Redis password. |
| `SHORTENER_REDIS_DB` | `0` | Redis database number. |
| `SHORTENER_REDIS_TIMEOUT` | `2s` | Redis dial, read, and write timeout. |
| `SHORTENER_REDIS_CACHE_TTL` | `10m` | Link resolution cache TTL. |
| `SHORTENER_REDIS_MISS_TTL` | `30s` | Negative cache TTL for missing or disabled links. |
| `WEB_PORT` | `5173` | Vite dev server port used by Docker Compose. |
| `PROMETHEUS_PORT` | `9090` | Prometheus UI port used by Docker Compose. |
| `GRAFANA_PORT` | `3000` | Grafana UI port used by Docker Compose. |
| `GRAFANA_ADMIN_USER` | `admin` | Local Grafana admin username. |
| `GRAFANA_ADMIN_PASSWORD` | `admin` | Local Grafana admin password. |

## Observability

Prometheus scrapes `shortener:8080/metrics` inside the Docker Compose network. Grafana uses file provisioning for the Prometheus datasource and dashboard, so the local observability stack is reproducible from version-controlled files.

Application metrics intentionally use low-cardinality labels only: HTTP method, route pattern, and status code. Dynamic values such as short codes, original URLs, IP addresses, user agents, referers, and request IDs are not used as Prometheus labels.
