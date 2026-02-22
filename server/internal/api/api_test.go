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
