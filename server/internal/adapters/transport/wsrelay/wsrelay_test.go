package wsrelay

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	coremsg "github.com/kyambuthia/go-chat-site/server/internal/core/messaging"
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

func dialWS(t *testing.T, serverURL string, header http.Header) (*websocket.Conn, *http.Response) {
	t.Helper()
	u := "ws" + strings.TrimPrefix(serverURL, "http")
	conn, resp, err := websocket.DefaultDialer.Dial(u, header)
	if err != nil {
		t.Fatalf("websocket dial failed: %v", err)
	}
	return conn, resp
}

func readUntilType(t *testing.T, conn *websocket.Conn, want coremsg.MessageKind, timeout time.Duration) Message {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		if err := conn.SetReadDeadline(deadline); err != nil {
			t.Fatalf("set read deadline: %v", err)
		}
		var msg Message
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("read json while waiting for %q: %v", want, err)
		}
		if msg.Type == want {
			return msg
		}
	}
}

func TestClientSendWithTimeout_AfterCloseReturnsFalse(t *testing.T) {
	c := &client{send: make(chan Message, 1)}
	c.close()

	if ok := c.sendWithTimeout(Message{Type: coremsg.KindDirectMessage}, 0); ok {
		t.Fatal("expected send to fail after close")
	}
}

func TestHubSendDirect_ConcurrentRemoveDoesNotPanic(t *testing.T) {
	h := NewHub()
	defer h.Shutdown()

	for i := 0; i < 500; i++ {
		c := &client{userID: 1, username: "alice", send: make(chan Message, 1), hub: h}
		if err := h.AddClient(c); err != nil {
			t.Fatalf("add client: %v", err)
		}

		done := make(chan struct{})
		go func() {
			defer close(done)
			_ = h.SendDirect(1, Message{Type: coremsg.KindDirectMessage, Body: "hello"})
		}()

		h.RemoveClient(1)
		<-done
	}
}

func TestHubAddClient_ReplacesAndClosesExistingClient(t *testing.T) {
	h := NewHub()
	defer h.Shutdown()

	first := &client{userID: 7, username: "alice", send: make(chan Message, 1), hub: h}
	if err := h.AddClient(first); err != nil {
		t.Fatalf("add first client: %v", err)
	}

	second := &client{userID: 7, username: "alice", send: make(chan Message, 1), hub: h}
	if err := h.AddClient(second); err != nil {
		t.Fatalf("add second client: %v", err)
	}

	first.sendMu.RLock()
	closed := first.closed
	first.sendMu.RUnlock()
	if !closed {
		t.Fatal("expected first client to be closed when replaced")
	}
}

func TestWebSocketHandler_SubprotocolTokenAuth_Succeeds(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Shutdown()

	authenticator := ExampleAuthenticatorForTests("subproto-token", 1, "alice")
	resolve := ExampleResolveUserIDForTests(map[string]int{"alice": 1})

	s := mustStartWSServer(t, WebSocketHandler(hub, authenticator, resolve))
	defer s.Close()

	u := "ws" + strings.TrimPrefix(s.URL, "http")
	dialer := websocket.Dialer{Subprotocols: []string{"bearer.subproto-token"}}
	conn, _, err := dialer.Dial(u, nil)
	if err != nil {
		t.Fatalf("dial with subprotocol token failed: %v", err)
	}
	defer conn.Close()
}

func TestWebSocketHandler_MissingToken_ReturnsUnauthorized(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Shutdown()

	authenticator := ExampleAuthenticatorForTests("valid", 1, "alice")
	resolve := ExampleResolveUserIDForTests(map[string]int{"alice": 1})

	s := mustStartWSServer(t, WebSocketHandler(hub, authenticator, resolve))
	defer s.Close()

	u := "ws" + strings.TrimPrefix(s.URL, "http")
	_, resp, err := websocket.DefaultDialer.Dial(u, nil)
	if err == nil {
		t.Fatal("expected dial error for missing token")
	}
	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		if resp == nil {
			t.Fatal("expected HTTP 401 response")
		}
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestWebSocketHandler_DirectMessageDeliveryAndAck(t *testing.T) {
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

	aliceHeader := http.Header{}
	aliceHeader.Add("Authorization", "Bearer alice-token")
	aliceConn, _ := dialWS(t, s.URL, aliceHeader)
	defer aliceConn.Close()

	bobHeader := http.Header{}
	bobHeader.Add("Authorization", "Bearer bob-token")
	bobConn, _ := dialWS(t, s.URL, bobHeader)
	defer bobConn.Close()

	_ = readUntilType(t, aliceConn, coremsg.KindUserOnline, 2*time.Second)

	msg := Message{ID: 99, Type: coremsg.KindDirectMessage, To: "bob", Body: "hello"}
	if err := aliceConn.WriteJSON(msg); err != nil {
		t.Fatalf("alice write json: %v", err)
	}

	delivered := readUntilType(t, bobConn, coremsg.KindDirectMessage, 2*time.Second)
	if delivered.From != "alice" {
		t.Fatalf("delivered.From = %q, want alice", delivered.From)
	}
	if delivered.Body != "hello" {
		t.Fatalf("delivered.Body = %q, want hello", delivered.Body)
	}

	ack := readUntilType(t, aliceConn, coremsg.KindMessageAck, 2*time.Second)
	if ack.ID != 99 {
		t.Fatalf("ack.ID = %d, want 99", ack.ID)
	}
}

func TestWebSocketHandler_PresenceOfflineBroadcast(t *testing.T) {
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

	aliceHeader := http.Header{}
	aliceHeader.Add("Authorization", "Bearer alice-token")
	aliceConn, _ := dialWS(t, s.URL, aliceHeader)
	defer aliceConn.Close()

	bobHeader := http.Header{}
	bobHeader.Add("Authorization", "Bearer bob-token")
	bobConn, _ := dialWS(t, s.URL, bobHeader)

	online := readUntilType(t, aliceConn, coremsg.KindUserOnline, 2*time.Second)
	if online.From != "bob" {
		t.Fatalf("online.From = %q, want bob", online.From)
	}

	if err := bobConn.Close(); err != nil {
		t.Fatalf("bob close: %v", err)
	}

	offline := readUntilType(t, aliceConn, coremsg.KindUserOffline, 2*time.Second)
	if offline.From != "bob" {
		t.Fatalf("offline.From = %q, want bob", offline.From)
	}
}

func TestWebSocketHandler_ReplacingSameUserClosesOldConnection(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Shutdown()

	authenticator := ExampleAuthenticatorForTests("same-user-token", 42, "alice")
	resolve := ExampleResolveUserIDForTests(map[string]int{"alice": 42})

	s := mustStartWSServer(t, WebSocketHandler(hub, authenticator, resolve))
	defer s.Close()

	header := http.Header{}
	header.Add("Authorization", "Bearer same-user-token")
	firstConn, _ := dialWS(t, s.URL, header)
	defer firstConn.Close()

	secondConn, _ := dialWS(t, s.URL, header)
	defer secondConn.Close()

	if err := firstConn.SetReadDeadline(time.Now().Add(1500 * time.Millisecond)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}

	var msg Message
	err := firstConn.ReadJSON(&msg)
	if err == nil {
		t.Fatalf("expected old connection to close, received message %+v", msg)
	}
	if nerr, ok := err.(interface{ Timeout() bool }); ok && nerr.Timeout() {
		t.Fatalf("expected old connection to be closed, got timeout instead: %v", err)
	}
}
