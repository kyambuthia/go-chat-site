package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/kyambuthia/go-chat-site/server/internal/adapters/transport/wsrelay"
	"github.com/kyambuthia/go-chat-site/server/internal/auth"
	"github.com/kyambuthia/go-chat-site/server/internal/migrate"
	"github.com/kyambuthia/go-chat-site/server/internal/store"
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

	hub := wsrelay.NewHub()
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

	t.Run("TestAuthSessions_RefreshReplayRevokesSession", func(t *testing.T) {
		type sessionPayload struct {
			ID          int64  `json:"id"`
			DeviceLabel string `json:"device_label"`
		}
		type authResponse struct {
			Token        string         `json:"token"`
			AccessToken  string         `json:"access_token"`
			RefreshToken string         `json:"refresh_token"`
			Session      sessionPayload `json:"session"`
		}

		loginReqBody, _ := json.Marshal(map[string]string{
			"username":     "testuser",
			"password":     "password123",
			"device_label": "Browser One",
		})
		loginReq := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewBuffer(loginReqBody))
		loginReq.Header.Set("Content-Type", "application/json")
		loginRR := httptest.NewRecorder()
		a.ServeHTTP(loginRR, loginReq)

		if loginRR.Code != http.StatusOK {
			t.Fatalf("login status = %d, want %d; body=%s", loginRR.Code, http.StatusOK, loginRR.Body.String())
		}

		var loginResp authResponse
		if err := json.Unmarshal(loginRR.Body.Bytes(), &loginResp); err != nil {
			t.Fatalf("unmarshal login response: %v", err)
		}
		if loginResp.AccessToken == "" || loginResp.RefreshToken == "" || loginResp.Session.ID == 0 {
			t.Fatalf("unexpected login payload: %+v", loginResp)
		}
		if loginResp.Session.DeviceLabel != "Browser One" {
			t.Fatalf("device label = %q, want Browser One", loginResp.Session.DeviceLabel)
		}

		refreshReqBody, _ := json.Marshal(map[string]string{
			"refresh_token": loginResp.RefreshToken,
			"device_label":  "Browser One",
		})
		refreshReq := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", bytes.NewBuffer(refreshReqBody))
		refreshReq.Header.Set("Content-Type", "application/json")
		refreshRR := httptest.NewRecorder()
		a.ServeHTTP(refreshRR, refreshReq)

		if refreshRR.Code != http.StatusOK {
			t.Fatalf("refresh status = %d, want %d; body=%s", refreshRR.Code, http.StatusOK, refreshRR.Body.String())
		}

		var refreshResp authResponse
		if err := json.Unmarshal(refreshRR.Body.Bytes(), &refreshResp); err != nil {
			t.Fatalf("unmarshal refresh response: %v", err)
		}
		if refreshResp.AccessToken == loginResp.AccessToken {
			t.Fatal("expected refresh to rotate access token")
		}
		if refreshResp.RefreshToken == loginResp.RefreshToken {
			t.Fatal("expected refresh to rotate refresh token")
		}
		if refreshResp.Session.ID != loginResp.Session.ID {
			t.Fatalf("session id = %d, want %d", refreshResp.Session.ID, loginResp.Session.ID)
		}

		replayReqBody, _ := json.Marshal(map[string]string{
			"refresh_token": loginResp.RefreshToken,
		})
		replayReq := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", bytes.NewBuffer(replayReqBody))
		replayReq.Header.Set("Content-Type", "application/json")
		replayRR := httptest.NewRecorder()
		a.ServeHTTP(replayRR, replayReq)

		if replayRR.Code != http.StatusUnauthorized {
			t.Fatalf("replay refresh status = %d, want %d; body=%s", replayRR.Code, http.StatusUnauthorized, replayRR.Body.String())
		}

		meReq := httptest.NewRequest(http.MethodGet, "/api/me", nil)
		meReq.Header.Set("Authorization", "Bearer "+refreshResp.AccessToken)
		meRR := httptest.NewRecorder()
		a.ServeHTTP(meRR, meReq)

		if meRR.Code != http.StatusUnauthorized {
			t.Fatalf("me after replay revoke status = %d, want %d; body=%s", meRR.Code, http.StatusUnauthorized, meRR.Body.String())
		}
	})

	t.Run("TestAuthSessions_ListRevokeAndLogout", func(t *testing.T) {
		type sessionPayload struct {
			ID      int64  `json:"id"`
			Current bool   `json:"current"`
			Device  string `json:"device_label"`
		}
		type authResponse struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			Session      struct {
				ID int64 `json:"id"`
			} `json:"session"`
		}

		login := func(device string) authResponse {
			reqBody, _ := json.Marshal(map[string]string{
				"username":     "testuser",
				"password":     "password123",
				"device_label": device,
			})
			req := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			a.ServeHTTP(rr, req)
			if rr.Code != http.StatusOK {
				t.Fatalf("login(%s) status = %d, want %d; body=%s", device, rr.Code, http.StatusOK, rr.Body.String())
			}

			var resp authResponse
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Fatalf("unmarshal login(%s): %v", device, err)
			}
			return resp
		}

		first := login("Browser Alpha")
		second := login("Phone Beta")

		listReq := httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
		listReq.Header.Set("Authorization", "Bearer "+first.AccessToken)
		listRR := httptest.NewRecorder()
		a.ServeHTTP(listRR, listReq)

		if listRR.Code != http.StatusOK {
			t.Fatalf("list sessions status = %d, want %d; body=%s", listRR.Code, http.StatusOK, listRR.Body.String())
		}

		var sessions []sessionPayload
		if err := json.Unmarshal(listRR.Body.Bytes(), &sessions); err != nil {
			t.Fatalf("unmarshal sessions: %v", err)
		}
		if len(sessions) < 2 {
			t.Fatalf("sessions len = %d, want at least 2", len(sessions))
		}

		var sawCurrent bool
		for _, session := range sessions {
			if session.ID == first.Session.ID && session.Current {
				sawCurrent = true
				break
			}
		}
		if !sawCurrent {
			t.Fatal("expected current session flag for first login")
		}

		revokeReqBody, _ := json.Marshal(map[string]int64{"session_id": second.Session.ID})
		revokeReq := httptest.NewRequest(http.MethodDelete, "/api/sessions", bytes.NewBuffer(revokeReqBody))
		revokeReq.Header.Set("Authorization", "Bearer "+first.AccessToken)
		revokeReq.Header.Set("Content-Type", "application/json")
		revokeRR := httptest.NewRecorder()
		a.ServeHTTP(revokeRR, revokeReq)

		if revokeRR.Code != http.StatusNoContent {
			t.Fatalf("revoke session status = %d, want %d; body=%s", revokeRR.Code, http.StatusNoContent, revokeRR.Body.String())
		}

		secondMeReq := httptest.NewRequest(http.MethodGet, "/api/me", nil)
		secondMeReq.Header.Set("Authorization", "Bearer "+second.AccessToken)
		secondMeRR := httptest.NewRecorder()
		a.ServeHTTP(secondMeRR, secondMeReq)

		if secondMeRR.Code != http.StatusUnauthorized {
			t.Fatalf("revoked session me status = %d, want %d; body=%s", secondMeRR.Code, http.StatusUnauthorized, secondMeRR.Body.String())
		}

		logoutReq := httptest.NewRequest(http.MethodPost, "/api/logout", nil)
		logoutReq.Header.Set("Authorization", "Bearer "+first.AccessToken)
		logoutRR := httptest.NewRecorder()
		a.ServeHTTP(logoutRR, logoutReq)

		if logoutRR.Code != http.StatusNoContent {
			t.Fatalf("logout status = %d, want %d; body=%s", logoutRR.Code, http.StatusNoContent, logoutRR.Body.String())
		}

		firstMeReq := httptest.NewRequest(http.MethodGet, "/api/me", nil)
		firstMeReq.Header.Set("Authorization", "Bearer "+first.AccessToken)
		firstMeRR := httptest.NewRecorder()
		a.ServeHTTP(firstMeRR, firstMeReq)

		if firstMeRR.Code != http.StatusUnauthorized {
			t.Fatalf("logged out me status = %d, want %d; body=%s", firstMeRR.Code, http.StatusUnauthorized, firstMeRR.Body.String())
		}
	})

	t.Run("TestDeviceKeys_RegisterRotateDirectoryAndRevoke", func(t *testing.T) {
		type authResponse struct {
			AccessToken string `json:"access_token"`
		}

		login := func(username string) authResponse {
			reqBody, _ := json.Marshal(map[string]string{
				"username":     username,
				"password":     "password123",
				"device_label": "Phase4 Browser",
			})
			req := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			a.ServeHTTP(rr, req)
			if rr.Code != http.StatusOK {
				t.Fatalf("login(%s) status = %d, want %d; body=%s", username, rr.Code, http.StatusOK, rr.Body.String())
			}

			var resp authResponse
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Fatalf("unmarshal login(%s): %v", username, err)
			}
			return resp
		}

		if _, err := s.CreateUser("phase4viewer", "password123"); err != nil {
			t.Fatalf("create phase4viewer: %v", err)
		}

		testUserAuth := login("testuser")
		viewerAuth := login("phase4viewer")

		registerReqBody, _ := json.Marshal(map[string]any{
			"label":                   "Alice Laptop",
			"identity_key":            "identity-key-a",
			"signed_prekey_id":        1,
			"signed_prekey":           "signed-prekey-a",
			"signed_prekey_signature": "signature-a",
			"prekeys": []map[string]any{
				{"prekey_id": 1, "public_key": "prekey-a1"},
				{"prekey_id": 2, "public_key": "prekey-a2"},
			},
		})
		registerReq := httptest.NewRequest(http.MethodPost, "/api/devices", bytes.NewBuffer(registerReqBody))
		registerReq.Header.Set("Authorization", "Bearer "+testUserAuth.AccessToken)
		registerReq.Header.Set("Content-Type", "application/json")
		registerRR := httptest.NewRecorder()
		a.ServeHTTP(registerRR, registerReq)

		if registerRR.Code != http.StatusCreated {
			t.Fatalf("register device status = %d, want %d; body=%s", registerRR.Code, http.StatusCreated, registerRR.Body.String())
		}

		var device map[string]any
		if err := json.Unmarshal(registerRR.Body.Bytes(), &device); err != nil {
			t.Fatalf("unmarshal register device response: %v", err)
		}
		deviceID := int64(device["id"].(float64))
		if device["label"] != "Alice Laptop" {
			t.Fatalf("device label = %v, want Alice Laptop", device["label"])
		}
		if device["current_session"] != true {
			t.Fatalf("expected current_session true, got %v", device["current_session"])
		}

		listReq := httptest.NewRequest(http.MethodGet, "/api/devices", nil)
		listReq.Header.Set("Authorization", "Bearer "+testUserAuth.AccessToken)
		listRR := httptest.NewRecorder()
		a.ServeHTTP(listRR, listReq)

		if listRR.Code != http.StatusOK {
			t.Fatalf("list devices status = %d, want %d; body=%s", listRR.Code, http.StatusOK, listRR.Body.String())
		}

		var devices []map[string]any
		if err := json.Unmarshal(listRR.Body.Bytes(), &devices); err != nil {
			t.Fatalf("unmarshal devices: %v", err)
		}
		if len(devices) == 0 {
			t.Fatal("expected at least one device")
		}
		if got := int(devices[0]["prekey_count"].(float64)); got < 2 {
			t.Fatalf("prekey_count = %d, want at least 2", got)
		}

		rotateReqBody, _ := json.Marshal(map[string]any{
			"device_id":               deviceID,
			"signed_prekey_id":        2,
			"signed_prekey":           "signed-prekey-b",
			"signed_prekey_signature": "signature-b",
			"prekeys": []map[string]any{
				{"prekey_id": 3, "public_key": "prekey-a3"},
			},
		})
		rotateReq := httptest.NewRequest(http.MethodPost, "/api/devices/rotate", bytes.NewBuffer(rotateReqBody))
		rotateReq.Header.Set("Authorization", "Bearer "+testUserAuth.AccessToken)
		rotateReq.Header.Set("Content-Type", "application/json")
		rotateRR := httptest.NewRecorder()
		a.ServeHTTP(rotateRR, rotateReq)

		if rotateRR.Code != http.StatusOK {
			t.Fatalf("rotate device status = %d, want %d; body=%s", rotateRR.Code, http.StatusOK, rotateRR.Body.String())
		}

		publishReqBody, _ := json.Marshal(map[string]any{
			"device_id": deviceID,
			"prekeys": []map[string]any{
				{"prekey_id": 4, "public_key": "prekey-a4"},
			},
		})
		publishReq := httptest.NewRequest(http.MethodPost, "/api/messaging/prekeys", bytes.NewBuffer(publishReqBody))
		publishReq.Header.Set("Authorization", "Bearer "+testUserAuth.AccessToken)
		publishReq.Header.Set("Content-Type", "application/json")
		publishRR := httptest.NewRecorder()
		a.ServeHTTP(publishRR, publishReq)

		if publishRR.Code != http.StatusOK {
			t.Fatalf("publish prekeys status = %d, want %d; body=%s", publishRR.Code, http.StatusOK, publishRR.Body.String())
		}

		directoryReq := httptest.NewRequest(http.MethodGet, "/api/devices/directory?username=testuser", nil)
		directoryReq.Header.Set("Authorization", "Bearer "+viewerAuth.AccessToken)
		directoryRR := httptest.NewRecorder()
		a.ServeHTTP(directoryRR, directoryReq)

		if directoryRR.Code != http.StatusOK {
			t.Fatalf("directory status = %d, want %d; body=%s", directoryRR.Code, http.StatusOK, directoryRR.Body.String())
		}

		var directory map[string]any
		if err := json.Unmarshal(directoryRR.Body.Bytes(), &directory); err != nil {
			t.Fatalf("unmarshal directory: %v", err)
		}
		dirDevices := directory["devices"].([]any)
		if len(dirDevices) != 1 {
			t.Fatalf("directory devices len = %d, want 1", len(dirDevices))
		}
		firstDevice := dirDevices[0].(map[string]any)
		if len(firstDevice["prekeys"].([]any)) < 4 {
			t.Fatalf("directory prekeys len = %d, want at least 4", len(firstDevice["prekeys"].([]any)))
		}

		revokeReqBody, _ := json.Marshal(map[string]any{"device_id": deviceID})
		revokeReq := httptest.NewRequest(http.MethodDelete, "/api/devices", bytes.NewBuffer(revokeReqBody))
		revokeReq.Header.Set("Authorization", "Bearer "+testUserAuth.AccessToken)
		revokeReq.Header.Set("Content-Type", "application/json")
		revokeRR := httptest.NewRecorder()
		a.ServeHTTP(revokeRR, revokeReq)

		if revokeRR.Code != http.StatusNoContent {
			t.Fatalf("revoke device status = %d, want %d; body=%s", revokeRR.Code, http.StatusNoContent, revokeRR.Body.String())
		}

		directoryRR = httptest.NewRecorder()
		a.ServeHTTP(directoryRR, directoryReq)
		if directoryRR.Code != http.StatusOK {
			t.Fatalf("directory after revoke status = %d, want %d; body=%s", directoryRR.Code, http.StatusOK, directoryRR.Body.String())
		}
		if err := json.Unmarshal(directoryRR.Body.Bytes(), &directory); err != nil {
			t.Fatalf("unmarshal directory after revoke: %v", err)
		}
		if len(directory["devices"].([]any)) != 0 {
			t.Fatalf("expected empty directory after revoke, got %v", directory["devices"])
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
	charlieID, err := s.CreateUser("charlie", "password123")
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

	hub := wsrelay.NewHub()
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

	t.Run("direct add and remove contact routes remain reachable", func(t *testing.T) {
		reqBody, _ := json.Marshal(map[string]string{"username": "charlie"})
		req := httptest.NewRequest(http.MethodPost, "/api/contacts", bytes.NewReader(reqBody))
		req.Header.Set("Authorization", "Bearer "+aliceToken)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		apiHandler.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Fatalf("add contact status = %d, want %d; body=%s", rr.Code, http.StatusCreated, rr.Body.String())
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
		if len(contacts) != 2 {
			t.Fatalf("contacts len = %d, want 2", len(contacts))
		}

		deleteBody, _ := json.Marshal(map[string]int{"contact_id": charlieID})
		req = httptest.NewRequest(http.MethodDelete, "/api/contacts", bytes.NewReader(deleteBody))
		req.Header.Set("Authorization", "Bearer "+aliceToken)
		req.Header.Set("Content-Type", "application/json")
		rr = httptest.NewRecorder()

		apiHandler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("remove contact status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
		}

		req = httptest.NewRequest(http.MethodGet, "/api/contacts", nil)
		req.Header.Set("Authorization", "Bearer "+aliceToken)
		rr = httptest.NewRecorder()

		apiHandler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("list contacts status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
		}

		contacts = nil
		if err := json.Unmarshal(rr.Body.Bytes(), &contacts); err != nil {
			t.Fatalf("unmarshal contacts after delete: %v", err)
		}
		if len(contacts) != 1 {
			t.Fatalf("contacts len after delete = %d, want 1", len(contacts))
		}
		if got := int(contacts[0]["id"].(float64)); got != bobID {
			t.Fatalf("remaining contact id = %d, want %d", got, bobID)
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

	hub := wsrelay.NewHub()
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

	t.Run("me route supports patch updates", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"display_name": "Alice Doe",
			"avatar_url":   "https://example.com/alice.png",
		})
		req := httptest.NewRequest(http.MethodPatch, "/api/me", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+aliceToken)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		apiHandler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
		}

		var displayName, avatarURL string
		if err := s.DB.QueryRow(`SELECT COALESCE(display_name, ''), COALESCE(avatar_url, '') FROM users WHERE id = ?`, aliceID).
			Scan(&displayName, &avatarURL); err != nil {
			t.Fatal(err)
		}
		if displayName != "Alice Doe" || avatarURL != "https://example.com/alice.png" {
			t.Fatalf("unexpected persisted profile: display_name=%q avatar_url=%q", displayName, avatarURL)
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

	t.Run("wallet transfer history returns recent transactions", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/wallet/transfers?limit=5", nil)
		req.Header.Set("Authorization", "Bearer "+aliceToken)
		rr := httptest.NewRecorder()

		apiHandler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
		}

		var resp []map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal transfer history: %v", err)
		}
		if len(resp) == 0 {
			t.Fatal("expected at least one transfer")
		}
		if got := resp[0]["direction"].(string); got != "sent" {
			t.Fatalf("direction = %q, want sent", got)
		}
		if got := resp[0]["counterparty_username"].(string); got != "bob" {
			t.Fatalf("counterparty_username = %q, want bob", got)
		}
		if _, ok := resp[0]["counterparty_avatar_url"]; !ok {
			t.Fatal("expected counterparty_avatar_url field")
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

	hub := wsrelay.NewHub()
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

	t.Run("supports before_id cursor pagination", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/messages/inbox?before_id="+strconv.FormatInt(lastID, 10)+"&limit=10", nil)
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
		if len(resp) != 1 {
			t.Fatalf("expected 1 message before cursor, got %d", len(resp))
		}
		if got := int64(resp[0]["id"].(float64)); got != firstID {
			t.Fatalf("id = %d, want %d", got, firstID)
		}
		if got := resp[0]["body"].(string); got != "hello" {
			t.Fatalf("body = %q, want hello", got)
		}
	})

	t.Run("supports after_id cursor polling", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/messages/inbox?after_id="+strconv.FormatInt(firstID, 10)+"&limit=10", nil)
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
		if len(resp) != 1 {
			t.Fatalf("expected 1 newer message, got %d", len(resp))
		}
		if got := int64(resp[0]["id"].(float64)); got != lastID {
			t.Fatalf("id = %d, want %d", got, lastID)
		}
		if got := resp[0]["body"].(string); got != "second" {
			t.Fatalf("body = %q, want second", got)
		}
	})

	t.Run("supports with_user_id filtering", func(t *testing.T) {
		charlieID, err := s.CreateUser("charlie", "password123")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := s.DB.Exec(`INSERT INTO messages (from_user_id, to_user_id, body) VALUES (?, ?, ?)`, charlieID, bobID, "charlie-msg"); err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest(http.MethodGet, "/api/messages/inbox?with_user_id="+strconv.Itoa(aliceID)+"&limit=10", nil)
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
			t.Fatalf("expected 2 alice messages only, got %d", len(resp))
		}
		for _, msg := range resp {
			if got := int(msg["from_user_id"].(float64)); got != aliceID {
				t.Fatalf("from_user_id = %d, want %d", got, aliceID)
			}
		}
	})

	t.Run("rejects invalid with_user_id filter", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/messages/inbox?with_user_id=abc", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		apiHandler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rr.Code)
		}
	})

	t.Run("supports unread_only filter", func(t *testing.T) {
		if _, err := s.DB.Exec(`INSERT INTO message_deliveries (message_id, delivered_at, read_at) VALUES (?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
			ON CONFLICT(message_id) DO UPDATE SET delivered_at=excluded.delivered_at, read_at=excluded.read_at`, firstID); err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest(http.MethodGet, "/api/messages/inbox?unread_only=true&with_user_id="+strconv.Itoa(aliceID)+"&limit=10", nil)
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
		if len(resp) != 1 {
			t.Fatalf("expected 1 unread message, got %d", len(resp))
		}
		if got := int64(resp[0]["id"].(float64)); got != lastID {
			t.Fatalf("id = %d, want %d", got, lastID)
		}
		if _, ok := resp[0]["read_at"]; ok {
			t.Fatal("did not expect read_at field on unread-only result")
		}
	})

	t.Run("rejects invalid unread_only filter", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/messages/inbox?unread_only=maybe", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		apiHandler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rr.Code)
		}
	})

	t.Run("returns outbox messages in descending order", func(t *testing.T) {
		res, err := s.DB.Exec(`INSERT INTO messages (from_user_id, to_user_id, body) VALUES (?, ?, ?), (?, ?, ?)`,
			bobID, aliceID, "out-1", bobID, aliceID, "out-2")
		if err != nil {
			t.Fatal(err)
		}
		lastID, err := res.LastInsertId()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := s.DB.Exec(`INSERT INTO message_client_correlations (sender_user_id, recipient_user_id, client_message_id, stored_message_id, delivered)
			VALUES (?, ?, ?, ?, ?)`, bobID, aliceID, 7001, lastID, 1); err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest(http.MethodGet, "/api/messages/outbox?limit=10", nil)
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
		if len(resp) < 2 {
			t.Fatalf("expected at least 2 outbox messages, got %d", len(resp))
		}
		if got := int(resp[0]["from_user_id"].(float64)); got != bobID {
			t.Fatalf("from_user_id = %d, want %d", got, bobID)
		}
		if got := int(resp[0]["to_user_id"].(float64)); got != aliceID {
			t.Fatalf("to_user_id = %d, want %d", got, aliceID)
		}
		if got := int64(resp[0]["client_message_id"].(float64)); got != 7001 {
			t.Fatalf("client_message_id = %d, want 7001", got)
		}
	})

	t.Run("supports outbox before_id and after_id cursors", func(t *testing.T) {
		res, err := s.DB.Exec(`INSERT INTO messages (from_user_id, to_user_id, body) VALUES (?, ?, ?), (?, ?, ?), (?, ?, ?)`,
			bobID, aliceID, "ob-1",
			bobID, aliceID, "ob-2",
			bobID, aliceID, "ob-3")
		if err != nil {
			t.Fatal(err)
		}
		lastOutboxID, err := res.LastInsertId()
		if err != nil {
			t.Fatal(err)
		}
		firstOutboxID := lastOutboxID - 2

		reqBefore := httptest.NewRequest(http.MethodGet, "/api/messages/outbox?before_id="+strconv.FormatInt(lastOutboxID, 10)+"&limit=10", nil)
		reqBefore.Header.Set("Authorization", "Bearer "+token)
		rrBefore := httptest.NewRecorder()
		apiHandler.ServeHTTP(rrBefore, reqBefore)
		if rrBefore.Code != http.StatusOK {
			t.Fatalf("before status = %d, want 200; body=%s", rrBefore.Code, rrBefore.Body.String())
		}
		var beforeResp []map[string]any
		if err := json.Unmarshal(rrBefore.Body.Bytes(), &beforeResp); err != nil {
			t.Fatalf("unmarshal before response: %v", err)
		}
		if len(beforeResp) < 2 {
			t.Fatalf("expected at least 2 outbox messages before cursor, got %d", len(beforeResp))
		}

		reqAfter := httptest.NewRequest(http.MethodGet, "/api/messages/outbox?after_id="+strconv.FormatInt(firstOutboxID, 10)+"&limit=10", nil)
		reqAfter.Header.Set("Authorization", "Bearer "+token)
		rrAfter := httptest.NewRecorder()
		apiHandler.ServeHTTP(rrAfter, reqAfter)
		if rrAfter.Code != http.StatusOK {
			t.Fatalf("after status = %d, want 200; body=%s", rrAfter.Code, rrAfter.Body.String())
		}
		var afterResp []map[string]any
		if err := json.Unmarshal(rrAfter.Body.Bytes(), &afterResp); err != nil {
			t.Fatalf("unmarshal after response: %v", err)
		}
		if len(afterResp) < 2 {
			t.Fatalf("expected at least 2 outbox messages after cursor, got %d", len(afterResp))
		}
	})
}

func TestMessagingSyncRoute_CursorStream(t *testing.T) {
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

	res, err := s.DB.Exec(`INSERT INTO messages (from_user_id, to_user_id, body) VALUES
		(?, ?, ?),
		(?, ?, ?),
		(?, ?, ?),
		(?, ?, ?)`,
		aliceID, bobID, "in-1",
		bobID, aliceID, "out-2",
		aliceID, bobID, "in-3",
		bobID, aliceID, "out-4")
	if err != nil {
		t.Fatal(err)
	}
	lastID, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	firstID := lastID - 3

	if _, err := s.DB.Exec(`INSERT INTO message_deliveries (message_id, delivered_at) VALUES (?, CURRENT_TIMESTAMP)`, lastID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB.Exec(`INSERT INTO message_client_correlations (sender_user_id, recipient_user_id, client_message_id, stored_message_id, delivered)
		VALUES (?, ?, ?, ?, ?)`, bobID, aliceID, 4002, firstID+1, 1); err != nil {
		t.Fatal(err)
	}

	hub := wsrelay.NewHub()
	go hub.Run()
	defer hub.Shutdown()
	apiHandler := NewAPI(s, hub)

	req := httptest.NewRequest(http.MethodGet, "/api/messaging/sync?after_id="+strconv.FormatInt(firstID, 10)+"&limit=2", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	apiHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal sync response: %v", err)
	}
	if got := resp["has_more"].(bool); !got {
		t.Fatal("expected has_more to be true")
	}

	cursor := resp["cursor"].(map[string]any)
	if got := int64(cursor["after_id"].(float64)); got != firstID {
		t.Fatalf("cursor.after_id = %d, want %d", got, firstID)
	}
	if got := int64(cursor["next_after_id"].(float64)); got != firstID+2 {
		t.Fatalf("cursor.next_after_id = %d, want %d", got, firstID+2)
	}

	messages := resp["messages"].([]any)
	if len(messages) != 2 {
		t.Fatalf("messages len = %d, want 2", len(messages))
	}

	first := messages[0].(map[string]any)
	second := messages[1].(map[string]any)
	if got := int64(first["id"].(float64)); got != firstID+1 {
		t.Fatalf("first id = %d, want %d", got, firstID+1)
	}
	if got := int64(first["client_message_id"].(float64)); got != 4002 {
		t.Fatalf("first client_message_id = %d, want 4002", got)
	}
	if got := int64(second["id"].(float64)); got != firstID+2 {
		t.Fatalf("second id = %d, want %d", got, firstID+2)
	}
	if _, ok := second["delivered_at"]; ok {
		t.Fatal("did not expect delivered_at on unsynced middle message")
	}
}

func TestMessagingThreadsRoute_DurableThreadSummaries(t *testing.T) {
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
	charlieID, err := s.CreateUser("charlie", "password123")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB.Exec(`UPDATE users SET display_name = ?, avatar_url = ? WHERE id = ?`, "Alice", "https://example.com/alice.png", aliceID); err != nil {
		t.Fatal(err)
	}
	token, err := auth.GenerateToken(bobID)
	if err != nil {
		t.Fatal(err)
	}

	res, err := s.DB.Exec(`INSERT INTO messages (from_user_id, to_user_id, body) VALUES
		(?, ?, ?),
		(?, ?, ?),
		(?, ?, ?)`,
		aliceID, bobID, "alice-1",
		bobID, aliceID, "bob-2",
		charlieID, bobID, "charlie-3")
	if err != nil {
		t.Fatal(err)
	}
	lastID, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	aliceLatestID := lastID - 1

	if _, err := s.DB.Exec(`INSERT INTO message_deliveries (message_id, delivered_at, read_at) VALUES (?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, aliceLatestID); err != nil {
		t.Fatal(err)
	}

	hub := wsrelay.NewHub()
	go hub.Run()
	defer hub.Shutdown()
	apiHandler := NewAPI(s, hub)

	for _, route := range []string{
		"/api/messaging/threads?limit=10",
		"/api/messages/threads?limit=10",
	} {
		req := httptest.NewRequest(http.MethodGet, route, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		apiHandler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want 200; body=%s", route, rr.Code, rr.Body.String())
		}

		var resp []map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("%s unmarshal response: %v", route, err)
		}
		if len(resp) != 2 {
			t.Fatalf("%s expected 2 summaries, got %d", route, len(resp))
		}
		if got := resp[0]["username"].(string); got != "charlie" {
			t.Fatalf("%s first username = %q, want charlie", route, got)
		}
		if got := int(resp[0]["unread_count"].(float64)); got != 1 {
			t.Fatalf("%s first unread_count = %d, want 1", route, got)
		}
		if got := resp[1]["username"].(string); got != "alice" {
			t.Fatalf("%s second username = %q, want alice", route, got)
		}
		if got := resp[1]["display_name"].(string); got != "Alice" {
			t.Fatalf("%s display_name = %q, want Alice", route, got)
		}
		if got := resp[1]["avatar_url"].(string); got != "https://example.com/alice.png" {
			t.Fatalf("%s avatar_url = %q", route, got)
		}
		lastMessage := resp[1]["last_message"].(map[string]any)
		if got := int64(lastMessage["id"].(float64)); got != aliceLatestID {
			t.Fatalf("%s last_message.id = %d, want %d", route, got, aliceLatestID)
		}
		if got := lastMessage["body"].(string); got != "bob-2" {
			t.Fatalf("%s last_message.body = %q, want bob-2", route, got)
		}
		if _, ok := lastMessage["read_at"]; !ok {
			t.Fatalf("%s expected read_at on alice thread", route)
		}
	}
}

func TestMessagesReadRoute_AdditiveReceiptEndpoint(t *testing.T) {
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

	res, err := s.DB.Exec(`INSERT INTO messages (from_user_id, to_user_id, body) VALUES (?, ?, ?)`, aliceID, bobID, "hello")
	if err != nil {
		t.Fatal(err)
	}
	messageID, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}

	hub := wsrelay.NewHub()
	go hub.Run()
	defer hub.Shutdown()
	apiHandler := NewAPI(s, hub)

	t.Run("unauthorized without token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/messages/read", bytes.NewReader([]byte(`{"message_id":1}`)))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		apiHandler.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rr.Code)
		}
	})

	t.Run("marks read and delivered timestamps", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/messages/read", bytes.NewReader([]byte(`{"message_id":`+strconv.FormatInt(messageID, 10)+`}`)))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		apiHandler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
		}

		var deliveredAt, readAt sql.NullTime
		if err := s.DB.QueryRow(`SELECT delivered_at, read_at FROM message_deliveries WHERE message_id = ?`, messageID).Scan(&deliveredAt, &readAt); err != nil {
			t.Fatalf("query message_deliveries: %v", err)
		}
		if !deliveredAt.Valid || !readAt.Valid {
			t.Fatalf("expected delivered/read timestamps, got delivered=%v read=%v", deliveredAt.Valid, readAt.Valid)
		}
	})

	t.Run("invalid request body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/messages/read", bytes.NewReader([]byte(`{"message_id":0}`)))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		apiHandler.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rr.Code)
		}
	})

	t.Run("returns not found for message owned by another recipient", func(t *testing.T) {
		charlieID, err := s.CreateUser("charlie", "password123")
		if err != nil {
			t.Fatal(err)
		}
		res, err := s.DB.Exec(`INSERT INTO messages (from_user_id, to_user_id, body) VALUES (?, ?, ?)`, aliceID, charlieID, "secret")
		if err != nil {
			t.Fatal(err)
		}
		otherMsgID, err := res.LastInsertId()
		if err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest(http.MethodPost, "/api/messages/read", bytes.NewReader([]byte(`{"message_id":`+strconv.FormatInt(otherMsgID, 10)+`}`)))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		apiHandler.ServeHTTP(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404; body=%s", rr.Code, rr.Body.String())
		}
	})
}

func TestMessagesDeliveredRoute_AdditiveReceiptEndpoint(t *testing.T) {
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

	res, err := s.DB.Exec(`INSERT INTO messages (from_user_id, to_user_id, body) VALUES (?, ?, ?)`, aliceID, bobID, "hello")
	if err != nil {
		t.Fatal(err)
	}
	messageID, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}

	hub := wsrelay.NewHub()
	go hub.Run()
	defer hub.Shutdown()
	apiHandler := NewAPI(s, hub)

	t.Run("marks delivered timestamp only", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/messages/delivered", bytes.NewReader([]byte(`{"message_id":`+strconv.FormatInt(messageID, 10)+`}`)))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		apiHandler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
		}

		var deliveredAt, readAt sql.NullTime
		if err := s.DB.QueryRow(`SELECT delivered_at, read_at FROM message_deliveries WHERE message_id = ?`, messageID).Scan(&deliveredAt, &readAt); err != nil {
			t.Fatalf("query message_deliveries: %v", err)
		}
		if !deliveredAt.Valid {
			t.Fatal("expected delivered_at to be set")
		}
		if readAt.Valid {
			t.Fatal("expected read_at to remain null")
		}
	})

	t.Run("invalid request body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/messages/delivered", bytes.NewReader([]byte(`{"message_id":0}`)))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		apiHandler.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rr.Code)
		}
	})

	t.Run("returns not found for message owned by another recipient", func(t *testing.T) {
		charlieID, err := s.CreateUser("charlie", "password123")
		if err != nil {
			t.Fatal(err)
		}
		res, err := s.DB.Exec(`INSERT INTO messages (from_user_id, to_user_id, body) VALUES (?, ?, ?)`, aliceID, charlieID, "secret")
		if err != nil {
			t.Fatal(err)
		}
		otherMsgID, err := res.LastInsertId()
		if err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest(http.MethodPost, "/api/messages/delivered", bytes.NewReader([]byte(`{"message_id":`+strconv.FormatInt(otherMsgID, 10)+`}`)))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		apiHandler.ServeHTTP(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404; body=%s", rr.Code, rr.Body.String())
		}
	})
}
