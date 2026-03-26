package wsrelay

import (
	"context"
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

func assertNoMessage(t *testing.T, conn *websocket.Conn, timeout time.Duration) {
	t.Helper()
	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	var msg Message
	err := conn.ReadJSON(&msg)
	if err == nil {
		t.Fatalf("expected no message, received %+v", msg)
	}
	if nerr, ok := err.(interface{ Timeout() bool }); !ok || !nerr.Timeout() {
		t.Fatalf("expected timeout while waiting for no message, got %v", err)
	}
}

type asyncReadResult struct {
	msg Message
	err error
}

func startAsyncReader(conn *websocket.Conn) <-chan asyncReadResult {
	results := make(chan asyncReadResult, 16)
	go func() {
		defer close(results)
		for {
			var msg Message
			if err := conn.ReadJSON(&msg); err != nil {
				results <- asyncReadResult{err: err}
				return
			}
			results <- asyncReadResult{msg: msg}
		}
	}()
	return results
}

func readUntilTypeFromChannel(t *testing.T, results <-chan asyncReadResult, want coremsg.MessageKind, timeout time.Duration) Message {
	t.Helper()
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case result, ok := <-results:
			if !ok {
				t.Fatalf("connection closed while waiting for %q", want)
			}
			if result.err != nil {
				t.Fatalf("read json while waiting for %q: %v", want, result.err)
			}
			if result.msg.Type == want {
				return result.msg
			}
		case <-timer.C:
			t.Fatalf("timed out waiting for %q", want)
		}
	}
}

func assertNoChannelMessage(t *testing.T, results <-chan asyncReadResult, timeout time.Duration) {
	t.Helper()
	select {
	case result, ok := <-results:
		if !ok {
			t.Fatal("connection closed unexpectedly while waiting for no message")
		}
		if result.err != nil {
			t.Fatalf("unexpected read error while waiting for no message: %v", result.err)
		}
		t.Fatalf("expected no message, received %+v", result.msg)
	case <-time.After(timeout):
	}
}

func waitForCondition(t *testing.T, timeout time.Duration, condition func() bool, description string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", description)
}

type stubDeliveryService struct {
	transport       coremsg.Transport
	storedMessageID int64
}

func (s *stubDeliveryService) SendDirect(ctx context.Context, req coremsg.DirectSendRequest) (coremsg.DeliveryReceipt, error) {
	_ = ctx
	delivered := s.transport.SendDirect(req.ToUserID, Message{
		Type:              coremsg.KindDirectMessage,
		ID:                s.storedMessageID,
		From:              req.From,
		Body:              req.Body,
		Ciphertext:        req.Ciphertext,
		EnvelopeVersion:   req.EnvelopeVersion,
		SenderDeviceID:    req.SenderDeviceID,
		RecipientDeviceID: req.RecipientDeviceID,
	})
	return coremsg.DeliveryReceipt{
		MessageID:       req.MessageID,
		StoredMessageID: s.storedMessageID,
		Delivered:       delivered,
	}, nil
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

		h.RemoveClient(c)
		<-done
	}
}

func TestHubAddClient_AllowsMultipleSessionsForSameUser(t *testing.T) {
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
	if closed {
		t.Fatal("expected first client to remain open")
	}

	if got := len(h.clients[7]); got != 2 {
		t.Fatalf("session count = %d, want 2", got)
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
	hub.SetDeliveryService(&stubDeliveryService{transport: hub, storedMessageID: 501})
	go hub.Run()
	defer hub.Shutdown()

	authenticator := func(token string) (int, string, int64, error) {
		switch token {
		case "alice-token":
			return 1, "alice", 101, nil
		case "bob-token":
			return 2, "bob", 202, nil
		default:
			return 0, "", 0, errors.New("invalid token")
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

	presence := readUntilType(t, bobConn, coremsg.KindPresenceState, 2*time.Second)
	if len(presence.Users) != 1 || presence.Users[0] != "alice" {
		t.Fatalf("presence users = %#v, want [alice]", presence.Users)
	}

	_ = readUntilType(t, aliceConn, coremsg.KindUserOnline, 2*time.Second)

	msg := Message{
		ID:                99,
		Type:              coremsg.KindDirectMessage,
		To:                "bob",
		Body:              "hello",
		Ciphertext:        "opaque-ciphertext",
		EnvelopeVersion:   "x3dh-dr-v1",
		SenderDeviceID:    7,
		RecipientDeviceID: 9,
	}
	if err := aliceConn.WriteJSON(msg); err != nil {
		t.Fatalf("alice write json: %v", err)
	}

	delivered := readUntilType(t, bobConn, coremsg.KindDirectMessage, 2*time.Second)
	if delivered.ID != 501 {
		t.Fatalf("delivered.ID = %d, want 501", delivered.ID)
	}
	if delivered.From != "alice" {
		t.Fatalf("delivered.From = %q, want alice", delivered.From)
	}
	if delivered.Body != "hello" {
		t.Fatalf("delivered.Body = %q, want hello", delivered.Body)
	}
	if delivered.Ciphertext != "opaque-ciphertext" {
		t.Fatalf("delivered.Ciphertext = %q, want opaque-ciphertext", delivered.Ciphertext)
	}
	if delivered.EnvelopeVersion != "x3dh-dr-v1" {
		t.Fatalf("delivered.EnvelopeVersion = %q, want x3dh-dr-v1", delivered.EnvelopeVersion)
	}
	if delivered.SenderDeviceID != 7 {
		t.Fatalf("delivered.SenderDeviceID = %d, want 7", delivered.SenderDeviceID)
	}
	if delivered.RecipientDeviceID != 9 {
		t.Fatalf("delivered.RecipientDeviceID = %d, want 9", delivered.RecipientDeviceID)
	}

	ack := readUntilType(t, aliceConn, coremsg.KindMessageAck, 2*time.Second)
	if ack.ID != 99 {
		t.Fatalf("ack.ID = %d, want 99", ack.ID)
	}
	if ack.StoredMessageID != 501 {
		t.Fatalf("ack.StoredMessageID = %d, want 501", ack.StoredMessageID)
	}
}

func TestWebSocketHandler_OfflineRecipientErrorIncludesClientMessageIDAndRecipient(t *testing.T) {
	hub := NewHub()
	hub.SetDeliveryService(&stubDeliveryService{transport: hub, storedMessageID: 502})
	go hub.Run()
	defer hub.Shutdown()

	authenticator := ExampleAuthenticatorForTests("alice-token", 1, "alice")
	resolve := ExampleResolveUserIDForTests(map[string]int{"alice": 1, "bob": 2})

	s := mustStartWSServer(t, WebSocketHandler(hub, authenticator, resolve))
	defer s.Close()

	aliceHeader := http.Header{}
	aliceHeader.Add("Authorization", "Bearer alice-token")
	aliceConn, _ := dialWS(t, s.URL, aliceHeader)
	defer aliceConn.Close()

	msg := Message{ID: 88, Type: coremsg.KindDirectMessage, To: "bob", Body: "hello"}
	if err := aliceConn.WriteJSON(msg); err != nil {
		t.Fatalf("alice write json: %v", err)
	}

	errMsg := readUntilType(t, aliceConn, coremsg.KindError, 2*time.Second)
	if errMsg.ID != 88 {
		t.Fatalf("error id = %d, want 88", errMsg.ID)
	}
	if errMsg.To != "bob" {
		t.Fatalf("error to = %q, want bob", errMsg.To)
	}
	if errMsg.Body != "User is not online: bob" {
		t.Fatalf("error body = %q, want offline error", errMsg.Body)
	}
	if errMsg.StoredMessageID != 502 {
		t.Fatalf("error stored_message_id = %d, want 502", errMsg.StoredMessageID)
	}
}

func TestWebSocketHandler_PresenceOfflineBroadcast(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Shutdown()

	authenticator := func(token string) (int, string, int64, error) {
		switch token {
		case "alice-token":
			return 1, "alice", 101, nil
		case "bob-token":
			return 2, "bob", 202, nil
		default:
			return 0, "", 0, errors.New("invalid token")
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

	presence := readUntilType(t, bobConn, coremsg.KindPresenceState, 2*time.Second)
	if len(presence.Users) != 1 || presence.Users[0] != "alice" {
		t.Fatalf("presence users = %#v, want [alice]", presence.Users)
	}

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

func TestWebSocketHandler_PresenceBootstrapListsCurrentlyOnlineUsers(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Shutdown()

	authenticator := func(token string) (int, string, int64, error) {
		switch token {
		case "alice-token":
			return 1, "alice", 101, nil
		case "bob-token":
			return 2, "bob", 202, nil
		case "carol-token":
			return 3, "carol", 303, nil
		default:
			return 0, "", 0, errors.New("invalid token")
		}
	}
	resolve := ExampleResolveUserIDForTests(map[string]int{"alice": 1, "bob": 2, "carol": 3})

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

	_ = readUntilType(t, bobConn, coremsg.KindPresenceState, 2*time.Second)
	_ = readUntilType(t, aliceConn, coremsg.KindUserOnline, 2*time.Second)

	carolHeader := http.Header{}
	carolHeader.Add("Authorization", "Bearer carol-token")
	carolConn, _ := dialWS(t, s.URL, carolHeader)
	defer carolConn.Close()

	presence := readUntilType(t, carolConn, coremsg.KindPresenceState, 2*time.Second)
	if len(presence.Users) != 2 || presence.Users[0] != "alice" || presence.Users[1] != "bob" {
		t.Fatalf("presence users = %#v, want [alice bob]", presence.Users)
	}
}

func TestWebSocketHandler_MultipleSessionsOnlyBroadcastOfflineAfterLastDisconnect(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Shutdown()

	authenticator := func(token string) (int, string, int64, error) {
		switch token {
		case "alice-token":
			return 1, "alice", 101, nil
		case "bob-token":
			return 2, "bob", 202, nil
		default:
			return 0, "", 0, errors.New("invalid token")
		}
	}
	resolve := ExampleResolveUserIDForTests(map[string]int{"alice": 1, "bob": 2})

	s := mustStartWSServer(t, WebSocketHandler(hub, authenticator, resolve))
	defer s.Close()

	aliceHeader := http.Header{}
	aliceHeader.Add("Authorization", "Bearer alice-token")
	aliceConn, _ := dialWS(t, s.URL, aliceHeader)
	defer aliceConn.Close()
	aliceResults := startAsyncReader(aliceConn)

	bobHeader := http.Header{}
	bobHeader.Add("Authorization", "Bearer bob-token")
	firstConn, _ := dialWS(t, s.URL, bobHeader)
	defer firstConn.Close()

	_ = readUntilType(t, firstConn, coremsg.KindPresenceState, 2*time.Second)
	online := readUntilTypeFromChannel(t, aliceResults, coremsg.KindUserOnline, 2*time.Second)
	if online.From != "bob" {
		t.Fatalf("online.From = %q, want bob", online.From)
	}

	secondConn, _ := dialWS(t, s.URL, bobHeader)
	defer secondConn.Close()

	presence := readUntilType(t, secondConn, coremsg.KindPresenceState, 2*time.Second)
	if len(presence.Users) != 1 || presence.Users[0] != "alice" {
		t.Fatalf("presence users = %#v, want [alice]", presence.Users)
	}
	assertNoChannelMessage(t, aliceResults, 300*time.Millisecond)

	if err := firstConn.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}
	waitForCondition(t, 2*time.Second, func() bool {
		hub.mu.RLock()
		defer hub.mu.RUnlock()
		return len(hub.clients[2]) == 1
	}, "first bob session to be removed")
	assertNoChannelMessage(t, aliceResults, 300*time.Millisecond)

	if err := secondConn.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}

	offline := readUntilTypeFromChannel(t, aliceResults, coremsg.KindUserOffline, 2*time.Second)
	if offline.From != "bob" {
		t.Fatalf("offline.From = %q, want bob", offline.From)
	}
}

func TestWebSocketHandler_DirectMessageFansOutToAllRecipientSessions(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Shutdown()

	authenticator := func(token string) (int, string, int64, error) {
		switch token {
		case "alice-token":
			return 1, "alice", 101, nil
		case "bob-token":
			return 2, "bob", 202, nil
		default:
			return 0, "", 0, errors.New("invalid token")
		}
	}
	resolve := ExampleResolveUserIDForTests(map[string]int{"alice": 1, "bob": 2})

	s := mustStartWSServer(t, WebSocketHandler(hub, authenticator, resolve))
	defer s.Close()

	aliceHeader := http.Header{}
	aliceHeader.Add("Authorization", "Bearer alice-token")
	aliceConnOne, _ := dialWS(t, s.URL, aliceHeader)
	defer aliceConnOne.Close()
	aliceConnTwo, _ := dialWS(t, s.URL, aliceHeader)
	defer aliceConnTwo.Close()

	_ = readUntilType(t, aliceConnOne, coremsg.KindPresenceState, 2*time.Second)
	_ = readUntilType(t, aliceConnTwo, coremsg.KindPresenceState, 2*time.Second)

	bobHeader := http.Header{}
	bobHeader.Add("Authorization", "Bearer bob-token")
	bobConn, _ := dialWS(t, s.URL, bobHeader)
	defer bobConn.Close()

	_ = readUntilType(t, bobConn, coremsg.KindPresenceState, 2*time.Second)
	_ = readUntilType(t, aliceConnOne, coremsg.KindUserOnline, 2*time.Second)
	_ = readUntilType(t, aliceConnTwo, coremsg.KindUserOnline, 2*time.Second)

	msg := Message{ID: 144, Type: coremsg.KindDirectMessage, To: "alice", Body: "hello both tabs"}
	if err := bobConn.WriteJSON(msg); err != nil {
		t.Fatalf("bob write json: %v", err)
	}

	firstDelivered := readUntilType(t, aliceConnOne, coremsg.KindDirectMessage, 2*time.Second)
	secondDelivered := readUntilType(t, aliceConnTwo, coremsg.KindDirectMessage, 2*time.Second)
	if firstDelivered.Body != "hello both tabs" || secondDelivered.Body != "hello both tabs" {
		t.Fatalf("unexpected delivered bodies: one=%q two=%q", firstDelivered.Body, secondDelivered.Body)
	}

	ack := readUntilType(t, bobConn, coremsg.KindMessageAck, 2*time.Second)
	if ack.ID != 144 {
		t.Fatalf("ack.ID = %d, want 144", ack.ID)
	}
}
