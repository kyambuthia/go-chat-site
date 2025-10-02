a go chat site - backend written using golang, frontend written using vite - smol db is sqlite3

## Development

This project requires two separate terminal sessions to run:

**1. Run the Backend Server**

```bash
# In your first terminal
cd server
go run cmd/main.go
```

**2. Run the Frontend Client**

```bash
# In your second terminal
cd client
bun install
bun run dev
```

Navigate to the URL provided by the Vite dev server (usually `http://localhost:5173`).

## Testing

### Server

```bash
cd server
go test ./...
```

### Client

Client-side tests were removed during the migration to Vite and need to be re-written using a React-compatible testing framework.

## API

See [docs/design/api.md](docs/design/api.md) for API documentation.