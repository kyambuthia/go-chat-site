# Load Test Scenarios

These scripts target critical paths:
- `login.js`: login endpoint pressure
- `ws-handshake.js`: websocket handshake pressure
- `messaging-relay.js`: direct-message relay pressure

## Prerequisites
- [k6](https://k6.io/) installed
- local server running at `http://localhost:8080`

## Run

```bash
k6 run server/test/load/login.js
k6 run -e JWT_TOKEN=<token> server/test/load/ws-handshake.js
k6 run -e SENDER_JWT=<token> -e RECIPIENT_USERNAME=<username> server/test/load/messaging-relay.js
```
