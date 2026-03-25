package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/kyambuthia/go-chat-site/server/internal/auth"
	coreid "github.com/kyambuthia/go-chat-site/server/internal/core/identity"
	"github.com/kyambuthia/go-chat-site/server/internal/web"
)

type sessionDisconnector interface {
	DisconnectSession(sessionID int64)
}

type AuthHandler struct {
	Identity   coreid.AuthService
	Sessions   coreid.SessionService
	Security   *authSecurity
	SessionHub sessionDisconnector
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		web.JSONError(w, errors.New("method not allowed"), http.StatusMethodNotAllowed)
		return
	}

	var creds struct {
		Username    string `json:"username"`
		Password    string `json:"password"`
		DeviceLabel string `json:"device_label"`
	}

	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		web.JSONError(w, errors.New("invalid request body"), http.StatusBadRequest)
		return
	}

	creds.Username = strings.TrimSpace(creds.Username)
	if creds.Username == "" {
		web.JSONError(w, errors.New("username is required"), http.StatusBadRequest)
		return
	}

	if len(creds.Password) < 8 {
		web.JSONError(w, errors.New("password too short"), http.StatusBadRequest)
		return
	}

	principal, err := h.Identity.RegisterPassword(r.Context(), coreid.PasswordCredential{
		Username: creds.Username,
		Password: creds.Password,
	})
	if err != nil {
		web.JSONError(w, err, http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"id":       int(principal.ID),
		"username": creds.Username,
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		web.JSONError(w, errors.New("method not allowed"), http.StatusMethodNotAllowed)
		return
	}

	var creds struct {
		Username    string `json:"username"`
		Password    string `json:"password"`
		DeviceLabel string `json:"device_label"`
	}

	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		web.JSONError(w, errors.New("invalid request body"), http.StatusBadRequest)
		return
	}

	username := strings.TrimSpace(creds.Username)
	ip := clientIP(r)
	requestID := r.Header.Get("X-Request-ID")
	if h.Security != nil {
		if err := h.Security.allowLogin(r.Context(), username, ip, requestID); err != nil {
			writeAuthThrottleError(w, err)
			return
		}
	}

	tokens, err := h.Identity.LoginPassword(r.Context(), coreid.PasswordCredential{
		Username: username,
		Password: creds.Password,
	}, sessionMetadataFromRequest(r, creds.DeviceLabel))
	if err != nil {
		if errors.Is(err, coreid.ErrInvalidCredentials) {
			if h.Security != nil {
				h.Security.recordLoginFailure(r.Context(), username, ip, requestID)
			}
			web.JSONError(w, errors.New("invalid username or password"), http.StatusUnauthorized)
			return
		}
		web.JSONError(w, err, http.StatusInternalServerError)
		return
	}
	if h.Security != nil {
		h.Security.recordLoginSuccess(r.Context(), int(tokens.Session.UserID), username, ip, requestID)
	}

	writeSessionTokensJSON(w, tokens)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		web.JSONError(w, errors.New("method not allowed"), http.StatusMethodNotAllowed)
		return
	}
	if h.Sessions == nil {
		web.JSONError(w, errors.New("session refresh unavailable"), http.StatusServiceUnavailable)
		return
	}

	var req struct {
		RefreshToken string `json:"refresh_token"`
		DeviceLabel  string `json:"device_label"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.JSONError(w, errors.New("invalid request body"), http.StatusBadRequest)
		return
	}

	requestID := r.Header.Get("X-Request-ID")
	ip := clientIP(r)
	if h.Security != nil {
		if err := h.Security.allowRefresh(r.Context(), ip, requestID); err != nil {
			writeAuthThrottleError(w, err)
			return
		}
	}

	tokens, err := h.Sessions.RefreshSession(r.Context(), strings.TrimSpace(req.RefreshToken), sessionMetadataFromRequest(r, req.DeviceLabel))
	if err != nil {
		auth.LogSecurityEvent("auth_refresh_failed", map[string]any{
			"request_id": requestID,
			"ip_address": ip,
			"reason":     err.Error(),
		})
		status := http.StatusUnauthorized
		if errors.Is(err, coreid.ErrRefreshTokenReplay) {
			status = http.StatusUnauthorized
		}
		web.JSONError(w, err, status)
		return
	}
	auth.LogSecurityEvent("auth_refresh_succeeded", map[string]any{
		"request_id": requestID,
		"user_id":    int(tokens.Session.UserID),
		"session_id": tokens.Session.ID,
		"ip_address": ip,
	})

	writeSessionTokensJSON(w, tokens)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		web.JSONError(w, errors.New("method not allowed"), http.StatusMethodNotAllowed)
		return
	}
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		web.JSONError(w, errors.New("unauthorized"), http.StatusUnauthorized)
		return
	}
	sessionID, ok := auth.SessionIDFromContext(r.Context())
	if !ok || sessionID <= 0 {
		web.JSONError(w, errors.New("session not found"), http.StatusNotFound)
		return
	}
	if h.Sessions == nil {
		web.JSONError(w, errors.New("session management unavailable"), http.StatusServiceUnavailable)
		return
	}
	if err := h.Sessions.RevokeSession(r.Context(), coreid.UserID(userID), sessionID); err != nil {
		if errors.Is(err, coreid.ErrSessionNotFound) {
			web.JSONError(w, errors.New("session not found"), http.StatusNotFound)
			return
		}
		web.JSONError(w, err, http.StatusInternalServerError)
		return
	}
	if h.SessionHub != nil {
		h.SessionHub.DisconnectSession(sessionID)
	}
	auth.LogSecurityEvent("auth_session_revoked", map[string]any{
		"request_id": r.Header.Get("X-Request-ID"),
		"user_id":    userID,
		"session_id": sessionID,
		"reason":     "logout",
	})
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) GetSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		web.JSONError(w, errors.New("method not allowed"), http.StatusMethodNotAllowed)
		return
	}
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		web.JSONError(w, errors.New("unauthorized"), http.StatusUnauthorized)
		return
	}
	currentSessionID, _ := auth.SessionIDFromContext(r.Context())
	if h.Sessions == nil {
		web.JSONError(w, errors.New("session management unavailable"), http.StatusServiceUnavailable)
		return
	}

	sessions, err := h.Sessions.ListSessions(r.Context(), coreid.UserID(userID))
	if err != nil {
		web.JSONError(w, err, http.StatusInternalServerError)
		return
	}

	resp := make([]map[string]any, 0, len(sessions))
	for _, session := range sessions {
		item := map[string]any{
			"id":                       session.ID,
			"device_label":             session.DeviceLabel,
			"user_agent":               session.UserAgent,
			"last_seen_ip":             session.LastSeenIP,
			"created_at":               session.CreatedAt,
			"last_seen_at":             session.LastSeenAt,
			"access_token_expires_at":  session.AccessTokenExpiresAt,
			"refresh_token_expires_at": session.RefreshTokenExpiresAt,
			"current":                  session.ID == currentSessionID,
		}
		if session.RevokedAt != nil {
			item["revoked_at"] = *session.RevokedAt
		}
		resp = append(resp, item)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *AuthHandler) RevokeSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		web.JSONError(w, errors.New("method not allowed"), http.StatusMethodNotAllowed)
		return
	}
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		web.JSONError(w, errors.New("unauthorized"), http.StatusUnauthorized)
		return
	}
	if h.Sessions == nil {
		web.JSONError(w, errors.New("session management unavailable"), http.StatusServiceUnavailable)
		return
	}

	var req struct {
		SessionID int64 `json:"session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.JSONError(w, errors.New("invalid request body"), http.StatusBadRequest)
		return
	}
	if req.SessionID <= 0 {
		web.JSONError(w, errors.New("invalid session_id"), http.StatusBadRequest)
		return
	}

	if err := h.Sessions.RevokeSession(r.Context(), coreid.UserID(userID), req.SessionID); err != nil {
		if errors.Is(err, coreid.ErrSessionNotFound) {
			web.JSONError(w, errors.New("session not found"), http.StatusNotFound)
			return
		}
		web.JSONError(w, err, http.StatusInternalServerError)
		return
	}
	if h.SessionHub != nil {
		h.SessionHub.DisconnectSession(req.SessionID)
	}
	auth.LogSecurityEvent("auth_session_revoked", map[string]any{
		"request_id": r.Header.Get("X-Request-ID"),
		"user_id":    userID,
		"session_id": req.SessionID,
		"reason":     "user_revoked",
	})
	w.WriteHeader(http.StatusNoContent)
}

func writeSessionTokensJSON(w http.ResponseWriter, tokens coreid.SessionTokens) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"token":                    tokens.AccessToken,
		"access_token":             tokens.AccessToken,
		"refresh_token":            tokens.RefreshToken,
		"access_token_expires_at":  tokens.AccessTokenExpiresAt,
		"refresh_token_expires_at": tokens.RefreshTokenExpiresAt,
		"session": map[string]any{
			"id":                       tokens.Session.ID,
			"device_label":             tokens.Session.DeviceLabel,
			"user_agent":               tokens.Session.UserAgent,
			"last_seen_ip":             tokens.Session.LastSeenIP,
			"created_at":               tokens.Session.CreatedAt,
			"last_seen_at":             tokens.Session.LastSeenAt,
			"access_token_expires_at":  tokens.Session.AccessTokenExpiresAt,
			"refresh_token_expires_at": tokens.Session.RefreshTokenExpiresAt,
		},
	})
}

func writeAuthThrottleError(w http.ResponseWriter, err error) {
	var locked loginLockedError
	if errors.As(err, &locked) {
		retryAfter := retryAfterSeconds(time.Until(locked.Until))
		if retryAfter > 0 {
			w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error":               err.Error(),
			"locked_until":        locked.Until.UTC().Format(time.RFC3339),
			"retry_after_seconds": retryAfter,
		})
		return
	}

	var limited rateLimitedError
	if errors.As(err, &limited) {
		retryAfter := retryAfterSeconds(limited.RetryAfter)
		if retryAfter > 0 {
			w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error":               errAuthRateLimited.Error(),
			"scope":               limited.Scope,
			"retry_after_seconds": retryAfter,
		})
		return
	}

	web.JSONError(w, err, http.StatusTooManyRequests)
}

func sessionMetadataFromRequest(r *http.Request, deviceLabel string) coreid.SessionMetadata {
	return coreid.SessionMetadata{
		DeviceLabel: strings.TrimSpace(deviceLabel),
		UserAgent:   r.UserAgent(),
		IPAddress:   clientIP(r),
	}
}
