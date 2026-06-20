# Shortener

REST API приложение на Go для сокращения URL с аналитикой кликов, кэшированием в Redis, rate limiting и React фронтендом. Сервис принимает длинные URL, генерирует короткие коды, сохраняет их в PostgreSQL, кэширует в Redis и перенаправляет пользователей на оригинальные URL с записью аналитики.

Проект собран как небольшой backend-сервис с PostgreSQL, Redis, Prometheus, Grafana, Swagger UI и React фронтендом.

## Технологический Стек

| Компонент | Технология |
| --- | --- |
| Язык | Go 1.26+ |
| HTTP | стандартный `net/http` |
| База данных | PostgreSQL, `jackc/pgx/v5` |
| Кеш | Redis, `redis/go-redis/v9` |
| Логирование | `go.uber.org/zap` |
| Конфигурация | `kelseyhightower/envconfig` |
| Валидация | `go-playground/validator/v10` |
| Rate Limiting | `golang.org/x/time/rate` |
| API contract | OpenAPI 3.1, Swagger UI |
| Миграции | `golang-migrate` |
| QR коды | `skip2/go-qrcode` |
| Метрики | Prometheus, Grafana |
| Тестирование | `testing`, `httptest`, `testcontainers` |
| Фронтенд | Vite, React, TypeScript, Tailwind CSS, shadcn/ui |
| Запуск окружения | Docker Compose, Makefile |

## Архитектура

Основной pipeline:

```text
HTTP POST /api/v1/shorten
    |
    | validate URL, generate code or use custom alias
    v
PostgreSQL links
    |
    | cache link in Redis
    v
HTTP GET /s/{code}
    |
    | resolve from Redis cache or PostgreSQL
    | record click analytics (best-effort)
    v
302 redirect to original URL
```

Статусы ссылок:

| Статус | Значение |
| --- | --- |
| `active` | Ссылка активна и перенаправляет |
| `disabled` | Ссылка отключена, возвращает 404 |
| `expired` | Ссылка истекла по времени, возвращает 404 |

## Архитектурные Решения

Проект разделён по слоям:

```text
Transport HTTP
    |
    | decode request, path/query params, response mapping
    v
Service
    |
    | business logic, validation, code generation
    v
Repository
    |
    | PostgreSQL access
    v
Domain
```

Интерфейсы объявляются на стороне потребителя. Например, сервис ссылок зависит от интерфейса `LinksRepository`, а конкретная реализация `postgres.Repository` подключается вручную в `internal/app/app.go`.

### Redis Cache

PostgreSQL остаётся источником истины. Redis используется как cache-aside кеш для разрешения коротких кодов:

- активная ссылка: `shortener:link:{code}`;
- негативный кеш: `shortener:link-miss:{code}`.

TTL кэша рассчитывается как `min(configured_ttl, time_until_expiration)` для ссылок с `expires_at`. При инвалидации ссылки удаляются оба ключа.

Redis не обязателен при старте приложения: если подключение не проходит, сервис работает напрямую с PostgreSQL. В runtime ошибки чтения/записи кеша не ломают основную бизнес-операцию.

### Rate Limiting

In-memory rate limiter на основе `golang.org/x/time/rate` защищает API от злоупотреблений:

- Лимиты настраиваются через `SHORTENER_RATE_LIMIT_RPS` и `SHORTENER_RATE_LIMIT_BURST`;
- Очистка неактивных клиентов каждые 10 минут;
- Служебные endpoints (`/healthz`, `/readyz`, `/metrics`, `/docs`) исключены из rate limiting;
- Поддержка trusted proxies через `SHORTENER_TRUSTED_PROXIES` для корректного определения IP за proxy.

### Link Expiration

Ссылки могут иметь опциональное время истечения `expires_at`:

- При создании передаётся ISO 8601 timestamp;
- Redis TTL автоматически уменьшается до времени истечения;
- При запросе истекшей ссылки возвращается 404;
- Аналитика остаётся доступной для истекших ссылок.

## Структура Проекта

```text
.
── .github/
│   └── workflows/
│       └── ci.yml                          # GitHub Actions: tests, vet, lint, build
── cmd/
│   └── shortener/
│       ├── Dockerfile                      # Multi-stage Docker build приложения
│       └── main.go                         # Точка входа: конфигурация, запуск HTTP server
├── docs/
│   └── openapi.yaml                        # OpenAPI 3.1 contract
├── internal/
│   ├── analytics/
│   │   ├── click.go                        # Domain models для кликов
│   │   ├── postgres/                       # PostgreSQL repository для аналитики
│   │   └── service/                        # Service layer для аналитики
│   ├── app/
│   │   └── app.go                          # Dependency wiring и lifecycle
│   ├── config/
│   │   └── config.go                       # Environment configuration
│   ├── docs/
│   │   ├── docs.go                         # Embedded OpenAPI spec и Swagger UI HTML
│   │   ├── openapi.yaml                    # OpenAPI 3.1 specification
│   │   └── swagger.html                    # Swagger UI HTML template
│   ├── errors/
│   │   └── errors.go                       # Sentinel domain errors
│   ├── httpapi/
│   │   ├── middleware/
│   │   │   ├── chain.go                    # Middleware chaining
│   │   │   ├── cors.go                     # CORS middleware
│   │   │   ├── rate_limit.go               # Rate limiting middleware
│   │   │   ├── recovery.go                 # Panic recovery middleware
│   │   │   ├── request_id.go               # Request ID middleware
│   │   │   └── request_logger.go           # Request logging middleware
│   │   ├── request/
│   │   │   ├── ip.go                       # IP resolver с trusted proxies
│   │   │   ├── json.go                     # JSON decode и validation
│   │   │   ├── path.go                     # Path parameter helpers
│   │   │   ── query.go                    # Query parameter helpers
│   │   ├── response/
│   │   │   ├── handler.go                  # Response handler
│   │   │   ├── json.go                     # JSON response helpers
│   │   │   └── writer.go                   # Custom ResponseWriter
│   │   └── server/
│   │       ├── config.go                   # HTTP server config
│   │       ├── docs.go                     # Swagger UI routes
│   │       ├── health.go                   # Health check route
│   │       ├── route.go                    # Route definition
│   │       ├── router.go                   # API version router
│   │       └── server.go                   # HTTP server
│   ├── links/
│   │   ├── generator.go                    # Random code generator
│   │   ├── link.go                         # Link domain model
│   │   ├── postgres/                       # PostgreSQL repository
│   │   ├── redis/                          # Redis cache
│   │   ├── service/                        # Link service layer
│   │   ├── transport/http/                 # HTTP handlers и DTO
│   │   └── validation.go                   # Domain validation
│   ├── logger/
│   │   └── logger.go                       # Zap logger setup
│   ├── observability/metrics/
│   │   ├── infrastructure.go               # Postgres и Redis pool metrics
│   │   ├── metrics.go                      # Application metrics
│   │   └── route.go                        # Metrics route
│   ├── qr/
│   │   ── generator.go                    # QR code generator
│   └── storage/postgres/pool/
│       ├── errors.go                       # PostgreSQL error mapping
│       ├── metrics.go                      # Pool metrics wrapper
│       ├── pgx/                            # pgx adapter
│       └── pool.go                         # Pool interface
├── migrations/                             # SQL migrations
├── observability/                          # Prometheus и Grafana provisioning
├── web/                                    # Vite React frontend
│   ├── src/
│   │   ├── App.tsx                         # Main application component
│   │   ├── components/                     # React components
│   │   ├── lib/                            # API client и helpers
│   │   ── types.ts                        # TypeScript types
│   ── public/                             # Built frontend assets
├── .dockerignore                           # Исключения из Docker build context
├── .env.example                            # Пример локальной конфигурации
├── .gitignore                              # Git exclusions
├── .golangci.yml                           # golangci-lint configuration
├── docker-compose.yaml                     # App, PostgreSQL, Redis, Prometheus, Grafana, migrations
├── Makefile
└── README.md
```

## Быстрый Старт

### Требования

- Docker и Docker Compose
- Go 1.26+
- Node.js 20+
- `make`

### Настройка `.env`

```bash
cp .env.example .env
```

Для локального запуска можно оставить значения из `.env.example`.

### Локальный Запуск Go-Приложения

Этот режим запускает PostgreSQL и Redis в Docker, а Go-приложение запускается локально через `go run`.

```bash
make env-up
make migrate-up
make shortener-run
```

После запуска:

- API: `http://localhost:8080/api/v1`
- Swagger UI: `http://localhost:8080/docs`
- Health check: `http://localhost:8080/healthz`
- Metrics: `http://localhost:8080/metrics`

### Запуск Go-Приложения В Docker

Этот режим поднимает PostgreSQL, Redis, применяет миграции и запускает само Go-приложение в Docker container.

```bash
make env-up
make shortener-deploy
```

`make shortener-deploy` делает `docker compose up -d --build shortener`. Сервис `shortener` зависит от:

- `postgres` в состоянии `healthy`;
- `migrate` в состоянии `service_completed_successfully`;
- `redis` в состоянии `healthy`.

После запуска:

- API: `http://localhost:8080/api/v1`
- Swagger UI: `http://localhost:8080/docs`

Посмотреть логи приложения:

```bash
make shortener-logs
```

Остановить и удалить только container приложения:

```bash
make shortener-undeploy
```

### Запуск С Фронтендом

Для разработки с React фронтендом:

```bash
make dev-up
```

Открыть:

```text
http://localhost:5173
```

Vite dev server проксирует API и redirect запросы на backend container. Если `web/node_modules` отсутствует, `make dev-up` устанавливает frontend зависимости через `npm ci` перед запуском Compose.

Остановить:

```bash
make dev-down
```

### Запуск С Observability

Для запуска с Prometheus и Grafana:

```bash
make observability-up
```

Открыть:

```text
Prometheus: http://localhost:9090
Grafana:    http://localhost:3000
```

Default Grafana credentials: `admin` / `admin`. Prometheus datasource и `Shortener Overview` dashboard provisioned из репозитория. Backend exposes metrics at `GET /metrics`; в production deployments этот endpoint должен быть доступен только Prometheus или доверенным внутренним сетям.

## Makefile Команды

| Команда | Описание |
| --- | --- |
| `make help` | Показать доступные команды |
| `make fmt` | Форматировать Go файлы |
| `make fmt-check` | Проверить Go форматирование |
| `make vet` | Запустить `go vet ./...` |
| `make lint` | Запустить `golangci-lint` |
| `make staticcheck` | Запустить Staticcheck |
| `make actionlint` | Lint GitHub Actions workflows |
| `make test` | Запустить `go test ./...` |
| `make test-cover` | Запустить unit tests с coverage summary |
| `make test-cover-func` | Показать unit test coverage по функциям |
| `make test-cover-html` | Записать unit test coverage HTML report |
| `make test-race` | Запустить tests с race detector |
| `make test-integration` | Запустить integration tests с тегом `integration` |
| `make test-integration-cover` | Показать integration coverage по функциям |
| `make test-integration-cover-html` | Записать integration coverage HTML report |
| `make check` | Запустить formatting, vet, golangci-lint, actionlint и unit tests |
| `make check-all` | Запустить все проверки |
| `make env-up` | Поднять PostgreSQL и Redis |
| `make env-down` | Остановить Docker Compose services |
| `make env-cleanup` | Остановить services и удалить volumes |
| `make migrate-create name=...` | Создать migration |
| `make migrate-up` | Применить миграции |
| `make migrate-down` | Откатить одну миграцию |
| `make shortener-run` | Запустить приложение локально |
| `make shortener-deploy` | Собрать и запустить приложение в Docker |
| `make shortener-undeploy` | Остановить и удалить Docker container приложения |
| `make shortener-logs` | Смотреть логи Docker container приложения |
| `make observability-up` | Запустить backend, Prometheus и Grafana |
| `make observability-down` | Остановить Prometheus и Grafana |
| `make observability-logs` | Смотреть логи Prometheus и Grafana |
| `make web-install` | Установить frontend зависимости |
| `make web-dev` | Запустить Vite dev server локально |
| `make web-build` | Собрать frontend assets в `web/public` |
| `make web-lint` | Lint frontend code |
| `make web-audit` | Audit frontend dependencies |
| `make web-check` | Запустить frontend lint, build и audit |
| `make dev-up` | Запустить backend и frontend с Docker Compose |
| `make dev-down` | Остановить Docker Compose development environment |
| `make dev-logs` | Смотреть логи backend и frontend |

## API

Все endpoints находятся под префиксом:

```text
/api/v1
```

| Метод | Путь | Описание |
| --- | --- | --- |
| `POST` | `/api/v1/shorten` | Создать короткую ссылку |
| `GET` | `/s/{code}` | Перенаправить на оригинальный URL |
| `GET` | `/api/v1/analytics/{code}` | Получить аналитику кликов |
| `DELETE` | `/api/v1/links/{code}` | Отключить короткую ссылку |
| `GET` | `/api/v1/links/{code}/qr` | Получить QR-код для короткой ссылки |
| `GET` | `/healthz` | Health check |
| `GET` | `/readyz` | Readiness check |
| `GET` | `/metrics` | Prometheus/OpenMetrics метрики |
| `GET` | `/docs` | Swagger UI |
| `GET` | `/docs/openapi.yaml` | OpenAPI specification |
| `GET` | `/` | Static UI (React frontend) |

### POST /api/v1/shorten

Создать короткую ссылку.

Request body:

```json
{
  "original_url": "https://example.com/very/long/url",
  "custom_alias": "my-link",
  "expires_at": "2026-12-31T23:59:59Z"
}
```

- `original_url` (required) - оригинальный URL, должен начинаться с `http://` или `https://`;
- `custom_alias` (optional) - пользовательский alias, 3-64 символа, только буквы, цифры, `_`, `-`;
- `expires_at` (optional) - время истечения ссылки в ISO 8601 формате.

Response `201 Created`:

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "code": "abc12345",
  "original_url": "https://example.com/very/long/url",
  "short_url": "http://localhost:8080/s/abc12345",
  "is_custom": false,
  "created_at": "2026-06-18T10:00:00Z",
  "expires_at": "2026-12-31T23:59:59Z"
}
```

### GET /s/{code}

Перенаправить на оригинальный URL. Записывает аналитику клика (best-effort).

Response `302 Found` с `Location` header.

### GET /api/v1/analytics/{code}

Получить аналитику кликов для короткой ссылки.

Query parameters:

- `from` (optional) - начальная дата в формате `YYYY-MM-DD`;
- `to` (optional) - конечная дата в формате `YYYY-MM-DD`;
- `recent_limit` (optional, default: 20, max: 100) - количество последних кликов.

Response `200 OK`:

```json
{
  "code": "abc12345",
  "original_url": "https://example.com",
  "total_clicks": 42,
  "clicks_by_day": [
    { "day": "2026-06-18", "clicks": 10 }
  ],
  "clicks_by_month": [
    { "month": "2026-06", "clicks": 42 }
  ],
  "clicks_by_user_agent": [
    { "user_agent": "Mozilla/5.0", "clicks": 30 }
  ],
  "recent_clicks": [
    {
      "clicked_at": "2026-06-18T10:00:00Z",
      "user_agent": "Mozilla/5.0",
      "referer": "https://example.com",
      "ip": "192.168.1.1"
    }
  ]
}
```

Аналитика возвращает точное `total_clicks` для запрошенного фильтра. Если `from` и `to` не указаны, bucketed aggregations и recent clicks по умолчанию за последние 90 дней; user-agent aggregation ограничена топ-50 значениями. Daily и monthly buckets используют `SHORTENER_TIME_ZONE`.

### DELETE /api/v1/links/{code}

Отключить короткую ссылку. Soft-delete: ссылка помечается как `disabled`, кэш инвалидируется, историческая аналитика остаётся доступной.

Response `204 No Content`.

### GET /api/v1/links/{code}/qr

Получить PNG QR-код для короткой ссылки.

Query parameters:

- `size` (optional, default: 256, min: 128, max: 1024) - размер QR-кода в пикселях.

Response `200 OK` с `Content-Type: image/png` и `Cache-Control: public, max-age=3600`.

### OpenAPI

OpenAPI contract лежит в:

```text
docs/openapi.yaml
```

Он описывает публичный HTTP API: paths, query/path parameters, request body, response schemas, status codes, examples и ошибки. В нём не хранятся секреты, токены или runtime-конфигурация.

Swagger UI доступен внутри приложения по адресу `/docs`. Он загружает спецификацию с `/docs/openapi.yaml` и отображает интерактивную документацию.

## Примеры Curl

### Создать короткую ссылку

```bash
curl -i -X POST http://localhost:8080/api/v1/shorten \
  -H 'Content-Type: application/json' \
  -d '{"original_url":"https://example.com/very/long/url"}'
```

### Создать ссылку с custom alias

```bash
curl -i -X POST http://localhost:8080/api/v1/shorten \
  -H 'Content-Type: application/json' \
  -d '{"original_url":"https://example.com","custom_alias":"my-link"}'
```

### Создать ссылку с expiration

```bash
curl -i -X POST http://localhost:8080/api/v1/shorten \
  -H 'Content-Type: application/json' \
  -d '{"original_url":"https://example.com","expires_at":"2026-12-31T23:59:59Z"}'
```

### Перенаправить

```bash
curl -i http://localhost:8080/s/abc12345
```

### Получить аналитику

```bash
curl -i 'http://localhost:8080/api/v1/analytics/abc12345?from=2026-06-01&to=2026-06-18&recent_limit=20'
```

### Получить QR-код

```bash
curl -o qr.png 'http://localhost:8080/api/v1/links/abc12345/qr?size=256'
```

### Отключить ссылку

```bash
curl -i -X DELETE http://localhost:8080/api/v1/links/abc12345
```

### Health check

```bash
curl -i http://localhost:8080/healthz
```

### Metrics

```bash
curl -i http://localhost:8080/metrics
```

## Переменные Окружения

| Переменная | Описание | Пример |
| --- | --- | --- |
| `SHORTENER_ENVIRONMENT` | Runtime environment | `development` |
| `SHORTENER_TIME_ZONE` | IANA timezone приложения | `UTC` |
| `SHORTENER_LOG_LEVEL` | Уровень логирования | `DEBUG` |
| `SHORTENER_LOG_FOLDER` | Папка log-файлов | `.out/logs` |
| `SHORTENER_HTTP_ADDR` | Адрес HTTP server | `:8080` |
| `SHORTENER_HTTP_PUBLIC_BASE_URL` | Public base URL для построения коротких ссылок | `http://localhost:8080` |
| `SHORTENER_HTTP_SHUTDOWN_TIMEOUT` | Graceful HTTP shutdown timeout | `10s` |
| `SHORTENER_HTTP_READ_HEADER_TIMEOUT` | Таймаут чтения HTTP headers | `5s` |
| `SHORTENER_HTTP_READ_TIMEOUT` | Таймаут чтения HTTP request | `10s` |
| `SHORTENER_HTTP_WRITE_TIMEOUT` | Таймаут записи HTTP response | `10s` |
| `SHORTENER_HTTP_IDLE_TIMEOUT` | Таймаут idle HTTP connections | `60s` |
| `SHORTENER_HTTP_ALLOWED_ORIGINS` | CORS origins через запятую | `*` |
| `SHORTENER_HTTP_ALLOWED_METHODS` | CORS methods через запятую | `GET,POST,DELETE,OPTIONS` |
| `SHORTENER_RATE_LIMIT_RPS` | Rate limit requests per second | `20` |
| `SHORTENER_RATE_LIMIT_BURST` | Rate limit burst size | `40` |
| `SHORTENER_TRUSTED_PROXIES` | Доверенные proxy CIDRs через запятую | `127.0.0.1/32,10.0.0.0/8` |
| `SHORTENER_DATABASE_URL` | PostgreSQL connection string | `postgres://shortener:shortener@localhost:5432/shortener?sslmode=disable` |
| `SHORTENER_POSTGRES_TIMEOUT` | Таймаут операций PostgreSQL | `5s` |
| `SHORTENER_POSTGRES_MAX_CONNS` | Максимум соединений PostgreSQL pool | `10` |
| `SHORTENER_POSTGRES_MIN_CONNS` | Минимум соединений PostgreSQL pool | `2` |
| `SHORTENER_POSTGRES_MAX_CONN_IDLE_TIME` | Максимальное idle-время соединения | `5m` |
| `SHORTENER_REDIS_ADDR` | Redis address | `localhost:6379` |
| `SHORTENER_REDIS_PASSWORD` | Redis password | пусто |
| `SHORTENER_REDIS_DB` | Redis database number | `0` |
| `SHORTENER_REDIS_TIMEOUT` | Таймаут Redis операций | `2s` |
| `SHORTENER_REDIS_CACHE_TTL` | Link resolution cache TTL | `10m` |
| `SHORTENER_REDIS_MISS_TTL` | Negative cache TTL для отсутствующих или отключённых ссылок | `30s` |
| `POSTGRES_DB` | Имя БД для Docker Compose | `shortener` |
| `POSTGRES_USER` | Пользователь БД для Docker Compose | `shortener` |
| `POSTGRES_PASSWORD` | Пароль БД для Docker Compose | `shortener` |
| `POSTGRES_PORT` | Port PostgreSQL для Docker Compose | `5432` |
| `REDIS_PORT` | Port Redis для Docker Compose | `6379` |
| `SHORTENER_PORT` | Port приложения для Docker Compose | `8080` |
| `WEB_PORT` | Vite dev server port для Docker Compose | `5173` |
| `PROMETHEUS_PORT` | Prometheus UI port для Docker Compose | `9090` |
| `GRAFANA_PORT` | Grafana UI port для Docker Compose | `3000` |
| `GRAFANA_ADMIN_USER` | Local Grafana admin username | `admin` |
| `GRAFANA_ADMIN_PASSWORD` | Local Grafana admin password | `admin` |

## Миграции

Создать миграцию:

```bash
make migrate-create name=add_some_column
```

Применить:

```bash
make migrate-up
```

Откатить:

```bash
make migrate-down
```

После применения миграции в реальной БД лучше не редактировать старые migration files. Для изменений схемы создавай новую миграцию.

## Тестирование И Проверки

Unit tests:

```bash
make test
```

Vet:

```bash
make vet
```

Lint:

```bash
make lint
```

Build:

```bash
make build
```

Полная локальная проверка:

```bash
make check
```

Все проверки:

```bash
make check-all
```

## Observability

Prometheus scrapes `shortener:8080/metrics` внутри Docker Compose network. Grafana использует file provisioning для Prometheus datasource и dashboard, так что local observability stack воспроизводим из version-controlled files.

Application metrics намеренно используют только low-cardinality labels: HTTP method, route pattern и status code. Dynamic values такие как short codes, original URLs, IP addresses, user agents, referers и request IDs не используются как Prometheus labels.

## Безопасность

### Rate Limiting

In-memory rate limiter защищает API от злоупотреблений. Служебные endpoints (`/healthz`, `/readyz`, `/metrics`, `/docs`) исключены из rate limiting, чтобы злоумышленник не мог заблокировать health checks или metrics collection.

### Trusted Proxies

При работе за proxy (например, nginx, load balancer) настройте `SHORTENER_TRUSTED_PROXIES` для корректного определения IP клиентов. Если proxy не доверенный, `X-Forwarded-For` и `X-Real-IP` headers игнорируются, и используется только `RemoteAddr`.

### Content-Type Validation

POST endpoints требуют `Content-Type: application/json`. Запросы с другим content-type отклоняются с `400 Bad Request`.

### Reserved Aliases

Следующие aliases зарезервированы и не могут быть использованы как custom alias:

- `healthz`, `readyz`, `api`, `swagger`, `static`, `assets`, `docs`, `s`

Это предотвращает конфликты с служебными routes.
