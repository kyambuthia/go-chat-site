# API Design

## Authentication

### Register

- Endpoint: `POST /api/register`
- Request:

```json
{
  "username": "alice",
  "password": "secret123"
}
```

- Response: `201 Created`

```json
{
  "id": 1,
  "username": "alice"
}
```

### Login

- Endpoint: `POST /api/login`
- Response: `200 OK`

```json
{
  "token": "<jwt>"
}
```

## Invites

- `GET /api/invites`
- `POST /api/invites/send`
- `POST /api/invites/accept`
- `POST /api/invites/reject`

## Wallet

### Get Wallet

- Endpoint: `GET /api/wallet`

```json
{
  "id": 1,
  "user_id": 10,
  "balance": 12.34,
  "balance_cents": 1234
}
```
