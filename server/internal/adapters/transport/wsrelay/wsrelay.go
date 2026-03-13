package wsrelay

import (
	"context"
	"errors"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kyambuthia/go-chat-site/server/internal/config"
	coremsg "github.com/kyambuthia/go-chat-site/server/internal/core/messaging"
)

// Authenticator validates a bearer token and returns (userID, username, sessionID, error).
type Authenticator func(token string) (userID int, username string, sessionID int64, err error)

type Message = coremsg.Message

type client struct {
	userID          int
	username        string
	sessionID       int64
	conn            *websocket.Conn
	send            chan Message
	sendMu          sync.RWMutex
	closed          bool
	hub             *Hub
	messaging       coremsg.Service
	resolveToUserID func(string) (int, error)
}

type Hub struct {
	mu              sync.RWMutex
	clients         map[int]map[*client]struct{}
	ctx             context.Context
	cancel          context.CancelFunc
	deliveryService coremsg.Service
}

func NewHub() *Hub {
	ctx, cancel := context.WithCancel(context.Background())
	h := &Hub{
		clients: make(map[int]map[*client]struct{}),
		ctx:     ctx,
		cancel:  cancel,
	}
	h.deliveryService = coremsg.NewRelayService(h)
	return h
}

var _ coremsg.Transport = (*Hub)(nil)

func (h *Hub) SetDeliveryService(svc coremsg.Service) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if svc == nil {
		h.deliveryService = coremsg.NewRelayService(h)
		return
	}
	h.deliveryService = svc
}

func (h *Hub) delivery() coremsg.Service {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.deliveryService == nil {
		return coremsg.NewRelayService(h)
	}
	return h.deliveryService
}

func (h *Hub) Run() {
	<-h.ctx.Done()
}

func (h *Hub) Shutdown() {
	h.cancel()
	h.mu.Lock()
	for _, userClients := range h.clients {
		for c := range userClients {
			c.close()
		}
	}
	h.clients = map[int]map[*client]struct{}{}
	h.mu.Unlock()
}

func (h *Hub) AddClient(c *client) error {
	if c == nil {
		return errors.New("nil client")
	}

	h.mu.Lock()
	userClients, ok := h.clients[c.userID]
	if !ok {
		userClients = make(map[*client]struct{})
		h.clients[c.userID] = userClients
	}
	isFirstSession := len(userClients) == 0
	userClients[c] = struct{}{}
	onlineUsers := h.onlineUsernamesExceptLocked(c.userID)
	h.mu.Unlock()

	c.trySend(Message{Type: coremsg.KindPresenceState, Users: onlineUsers})
	if isFirstSession {
		go h.broadcastExcept(c.userID, Message{Type: coremsg.KindUserOnline, From: c.username})
	}
	return nil
}

func (h *Hub) RemoveClient(c *client) {
	if c == nil {
		return
	}

	h.mu.Lock()
	userClients, ok := h.clients[c.userID]
	if !ok {
		h.mu.Unlock()
		return
	}
	_, removed := userClients[c]
	if removed {
		delete(userClients, c)
	}
	isLastSession := removed && len(userClients) == 0
	if isLastSession {
		delete(h.clients, c.userID)
	}
	h.mu.Unlock()

	if !removed {
		return
	}
	c.close()
	if isLastSession {
		go h.broadcastExcept(c.userID, Message{Type: coremsg.KindUserOffline, From: c.username})
	}
}

func (h *Hub) DisconnectSession(sessionID int64) {
	if sessionID <= 0 {
		return
	}

	h.mu.RLock()
	matches := make([]*client, 0)
	for _, userClients := range h.clients {
		for c := range userClients {
			if c.sessionID == sessionID {
				matches = append(matches, c)
			}
		}
	}
	h.mu.RUnlock()

	for _, c := range matches {
		h.RemoveClient(c)
	}
}

func (h *Hub) SendDirect(toUserID int, msg Message) bool {
	h.mu.RLock()
	userClients, ok := h.clients[toUserID]
	clients := make([]*client, 0, len(userClients))
	if ok {
		for c := range userClients {
			clients = append(clients, c)
		}
	}
	h.mu.RUnlock()
	if !ok {
		return false
	}

	delivered := false
	for _, c := range clients {
		if c.sendWithTimeout(msg, 500*time.Millisecond) {
			delivered = true
		}
	}
	return delivered
}

func (h *Hub) broadcastExcept(excludedUserID int, msg Message) {
	h.mu.RLock()
	clients := make([]*client, 0)
	for userID, userClients := range h.clients {
		if userID == excludedUserID {
			continue
		}
		for c := range userClients {
			clients = append(clients, c)
		}
	}
	h.mu.RUnlock()

	for _, c := range clients {
		_ = c.sendWithTimeout(msg, 0)
	}
}

func (h *Hub) onlineUsernamesExceptLocked(excludedUserID int) []string {
	users := make([]string, 0, len(h.clients))
	for userID, userClients := range h.clients {
		if userID == excludedUserID || len(userClients) == 0 {
			continue
		}
		for c := range userClients {
			users = append(users, c.username)
			break
		}
	}
	sort.Strings(users)
	return users
}

func (c *client) readLoop() {
	defer c.hub.RemoveClient(c)

	c.conn.SetReadLimit(1024)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var msg Message
		if err := c.conn.ReadJSON(&msg); err != nil {
			return
		}

		if msg.Type != coremsg.KindDirectMessage {
			continue
		}
		if c.resolveToUserID == nil || strings.TrimSpace(msg.To) == "" {
			continue
		}

		recipientID, err := c.resolveToUserID(msg.To)
		if err != nil {
			c.trySend(Message{Type: coremsg.KindError, ID: msg.ID, To: msg.To, Body: "User not found: " + msg.To})
			continue
		}

		if c.messaging == nil {
			c.trySend(Message{Type: coremsg.KindError, ID: msg.ID, To: msg.To, Body: "relay unavailable"})
			continue
		}

		receipt, err := c.messaging.SendDirect(c.hub.ctx, coremsg.DirectSendRequest{
			FromUserID: c.userID,
			From:       c.username,
			ToUserID:   recipientID,
			Body:       msg.Body,
			MessageID:  msg.ID,
		})
		if err != nil {
			c.trySend(Message{Type: coremsg.KindError, ID: msg.ID, To: msg.To, Body: "delivery failed"})
			continue
		}
		if !receipt.Delivered {
			c.trySend(Message{
				Type:            coremsg.KindError,
				ID:              msg.ID,
				To:              msg.To,
				Body:            "User is not online: " + msg.To,
				StoredMessageID: receipt.StoredMessageID,
			})
			continue
		}

		c.trySend(Message{Type: coremsg.KindMessageAck, ID: msg.ID, StoredMessageID: receipt.StoredMessageID})
	}
}

func (c *client) trySend(msg Message) {
	_ = c.sendWithTimeout(msg, 0)
}

func (c *client) writeLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteJSON(msg); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-c.hub.ctx.Done():
			return
		}
	}
}

func (c *client) sendWithTimeout(msg Message, timeout time.Duration) bool {
	c.sendMu.RLock()
	defer c.sendMu.RUnlock()

	if c.closed {
		return false
	}

	if timeout <= 0 {
		select {
		case c.send <- msg:
			return true
		default:
			return false
		}
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case c.send <- msg:
		return true
	case <-timer.C:
		return false
	}
}

func (c *client) close() {
	c.sendMu.Lock()
	if c.closed {
		c.sendMu.Unlock()
		return
	}
	c.closed = true
	close(c.send)
	c.sendMu.Unlock()

	if c.conn != nil {
		_ = c.conn.Close()
	}
}

func newUpgrader(checkOrigin func(*http.Request) bool) websocket.Upgrader {
	if checkOrigin == nil {
		checkOrigin = config.WSOriginCheckFunc()
	}
	return websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     checkOrigin,
	}
}

func WebSocketHandler(h *Hub, authenticator Authenticator, resolveToUserID func(username string) (int, error)) http.HandlerFunc {
	upgrader := newUpgrader(nil)
	return func(w http.ResponseWriter, r *http.Request) {
		token := ""
		authHeader := r.Header.Get("Authorization")
		const prefix = "Bearer "
		if strings.HasPrefix(strings.TrimSpace(authHeader), prefix) {
			token = strings.TrimSpace(strings.TrimPrefix(authHeader, prefix))
		}

		// Browser-safe auth transport without query-string token leakage.
		if token == "" {
			for _, proto := range websocket.Subprotocols(r) {
				if strings.HasPrefix(proto, "bearer.") {
					token = strings.TrimPrefix(proto, "bearer.")
					break
				}
			}
		}

		if token == "" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}

		userID, username, sessionID, err := authenticator(token)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("upgrade failed:", err)
			http.Error(w, "failed to upgrade connection", http.StatusInternalServerError)
			return
		}

		c := &client{
			userID:          userID,
			username:        username,
			sessionID:       sessionID,
			conn:            conn,
			send:            make(chan Message, 16),
			hub:             h,
			messaging:       h.delivery(),
			resolveToUserID: resolveToUserID,
		}
		if err := h.AddClient(c); err != nil {
			http.Error(w, "failed to register client", http.StatusInternalServerError)
			_ = conn.Close()
			return
		}

		go c.writeLoop()
		go c.readLoop()
	}
}

func ExampleAuthenticatorForTests(validToken string, userID int, username string) Authenticator {
	return func(token string) (int, string, int64, error) {
		if token != validToken {
			return 0, "", 0, errors.New("invalid token")
		}
		return userID, username, int64(userID), nil
	}
}

func ExampleResolveUserIDForTests(mapping map[string]int) func(string) (int, error) {
	return func(username string) (int, error) {
		if id, ok := mapping[username]; ok {
			return id, nil
		}
		return 0, errors.New("unknown user")
	}
}
