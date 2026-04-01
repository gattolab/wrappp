# Wrappp

> **Live:** [wrappp.link](https://wrappp.link/)  
> **Frontend repo:** [gattolab/wrappp-web](https://github.com/gattolab/wrappp-web)

A high-performance **URL shortener** service built with Go. It converts long URLs into short, shareable links, tracks click counts, and redirects visitors to the original destination.

---

## Features

- **URL shortening** — generates an 8-character short code (KSUID-based)
- **Redirect** — instant HTTP 302 redirect from short code to original URL
- **Click tracking** — batched, non-blocking click counter
- **Expiry support** — optional expiration timestamp per URL
- **Redis/In-memory caching** — standalone or cluster mode

---
##  Getting Started

### Prerequisites

- Go 1.26+
- PostgreSQL
- Redis/In-memory cache (optional, for performance)

### 1. Clone the repository

```bash
git clone https://github.com/gattolab/wrappp.git
cd wrappp
```

### 2. Configure environment

Create a `.env` file in the project root (see [Environment Variables](#environment-variables) below).

```bash
cp .env.example .env   # adjust values as needed
```

### 3. Run database migration

Execute the SQL script against your PostgreSQL database:

```bash
psql -U wrappp -d wrappp -f pkg/db/migration/initail.sql
```

### 4. Run the service

```bash
go run ./cmd
```

Or build and run:

```bash
make build
./bin/wrappp
```

### 5. Docker

```bash
docker build -t wrappp .
docker run -p 3000:3000 --env-file .env wrappp
```

---

### Redirect

```
GET https://wrappp.link/api/r/abc12345  →  302 https://example.com/very/long/path
```

---

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `SERVER_PORT` | `:8080` | HTTP listen port |
| `SERVER_BASE_URI` | `` | Base URI prefix |
| `SERVER_MODE` | `debug` | App mode (`debug` / `release`) |
| `SERVER_CACHE_DEPLOYMENT_TYPE` | `1` | `1` = Redis standalone, `2` = Redis cluster |
| `DATABASE_HOST` | `localhost` | PostgreSQL host |
| `DATABASE_PORT` | `6432` | PostgreSQL port (PgBouncer default) |
| `DATABASE_USER` | `postgres` | PostgreSQL user |
| `DATABASE_PASSWORD` | `` | PostgreSQL password |
| `DATABASE_NAME` | `wrappp` | PostgreSQL database name |
| `DATABASE_READ_NAME` | `` | Read-replica DB name (PgBouncer) |
| `DATABASE_MAX_POOL_OPEN` | `50` | Max open DB connections |
| `REDIS_ADDRESS` | `` | Redis host:port |
| `REDIS_PASSWORD` | `` | Redis password |
| `REDIS_DB` | `0` | Redis DB index |
| `REDIS_POOL_SIZE` | `` | Redis connection pool size |
| `REDIS_CLUSTER_ADDRESS` | `` | Redis cluster addresses (comma-separated) |
| `AUTHORIZATION_JWT_SECRET` | `ais-jwt` | JWT signing secret |
| `AUTHORIZATION_JWT_EXPIRATION` | `3600` | Access token TTL (seconds) |
| `AUTHORIZATION_JWT_REFRESH_EXPIRATION` | `360000` | Refresh token TTL (seconds) |
| `LOGGER_LEVEL` | `` | Log level (`debug`, `info`, `warn`, `error`) |
| `LOGGER_ENCODING` | `json` | Log format (`json` / `console`) |

---

## Click Batcher

Click counting uses an in-memory `ClickBatcher` to avoid a DB write on every redirect:

- Clicks are queued in a **buffered channel** (16 384 slots).
- A background goroutine **flushes aggregated counts** to PostgreSQL every **1 second** or when the batch reaches **500** entries.
- Under extreme back-pressure (channel full) clicks are **dropped gracefully** rather than blocking the HTTP response.

---

## Testing

```bash
# Run all tests with coverage
make test

# Run unit tests only
make unittest

# Lint
make lint
```

---

## Make Targets

| Target | Description |
|---|---|
| `make build` | Compile binary to `bin/wrappp` |
| `make test` | Run tests with atomic coverage |
| `make unittest` | Run unit tests (short mode) |
| `make coverage` | Generate `coverage.out` |
| `make lint` | Run `golangci-lint` |
| `make clean` | Remove `bin/` directory |

---

## License

MIT © [Gatto Lab](https://github.com/gattolab)
