// Fixed syntax error and added robust error handling.

package ws

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Authenticator validates a bearer token and returns (userID, username, error).
// Keep this abstract so tests can inject a fake authenticator.
type Authenticator func(token string) (userID int, username string, err error)

// Message represents the wire format for messages sent/received over the WS.
type Message struct {
	ID   int64  `json:"id,omitempty"`
	Type string `json:"type"` // e.g. "direct_message"
	From string `json:"from,omitempty"`
	To   string `json:"to,omitempty"`
	Body string `json:"body,omitempty"`
}

// client represents a connected WebSocket client.
type client struct {
	userID          int
	username        string
	conn            *websocket.Conn
	send            chan Message
	hub             *Hub
	resolveToUserID func(string) (int, error)
}

// Hub keeps track of connected clients.
type Hub struct {
	mu      sync.RWMutex
	clients map[int]*client // map userID -> client
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewHub creates a new Hub instance.
func NewHub() *Hub {
	ctx, cancel := context.WithCancel(context.Background())
	return &Hub{
		clients: make(map[int]*client),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Run starts background goroutines if needed. Currently a placeholder for future tasks.
func (h *Hub) Run() {
	// currently no global ticker, but this is where cleanup timers could live
	go func() {
		<-h.ctx.Done()
		// perform any shutdown tasks here
	}()
}

// Shutdown stops the hub and disconnects clients.
func (h *Hub) Shutdown() {
	h.cancel()
	// close all client connections
	h.mu.Lock()
	for _, c := range h.clients {
		c.conn.Close()
		close(c.send)
	}
	h.clients = map[int]*client{}
	h.mu.Unlock()
}

// AddClient registers a connected client in the hub.
func (h *Hub) AddClient(c *client) error {
	if c == nil {
		return errors.New("nil client")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	// if another client exists for this user, close it (single-session policy)
	if existing, ok := h.clients[c.userID]; ok {
		existing.conn.Close()
		close(existing.send)
	}
	h.clients[c.userID] = c
	return nil
}

// RemoveClient removes a client from the hub.
func (h *Hub) RemoveClient(userID int) {
	h.mu.Lock()
	if c, ok := h.clients[userID]; ok {
		c.conn.Close()
		close(c.send)
		delete(h.clients, userID)
	}
	h.mu.Unlock()
}

// SendDirect sends a message to a specific user if connected.
// Returns true if delivered to an active connection, false otherwise.
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
		// send timed out
		return false
	}
}

// Broadcast sends a message to all connected clients. Use sparingly.
func (h *Hub) Broadcast(msg Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, c := range h.clients {
		select {
		case c.send <- msg:
		default:
			// if send buffer is full, drop message for that client
		}
	}
}

// ---------------- client read/write loops ----------------

func (c *client) readLoop() {
	log.Println("Starting read loop for client", c.userID)
	defer func() {
		log.Println("Exiting read loop for client", c.userID)
		c.hub.RemoveClient(c.userID)
	}()
	c.conn.SetReadLimit(1024)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	for {
		var msg Message
		if err := c.conn.ReadJSON(&msg); err != nil {
			// client disconnected or sent invalid data
			return
		}

		if msg.Type == "direct_message" {
			if c.resolveToUserID == nil || msg.To == "" {
				continue // or send error
			}

			recipientID, err := c.resolveToUserID(msg.To)
			if err != nil {
				c.send <- Message{Type: "error", Body: "User not found: " + msg.To}
				continue
			}

			// The message to be forwarded
			forwardMsg := Message{Type: "direct_message", From: c.username, Body: msg.Body}

			if !c.hub.SendDirect(recipientID, forwardMsg) {
				c.send <- Message{Type: "error", Body: "User is not online: " + msg.To}
			} else {
				// Optional: send an ack to the original sender
				c.send <- Message{Type: "message_ack", ID: msg.ID}
			}
		}
	}
}
func (c *client) writeLoop() {
	log.Println("Starting write loop for client", c.userID)
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		log.Println("Exiting write loop for client", c.userID)
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteJSON(msg); err != nil {
				log.Println("writeLoop: WriteJSON error:", err)
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Println("writeLoop: Ping error:", err)
				return
			}
		case <-c.hub.ctx.Done():
			return
		}
	}
}

// ---------------- HTTP Handler / upgrader ----------------

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// TODO: restrict origins in prod
		return true
	},
}

// WebSocketHandler returns an http.HandlerFunc that upgrades a connection and registers the client.
// authenticator: function that accepts a bearer token and returns userID, username, error.
// resolveToUserID: optional function to resolve username -> userID for direct sends. If nil, hub won't resolve.
func WebSocketHandler(h *Hub, authenticator Authenticator, resolveToUserID func(username string) (int, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				log.Println("Recovered from panic in WebSocketHandler:", r)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()

		log.Println("WebSocket handler called")
		// First, try to get the token from the Authorization header
		token := ""
		authHeader := r.Header.Get("Authorization")
		const prefix = "Bearer "
		if strings.HasPrefix(strings.TrimSpace(authHeader), prefix) {
			token = strings.TrimSpace(strings.TrimPrefix(authHeader, prefix))
		}

		// If not in header, try query parameter (for browser clients)
		if token == "" {
			token = r.URL.Query().Get("token")
		}

		if token == "" {
			log.Println("Token not found")
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}

		log.Println("Authenticating token")
		userID, username, err := authenticator(token)
		if err != nil {
			log.Println("Authentication failed:", err)
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		log.Println("Authentication successful, upgrading connection")
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Upgrade failed:", err)
			http.Error(w, "failed to upgrade connection", http.StatusInternalServerError)
			return
		}

		log.Println("Client added, launching read/write loops")
		c := &client{
			userID:          userID,
			username:        username,
			conn:            conn,
			send:            make(chan Message, 16),
			hub:             h,
			resolveToUserID: resolveToUserID,
		}
		if err := h.AddClient(c); err != nil {
			log.Println("WebSocketHandler: AddClient error:", err)
			http.Error(w, "failed to register client", http.StatusInternalServerError)
			conn.Close()
			return
		}
		go c.writeLoop()
		go c.readLoop()

	}
}

// ---------------- Utilities / Examples ----------------

// Example authenticator wrapper (NOT for production) — included for tests and examples.
// In real code, replace with your auth.ValidateToken implementation.
func ExampleAuthenticatorForTests(validToken string, userID int, username string) Authenticator {
	return func(token string) (int, string, error) {
		if token != validToken {
			return 0, "", errors.New("invalid token")
		}
		return userID, username, nil
	}
}

// Resolve helper example for tests
func ExampleResolveUserIDForTests(mapping map[string]int) func(string) (int, error) {
	return func(username string) (int, error) {
		if id, ok := mapping[username]; ok {
			return id, nil
		}
		return 0, errors.New("unknown user")
	}
}

// NOTE:
// - This is a skeleton implementation. It intentionally keeps the web/DB boundaries thin:
//   the Hub does not know how to resolve usernames to user IDs (resolveToUserID optional param),
//   so your handler or higher-level service should provide that.
// - Tests should inject ExampleAuthenticatorForTests and ExampleResolveUserIDForTests to avoid
//   depending on the real auth/database layers.
// - Add logging, metrics, per-user rate limiting, and proper origin checks before production use.