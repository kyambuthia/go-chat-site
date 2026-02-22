package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kyambuthia/go-chat-site/server/internal/auth"
	"github.com/kyambuthia/go-chat-site/server/internal/migrate"
	"github.com/kyambuthia/go-chat-site/server/internal/store"
	"github.com/kyambuthia/go-chat-site/server/internal/ws"
)

func TestAuthHandlers(t *testing.T) {
	if err := auth.ConfigureJWT("test-secret-123456"); err != nil {
		t.Fatal(err)
	}

	s, err := store.NewSqliteStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}

	err = migrate.RunMigrations(s.DB, "../../migrations")
	if err != nil {
		t.Fatal(err)
	}

	hub := ws.NewHub()
	go hub.Run()
	defer hub.Shutdown()

	a := NewAPI(s, hub)

	t.Run("TestRegister_Succeeds", func(t *testing.T) {
		rr := httptest.NewRecorder()
		reqBody, _ := json.Marshal(map[string]string{
			"username": "testuser",
			"password": "password123",
		})
		req, _ := http.NewRequest("POST", "/api/register", bytes.NewBuffer(reqBody))

		a.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusCreated)
		}

		var resp map[string]interface{}
		_ = json.Unmarshal(rr.Body.Bytes(), &resp)
		if resp["username"] != "testuser" {
			t.Errorf("handler returned unexpected body: got %v want %v", resp["username"], "testuser")
		}
	})

	t.Run("TestRegister_FailsOnDuplicateUsername", func(t *testing.T) {
		rr := httptest.NewRecorder()
		reqBody, _ := json.Marshal(map[string]string{
			"username": "testuser",
			"password": "password123",
		})
		req, _ := http.NewRequest("POST", "/api/register", bytes.NewBuffer(reqBody))

		a.ServeHTTP(rr, req)

		if rr.Code != http.StatusConflict {
			t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusConflict)
		}
	})

	t.Run("TestRegister_FailsOnShortPassword", func(t *testing.T) {
		rr := httptest.NewRecorder()
		reqBody, _ := json.Marshal(map[string]string{
			"username": "newuser",
			"password": "short",
		})
		req, _ := http.NewRequest("POST", "/api/register", bytes.NewBuffer(reqBody))

		a.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("TestLogin_Succeeds", func(t *testing.T) {
		rr := httptest.NewRecorder()
		reqBody, _ := json.Marshal(map[string]string{
			"username": "testuser",
			"password": "password123",
		})
		req, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(reqBody))

		a.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusOK)
		}

		var resp map[string]string
		_ = json.Unmarshal(rr.Body.Bytes(), &resp)
		if resp["token"] == "" {
			t.Error("handler returned no token")
		}
	})

	t.Run("TestLogin_FailsOnWrongPassword", func(t *testing.T) {
		rr := httptest.NewRecorder()
		reqBody, _ := json.Marshal(map[string]string{
			"username": "testuser",
			"password": "wrongpassword",
		})
		req, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(reqBody))

		a.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusUnauthorized)
		}
	})

	t.Run("TestLogin_FailsOnUnknownUser", func(t *testing.T) {
		rr := httptest.NewRecorder()
		reqBody, _ := json.Marshal(map[string]string{
			"username": "unknownuser",
			"password": "password123",
		})
		req, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(reqBody))

		a.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusUnauthorized)
		}
	})
}

func TestContactsAndInvitesRoutes_Compatibility(t *testing.T) {
	if err := auth.ConfigureJWT("test-secret-123456"); err != nil {
		t.Fatal(err)
	}

	s, err := store.NewSqliteStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.DB.Close() })

	if err := migrate.RunMigrations(s.DB, "../../migrations"); err != nil {
		t.Fatal(err)
	}

	aliceID, err := s.CreateUser("alice", "password123")
	if err != nil {
		t.Fatal(err)
	}
	bobID, err := s.CreateUser("bob", "password123")
	if err != nil {
		t.Fatal(err)
	}
	aliceToken, err := auth.GenerateToken(aliceID)
	if err != nil {
		t.Fatal(err)
	}
	bobToken, err := auth.GenerateToken(bobID)
	if err != nil {
		t.Fatal(err)
	}

	hub := ws.NewHub()
	go hub.Run()
	defer hub.Shutdown()

	apiHandler := NewAPI(s, hub)

	t.Run("send invite and list invites", func(t *testing.T) {
		reqBody, _ := json.Marshal(map[string]string{"username": "bob"})
		req := httptest.NewRequest(http.MethodPost, "/api/invites/send", bytes.NewReader(reqBody))
		req.Header.Set("Authorization", "Bearer "+aliceToken)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		apiHandler.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Fatalf("send invite status = %d, want %d; body=%s", rr.Code, http.StatusCreated, rr.Body.String())
		}

		req = httptest.NewRequest(http.MethodGet, "/api/invites", nil)
		req.Header.Set("Authorization", "Bearer "+bobToken)
		rr = httptest.NewRecorder()

		apiHandler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("list invites status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
		}

		var invites []map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &invites); err != nil {
			t.Fatalf("unmarshal invites: %v", err)
		}
		if len(invites) != 1 {
			t.Fatalf("invites len = %d, want 1", len(invites))
		}
		if got := invites[0]["inviter_username"]; got != "alice" {
			t.Fatalf("inviter_username = %v, want alice", got)
		}
		if _, ok := invites[0]["id"]; !ok {
			t.Fatal("expected invite id field")
		}
	})

	t.Run("accept invite creates bidirectional contacts and preserves contact shape", func(t *testing.T) {
		var inviteID int
		if err := s.DB.QueryRow(`SELECT id FROM contact_invites WHERE recipient_id = ? ORDER BY id DESC LIMIT 1`, bobID).Scan(&inviteID); err != nil {
			t.Fatalf("query invite id: %v", err)
		}

		reqBody, _ := json.Marshal(map[string]int{"invite_id": inviteID})
		req := httptest.NewRequest(http.MethodPost, "/api/invites/accept", bytes.NewReader(reqBody))
		req.Header.Set("Authorization", "Bearer "+bobToken)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		apiHandler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("accept invite status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
		}

		req = httptest.NewRequest(http.MethodGet, "/api/contacts", nil)
		req.Header.Set("Authorization", "Bearer "+aliceToken)
		rr = httptest.NewRecorder()

		apiHandler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("list contacts status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
		}

		var contacts []map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &contacts); err != nil {
			t.Fatalf("unmarshal contacts: %v", err)
		}
		if len(contacts) != 1 {
			t.Fatalf("contacts len = %d, want 1", len(contacts))
		}
		if got := int(contacts[0]["id"].(float64)); got != bobID {
			t.Fatalf("contact id = %d, want %d", got, bobID)
		}
		if got := contacts[0]["username"]; got != "bob" {
			t.Fatalf("username = %v, want bob", got)
		}
		if _, ok := contacts[0]["display_name"]; !ok {
			t.Fatal("expected display_name field")
		}
		if _, ok := contacts[0]["avatar_url"]; !ok {
			t.Fatal("expected avatar_url field")
		}
	})
}

func TestMeAndWalletRoutes_Compatibility(t *testing.T) {
	if err := auth.ConfigureJWT("test-secret-123456"); err != nil {
		t.Fatal(err)
	}

	s, err := store.NewSqliteStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.DB.Close() })

	if err := migrate.RunMigrations(s.DB, "../../migrations"); err != nil {
		t.Fatal(err)
	}

	aliceID, err := s.CreateUser("alice", "password123")
	if err != nil {
		t.Fatal(err)
	}
	bobID, err := s.CreateUser("bob", "password123")
	if err != nil {
		t.Fatal(err)
	}
	aliceToken, err := auth.GenerateToken(aliceID)
	if err != nil {
		t.Fatal(err)
	}

	// Seed wallet balances using the existing sqlite schema.
	if _, err := s.DB.Exec(`INSERT OR IGNORE INTO wallet_accounts (user_id) VALUES (?), (?)`, aliceID, bobID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB.Exec(`UPDATE wallet_accounts SET balance_cents = CASE user_id WHEN ? THEN 2000 WHEN ? THEN 300 ELSE balance_cents END`, aliceID, bobID); err != nil {
		t.Fatal(err)
	}

	hub := ws.NewHub()
	go hub.Run()
	defer hub.Shutdown()
	apiHandler := NewAPI(s, hub)

	t.Run("me route preserves response shape", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
		req.Header.Set("Authorization", "Bearer "+aliceToken)
		rr := httptest.NewRecorder()

		apiHandler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
		}
		var resp map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal me response: %v", err)
		}
		if got := int(resp["id"].(float64)); got != aliceID {
			t.Fatalf("id = %d, want %d", got, aliceID)
		}
		if got := resp["username"].(string); got != "alice" {
			t.Fatalf("username = %q, want alice", got)
		}
		if _, ok := resp["display_name"]; !ok {
			t.Fatal("expected display_name field")
		}
		if _, ok := resp["avatar_url"]; !ok {
			t.Fatal("expected avatar_url field")
		}
	})

	t.Run("wallet get route preserves response shape", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/wallet", nil)
		req.Header.Set("Authorization", "Bearer "+aliceToken)
		rr := httptest.NewRecorder()

		apiHandler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
		}
		var resp map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal wallet response: %v", err)
		}
		if got := int(resp["user_id"].(float64)); got != aliceID {
			t.Fatalf("user_id = %d, want %d", got, aliceID)
		}
		if got := int(resp["balance_cents"].(float64)); got != 2000 {
			t.Fatalf("balance_cents = %d, want 2000", got)
		}
	})

	t.Run("wallet send success and insufficient funds", func(t *testing.T) {
		sendBody, _ := json.Marshal(map[string]any{"username": "bob", "amount": 5.25})
		req := httptest.NewRequest(http.MethodPost, "/api/wallet/send", bytes.NewReader(sendBody))
		req.Header.Set("Authorization", "Bearer "+aliceToken)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		apiHandler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("send success status = %d, want 200; body=%s", rr.Code, rr.Body.String())
		}

		var aliceBal, bobBal int64
		if err := s.DB.QueryRow(`SELECT balance_cents FROM wallet_accounts WHERE user_id = ?`, aliceID).Scan(&aliceBal); err != nil {
			t.Fatal(err)
		}
		if err := s.DB.QueryRow(`SELECT balance_cents FROM wallet_accounts WHERE user_id = ?`, bobID).Scan(&bobBal); err != nil {
			t.Fatal(err)
		}
		if aliceBal != 1475 || bobBal != 825 {
			t.Fatalf("unexpected balances after send: alice=%d bob=%d", aliceBal, bobBal)
		}

		sendBody, _ = json.Marshal(map[string]any{"username": "bob", "amount": 50.00})
		req = httptest.NewRequest(http.MethodPost, "/api/wallet/send", bytes.NewReader(sendBody))
		req.Header.Set("Authorization", "Bearer "+aliceToken)
		req.Header.Set("Content-Type", "application/json")
		rr = httptest.NewRecorder()

		apiHandler.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("send insufficient status = %d, want 400; body=%s", rr.Code, rr.Body.String())
		}
	})
}

func TestMessagesInboxRoute_AdditiveSyncEndpoint(t *testing.T) {
	if err := auth.ConfigureJWT("test-secret-123456"); err != nil {
		t.Fatal(err)
	}

	s, err := store.NewSqliteStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.DB.Close() })

	if err := migrate.RunMigrations(s.DB, "../../migrations"); err != nil {
		t.Fatal(err)
	}

	aliceID, err := s.CreateUser("alice", "password123")
	if err != nil {
		t.Fatal(err)
	}
	bobID, err := s.CreateUser("bob", "password123")
	if err != nil {
		t.Fatal(err)
	}
	token, err := auth.GenerateToken(bobID)
	if err != nil {
		t.Fatal(err)
	}

	// Seed durable messages + receipt metadata using the existing schema.
	res, err := s.DB.Exec(`INSERT INTO messages (from_user_id, to_user_id, body) VALUES (?, ?, ?), (?, ?, ?)`,
		aliceID, bobID, "hello", aliceID, bobID, "second")
	if err != nil {
		t.Fatal(err)
	}
	lastID, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	firstID := lastID - 1
	if _, err := s.DB.Exec(`INSERT INTO message_deliveries (message_id, delivered_at) VALUES (?, CURRENT_TIMESTAMP)`, firstID); err != nil {
		t.Fatal(err)
	}

	hub := ws.NewHub()
	go hub.Run()
	defer hub.Shutdown()
	apiHandler := NewAPI(s, hub)

	t.Run("unauthorized without token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/messages/inbox", nil)
		rr := httptest.NewRecorder()
		apiHandler.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rr.Code)
		}
	})

	t.Run("returns inbox messages in descending order", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/messages/inbox?limit=10", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		apiHandler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
		}

		var resp []map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		if len(resp) != 2 {
			t.Fatalf("expected 2 messages, got %d", len(resp))
		}
		if got := int64(resp[0]["id"].(float64)); got != lastID {
			t.Fatalf("first id = %d, want %d", got, lastID)
		}
		if got := resp[0]["body"].(string); got != "second" {
			t.Fatalf("first body = %q, want second", got)
		}
		if got := int(resp[0]["to_user_id"].(float64)); got != bobID {
			t.Fatalf("to_user_id = %d, want %d", got, bobID)
		}
		if _, ok := resp[0]["created_at"]; !ok {
			t.Fatal("expected created_at field")
		}
		if _, ok := resp[1]["delivered_at"]; !ok {
			t.Fatal("expected delivered_at field")
		}
	})
}
