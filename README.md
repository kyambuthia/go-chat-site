a go chat site - backend written using golang, frontend written using vite - smol db is sqlite3

## Setup and Testing

### Server

Migrations are automatically run at startup. To run the server:

```bash
cd server
go run cmd/main.go
```

To run the tests:

```bash
cd server
go test ./...
```

### Client

```bash
cd client
bun install
bun test
```

## API

See [docs/design/api.md](docs/design/api.md) for API documentation.