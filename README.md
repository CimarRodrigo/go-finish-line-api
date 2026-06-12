# FinishLine

Backend for a footrace participant registration system. Built with Go, following Hexagonal Architecture with DDD Lite.

## Requirements

- Go 1.26+
- Docker (to run the containerized app)
- [golangci-lint](https://golangci-lint.run/) (linting)

## Getting started

Create a `.env` file in the project root:

```
APP_ENV=development
APP_PORT=8080
DB_HOST=<database host>
DB_USER=<database user>
DB_PASSWORD=<database password>
DB_NAME=<database name>
DB_PORT=5432
DB_SSLMODE=require
```

Then:

```sh
make run          # run locally on :8080
# or
make up           # build and run with Docker
```

Verify it's alive:

```sh
curl http://localhost:8080/health
```

## Project structure

```
cmd/api/            # entrypoint: wiring, config, HTTP server
internal/
  common/           # cross-cutting: config, postgres, server
  <module>/         # one package per feature (user, auth, event, ...)
```

## Commands

| Command | Description |
|---|---|
| `make run` | Run the API locally |
| `make build` | Build binary to `bin/finishline` |
| `make test` | Run tests |
| `make lint` | Run golangci-lint |
| `make up` | Build image and start with Docker Compose |
| `make down` | Stop Docker Compose |
