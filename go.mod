module github.com/kyambuthia/go-chat-site

go 1.24.0

toolchain go1.24.7

require (
	github.com/golang-jwt/jwt/v5 v5.3.0
	github.com/mattn/go-sqlite3 v1.14.32
	golang.org/x/crypto v0.42.0
)

replace github.com/kyambuthia/go-chat-site => ./
