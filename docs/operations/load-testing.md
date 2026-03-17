# Load Testing

## Scope
Lightweight k6 scenarios live under `server/test/load/` and cover a few critical runtime paths:
- `login.js`: login endpoint pressure
- `ws-handshake.js`: WebSocket handshake pressure
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

## Source
The scenario scripts remain next to backend tests in `server/test/load/README.md` and `server/test/load/*.js`.
