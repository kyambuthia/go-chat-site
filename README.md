a go chat site - backend written in golang, frontend written using vite, sqlite for storage.

## Quick Start

### 1. Configure environment

```bash
cp .env.example .env
cp client/.env.example client/.env
```

Notes:
- `VITE_API_BASE_URL` is the explicit API origin used by the client in split-origin deployments.
- `VITE_WS_URL` can be either `ws(s)://host` or `ws(s)://host/ws`.

### 2. Install frontend dependencies

```bash
make setup
```

### 3. Run backend

```bash
go run ./server/cmd/main.go
```

The server runs migrations automatically on startup.

### 4. Run frontend

```bash
cd client && npm run dev
```

Navigate to `http://localhost:5173`.

## Documentation

Project documentation is organized under `docs/`:
- `docs/overview/` for product direction
- `docs/architecture/` for system contracts and ADRs
- `docs/roadmaps/` for delivery plans
- `docs/product/` for marketplace/payment strategy
- `docs/operations/` for runbooks and operator procedures

## Verify Locally

```bash
make check
```

`make check` runs:
- go format + vet checks
- backend tests
- frontend lint + typecheck + tests
- backend + frontend builds
