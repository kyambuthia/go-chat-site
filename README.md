a go chat site - backend written using golang, frontend written using vite - smol db is sqlite3

## Development

This project uses environment variables for configuration. While it can run with default settings, you can customize ports and URLs by creating environment files.

### 1. Configure the Client (Optional)

The client is configured to connect to a backend running on `http://localhost:8080`. To customize this, create a file named `client/.env` and add the following variables:

```
VITE_API_URL=http://localhost:8080
VITE_WS_URL=ws://localhost:8080
```

### 2. Run the Backend Server

The server runs on port `8080` by default. You can override this by setting the `PORT` environment variable.

```bash
# In your first terminal, from the project root
go run ./server/cmd/main.go

# To run on a different port (e.g., 8888)
PORT=8888 go run ./server/cmd/main.go
```

### 3. Run the Frontend Client

```bash
# In your second terminal
cd client
bun install
bun run dev
```

Navigate to the URL provided by the Vite dev server (usually `http://localhost:5173`). The client will proxy API requests to the Go backend.
