package ws

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func mustStartWSServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Skipf("websocket test skipped: cannot start httptest server in this environment: %v", r)
		}
	}()
	return httptest.NewServer(handler)
}

func TestWebSocketHandler_ValidToken(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Shutdown()

	authenticator := func(token string) (int, string, error) {
		switch token {
		case "alice-token":
			return 1, "alice", nil
		case "bob-token":
			return 2, "bob", nil
		default:
			return 0, "", errors.New("invalid token")
		}
	}
	resolve := ExampleResolveUserIDForTests(map[string]int{"alice": 1, "bob": 2})

	s := mustStartWSServer(t, WebSocketHandler(hub, authenticator, resolve))
	defer s.Close()

	u := "ws" + strings.TrimPrefix(s.URL, "http")

	aliceHeader := http.Header{}
	aliceHeader.Add("Authorization", "Bearer alice-token")
	aliceConn, _, err := websocket.DefaultDialer.Dial(u, aliceHeader)
	if err != nil {
		t.Fatalf("could not connect alice websocket: %v", err)
	}
	defer aliceConn.Close()

	bobHeader := http.Header{}
	bobHeader.Add("Authorization", "Bearer bob-token")
	bobConn, _, err := websocket.DefaultDialer.Dial(u, bobHeader)
	if err != nil {
		t.Fatalf("could not connect bob websocket: %v", err)
	}
	defer bobConn.Close()

	msg := Message{ID: 99, Type: "direct_message", To: "bob", Body: "hello"}
	if err := aliceConn.WriteJSON(msg); err != nil {
		t.Fatalf("could not write json: %v", err)
	}

	var delivered Message
	if err := bobConn.ReadJSON(&delivered); err != nil {
		t.Fatalf("bob could not read json: %v", err)
	}
	if delivered.Type != "direct_message" {
		t.Fatalf("expected direct_message message, got %s", delivered.Type)
	}
	if delivered.From != "alice" {
		t.Fatalf("expected message from alice, got %q", delivered.From)
	}
	if delivered.Body != "hello" {
		t.Fatalf("expected message body hello, got %q", delivered.Body)
	}

	if err := aliceConn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("alice could not set read deadline: %v", err)
	}
	for {
		var inbound Message
		if err := aliceConn.ReadJSON(&inbound); err != nil {
			t.Fatalf("alice could not read ack json: %v", err)
		}
		if inbound.Type != "message_ack" {
			continue
		}
		if inbound.ID != 99 {
			t.Fatalf("expected ack id 99, got %d", inbound.ID)
		}
		break
	}
}

func TestWebSocketHandler_InvalidToken(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Shutdown()

	authenticator := ExampleAuthenticatorForTests("valid-token", 1, "testuser")
	resolve := ExampleResolveUserIDForTests(map[string]int{"testuser": 1})

	s := mustStartWSServer(t, WebSocketHandler(hub, authenticator, resolve))
	defer s.Close()

	u := "ws" + strings.TrimPrefix(s.URL, "http")

	header := http.Header{}
	header.Add("Authorization", "Bearer invalid-token")

	_, _, err := websocket.DefaultDialer.Dial(u, header)
	if err == nil {
		t.Fatal("expected error connecting with invalid token")
	}
}

func TestWebSocketHandler_OfflineRecipientGetsErrorAndNoAck(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Shutdown()

	authenticator := ExampleAuthenticatorForTests("alice-token", 1, "alice")
	resolve := ExampleResolveUserIDForTests(map[string]int{"bob": 2})

	s := mustStartWSServer(t, WebSocketHandler(hub, authenticator, resolve))
	defer s.Close()

	u := "ws" + strings.TrimPrefix(s.URL, "http")
	header := http.Header{}
	header.Add("Authorization", "Bearer alice-token")

	conn, _, err := websocket.DefaultDialer.Dial(u, header)
	if err != nil {
		t.Fatalf("could not connect to websocket: %v", err)
	}
	defer conn.Close()

	if err := conn.WriteJSON(Message{ID: 123, Type: "direct_message", To: "bob", Body: "ping"}); err != nil {
		t.Fatalf("could not write json: %v", err)
	}

	if err := conn.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
		t.Fatalf("could not set read deadline: %v", err)
	}

	var first Message
	if err := conn.ReadJSON(&first); err != nil {
		t.Fatalf("could not read json: %v", err)
	}
	if first.Type != "error" {
		t.Fatalf("expected error message, got %s", first.Type)
	}
	if !strings.Contains(first.Body, "not online") {
		t.Fatalf("expected offline error, got %q", first.Body)
	}

	var second Message
	err = conn.ReadJSON(&second)
	if err == nil {
		t.Fatalf("expected no ack, but received %+v", second)
	}
	if !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		if nerr, ok := err.(interface{ Timeout() bool }); !ok || !nerr.Timeout() {
			t.Fatalf("expected read timeout when no ack is sent, got %v", err)
		}
	}
}
