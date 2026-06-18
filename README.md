# Shortener

Shortener is a Go backend service for URL shortening and click analytics. This bootstrap stage includes the project skeleton, configuration, file and stdout logging, HTTP wiring, Docker environment, and `/healthz`.

## Stack

- Go with standard `net/http`
- PostgreSQL as the source of truth
- Redis as cache-aside storage for link resolution
- `github.com/kelseyhightower/envconfig` for configuration
- `go.uber.org/zap` for logging
- `golang-migrate` for migrations
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
web/public                 static assets
docs/openapi.yaml          OpenAPI contract
```

## Quick Start

```bash
cp .env.example .env
make env-up
make shortener-run
curl http://localhost:8080/healthz
```

## Makefile Commands

| Command | Description |
| --- | --- |
| `make help` | Show available commands. |
| `make fmt` | Format Go files. |
| `make fmt-check` | Check Go formatting. |
| `make vet` | Run `go vet ./...`. |
| `make test` | Run `go test ./...`. |
| `make test-race` | Run tests with the race detector. |
| `make test-integration` | Run integration tests with the `integration` tag. |
| `make check` | Run formatting, vet, and unit tests. |
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

## API

| Method | Path | Status |
| --- | --- | --- |
| `GET` | `/healthz` | Implemented. |
| `GET` | `/readyz` | Implemented. |
| `POST` | `/api/v1/shorten` | Implemented. |
| `GET` | `/s/{code}` | Planned. |
| `GET` | `/api/v1/analytics/{code}` | Planned. |

## Examples

```bash
curl -i http://localhost:8080/healthz
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

Future API examples:

```bash
curl -i http://localhost:8080/s/abc1234
curl http://localhost:8080/api/v1/analytics/abc1234
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
| `SHORTENER_HTTP_ALLOWED_METHODS` | `GET,POST,OPTIONS` | Comma-separated CORS methods. |
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
