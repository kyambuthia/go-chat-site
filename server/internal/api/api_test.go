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
