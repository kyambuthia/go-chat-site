a go chat site - backend written in golang, frontend written using vite, sqlite for storage.

## Development

This project uses environment variables for configuration.

### 1. Configure Environment

Create `.env` files as needed.

Server env (required):

```bash
JWT_SECRET=replace-with-a-long-secret
PORT=8080
```

Client env (optional):

```bash
VITE_API_URL=http://localhost:8080
VITE_WS_URL=ws://localhost:8080
VITE_API_BASE_URL=http://localhost:8080
```

Notes:
- `VITE_API_BASE_URL` is the explicit API origin used by the client in split-origin deployments.
- `VITE_WS_URL` can be either `ws(s)://host` or `ws(s)://host/ws`.

### 2. Run Backend

```bash
# from project root
go run ./server/cmd/main.go
```

The server now runs migrations automatically on startup.

### 3. Run Frontend

```bash
cd client
bun install
bun run dev
```

Navigate to the URL printed by Vite (usually `http://localhost:5173`).
