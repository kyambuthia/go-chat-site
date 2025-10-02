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

## Chat Demo

1.  Start the server: `cd server && go run cmd/main.go`
2.  Open two browser tabs to `http://localhost:8080`
3.  Register two different users, one in each tab.
4.  Log in as each user.
5.  Open the browser console and use the following commands to send messages:

```javascript
// In tab 1 (user A)
const wsA = new WebSocket(`ws://localhost:8080/ws`, ["Bearer", localStorage.getItem("token")]);
wsA.onmessage = (e) => console.log("A received:", e.data);
wsA.send(JSON.stringify({ type: "direct_message", to: "userB", body: "Hello from A" }));

// In tab 2 (user B)
const wsB = new WebSocket(`ws://localhost:8080/ws`, ["Bearer", localStorage.getItem("token")]);
wsB.onmessage = (e) => console.log("B received:", e.data);
wsB.send(JSON.stringify({ type: "direct_message", to: "userA", body: "Hello from B" }));
```
