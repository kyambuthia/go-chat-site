package wsrelay

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kyambuthia/go-chat-site/server/internal/config"
	coremsg "github.com/kyambuthia/go-chat-site/server/internal/core/messaging"
)

// Authenticator validates a bearer token and returns (userID, username, error).
type Authenticator func(token string) (userID int, username string, err error)

type Message = coremsg.Message

type client struct {
	userID          int
	username        string
	conn            *websocket.Conn
	send            chan Message
	hub             *Hub
	resolveToUserID func(string) (int, error)
}

type Hub struct {
	mu      sync.RWMutex
	clients map[int]*client
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewHub() *Hub {
	ctx, cancel := context.WithCancel(context.Background())
	return &Hub{
		clients: make(map[int]*client),
		ctx:     ctx,
		cancel:  cancel,
	}
}

var _ coremsg.Transport = (*Hub)(nil)

func (h *Hub) Run() {
	<-h.ctx.Done()
}

func (h *Hub) Shutdown() {
	h.cancel()
	h.mu.Lock()
	for _, c := range h.clients {
		_ = c.conn.Close()
		close(c.send)
	}
	h.clients = map[int]*client{}
	h.mu.Unlock()
}

func (h *Hub) AddClient(c *client) error {
	if c == nil {
		return errors.New("nil client")
	}
	h.mu.Lock()
	if existing, ok := h.clients[c.userID]; ok {
		_ = existing.conn.Close()
		close(existing.send)
	}
	h.clients[c.userID] = c
	h.mu.Unlock()

	go h.broadcastExcept(c.userID, Message{Type: coremsg.KindUserOnline, From: c.username})
	return nil
}

func (h *Hub) RemoveClient(userID int) {
	h.mu.Lock()
	client, ok := h.clients[userID]
	if ok {
		close(client.send)
		delete(h.clients, userID)
	}
	h.mu.Unlock()

	if ok {
		go h.broadcastExcept(userID, Message{Type: coremsg.KindUserOffline, From: client.username})
	}
}

func (h *Hub) SendDirect(toUserID int, msg Message) bool {
	h.mu.RLock()
	c, ok := h.clients[toUserID]
	h.mu.RUnlock()
	if !ok {
		return false
	}
	select {
	case c.send <- msg:
		return true
	case <-time.After(500 * time.Millisecond):
		return false
	}
}

func (h *Hub) broadcastExcept(excludedUserID int, msg Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for userID, c := range h.clients {
		if userID == excludedUserID {
			continue
		}
		select {
		case c.send <- msg:
		default:
		}
	}
}

func (c *client) readLoop() {
	defer c.hub.RemoveClient(c.userID)

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
			c.trySend(Message{Type: coremsg.KindError, Body: "User not found: " + msg.To})
			continue
		}

		forwardMsg := Message{Type: coremsg.KindDirectMessage, From: c.username, Body: msg.Body}
		if !c.hub.SendDirect(recipientID, forwardMsg) {
			c.trySend(Message{Type: coremsg.KindError, Body: "User is not online: " + msg.To})
			continue
		}

		c.trySend(Message{Type: coremsg.KindMessageAck, ID: msg.ID})
	}
}

func (c *client) trySend(msg Message) {
	select {
	case c.send <- msg:
	default:
	}
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

		userID, username, err := authenticator(token)
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
			conn:            conn,
			send:            make(chan Message, 16),
			hub:             h,
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
	return func(token string) (int, string, error) {
		if token != validToken {
			return 0, "", errors.New("invalid token")
		}
		return userID, username, nil
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
