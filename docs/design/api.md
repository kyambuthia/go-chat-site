# API Design

## Authentication

### Register

- **Endpoint**: `POST /api/register`
- **Request**:

```json
{
  "username": "alice",
  "password": "secret"
}
```

- **Response**: `201 Created`

```json
{
  "id": 1,
  "username": "alice"
}
```

### Login

- **Endpoint**: `POST /api/login`
- **Request**:

```json
{
  "username": "alice",
  "password": "secret"
}
```

- **Response**: `200 OK`

```json
{
  "token": "<jwt>"
}
```
