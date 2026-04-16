module github.com/kyambuthia/go-chat-site

go 1.25.0

toolchain go1.25.9

require (
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/gorilla/websocket v1.5.3
	github.com/mattn/go-sqlite3 v1.14.42
	golang.org/x/crypto v0.50.0
)

replace github.com/kyambuthia/go-chat-site => ./
