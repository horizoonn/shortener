# Shortener architecture

## Goal

Shortener is a small production-style Go backend service for shortening URLs and collecting click analytics.

## Core flows

### Create short link

```text
POST /api/v1/shorten
    -> decode JSON
    -> validate URL and optional custom alias
    -> generate code if alias is not provided
    -> insert into PostgreSQL
    -> return short URL
```

### Redirect

```text
GET /s/{code}
    -> try Redis
    -> on cache miss read PostgreSQL
    -> cache link resolution
    -> insert click analytics
    -> redirect to original URL
```

### Analytics

```text
GET /api/v1/analytics/{code}
    -> resolve link by code
    -> aggregate clicks from PostgreSQL
    -> return totals, daily/monthly stats, User-Agent stats
```

## Package map

```text
internal/app                  dependency wiring and app lifecycle
internal/config               env config
internal/logger               zap logger
internal/errors               sentinel errors
internal/httpapi              common HTTP helpers
internal/links                link domain and use cases
internal/links/postgres       PostgreSQL repository
internal/links/redis          Redis cache
internal/analytics            analytics use cases
internal/analytics/postgres   PostgreSQL analytics repository
```

## Persistence

PostgreSQL is the source of truth.

Redis is only a cache-aside layer for link resolution.

## First version priorities

1. Clean structure.
2. Working API.
3. Correct migrations.
4. Useful tests.
5. Docker Compose and Makefile.
6. README and OpenAPI.
7. Simple static UI.
