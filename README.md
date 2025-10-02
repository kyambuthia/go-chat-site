a go chat site - backend written using golang, frontend written using vite - smol db is sqlite3

## Setup and Testing

### Server

Migrations are automatically run at startup. To run the tests:

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
