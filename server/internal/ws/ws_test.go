package ws

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

func TestWebSocketHandler_ValidToken(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Shutdown()

	authenticator := ExampleAuthenticatorForTests("valid-token", 1, "testuser")
	resolve := ExampleResolveUserIDForTests(map[string]int{"testuser": 1})

	s := httptest.NewServer(WebSocketHandler(hub, authenticator, resolve))
	defer s.Close()

	u := "ws" + strings.TrimPrefix(s.URL, "http")

	header := http.Header{}
	header.Add("Authorization", "Bearer valid-token")

	conn, _, err := websocket.DefaultDialer.Dial(u, header)
	if err != nil {
		t.Fatalf("could not connect to websocket: %v", err)
	}
	defer conn.Close()

	// Test sending a message
	msg := Message{Type: "direct_message", To: "testuser", Body: "hello"}
	if err := conn.WriteJSON(msg); err != nil {
		t.Fatalf("could not write json: %v", err)
	}

	// Test receiving a message
	var receivedMsg Message
	if err := conn.ReadJSON(&receivedMsg); err != nil {
		t.Fatalf("could not read json: %v", err)
	}

	if receivedMsg.Type != "direct_message" {
		t.Errorf("expected direct_message message, got %s", receivedMsg.Type)
	}

	if err := conn.ReadJSON(&receivedMsg); err != nil {
		t.Fatalf("could not read json: %v", err)
	}

	if receivedMsg.Type != "message_ack" {
		t.Errorf("expected message_ack message, got %s", receivedMsg.Type)
	}
}

func TestWebSocketHandler_InvalidToken(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Shutdown()

	authenticator := ExampleAuthenticatorForTests("valid-token", 1, "testuser")
	resolve := ExampleResolveUserIDForTests(map[string]int{"testuser": 1})

	s := httptest.NewServer(WebSocketHandler(hub, authenticator, resolve))
	defer s.Close()

	u := "ws" + strings.TrimPrefix(s.URL, "http")

	header := http.Header{}
	header.Add("Authorization", "Bearer invalid-token")

	_, _, err := websocket.DefaultDialer.Dial(u, header)
	if err == nil {
		t.Fatal("expected error connecting with invalid token")
	}
}
