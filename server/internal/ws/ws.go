package ws

import (
	"net/http"

	"github.com/kyambuthia/go-chat-site/server/internal/adapters/transport/wsrelay"
)

type Authenticator = wsrelay.Authenticator
type Message = wsrelay.Message
type Hub = wsrelay.Hub

func NewHub() *Hub {
	return wsrelay.NewHub()
}

func WebSocketHandler(h *Hub, authenticator Authenticator, resolveToUserID func(username string) (int, error)) http.HandlerFunc {
	return wsrelay.WebSocketHandler(h, authenticator, resolveToUserID)
}

func ExampleAuthenticatorForTests(validToken string, userID int, username string) Authenticator {
	return wsrelay.ExampleAuthenticatorForTests(validToken, userID, username)
}

func ExampleResolveUserIDForTests(mapping map[string]int) func(string) (int, error) {
	return wsrelay.ExampleResolveUserIDForTests(mapping)
}
