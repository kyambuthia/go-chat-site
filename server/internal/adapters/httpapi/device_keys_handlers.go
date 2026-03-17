package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/kyambuthia/go-chat-site/server/internal/auth"
	coreid "github.com/kyambuthia/go-chat-site/server/internal/core/identity"
	"github.com/kyambuthia/go-chat-site/server/internal/store"
	"github.com/kyambuthia/go-chat-site/server/internal/web"
)

type DeviceKeysHandler struct {
	Devices coreid.DeviceIdentityService
}

func (h *DeviceKeysHandler) GetDevices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		web.JSONError(w, errors.New("method not allowed"), http.StatusMethodNotAllowed)
		return
	}
	if h.Devices == nil {
		web.JSONError(w, errors.New("device identity service unavailable"), http.StatusServiceUnavailable)
		return
	}

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		web.JSONError(w, errors.New("unauthorized"), http.StatusUnauthorized)
		return
	}

	devices, err := h.Devices.ListDeviceIdentities(r.Context(), coreid.UserID(userID), currentSessionID(r))
	if err != nil {
		web.JSONError(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(deviceIdentitiesToJSON(devices))
}

func (h *DeviceKeysHandler) RegisterDevice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		web.JSONError(w, errors.New("method not allowed"), http.StatusMethodNotAllowed)
		return
	}
	if h.Devices == nil {
		web.JSONError(w, errors.New("device identity service unavailable"), http.StatusServiceUnavailable)
		return
	}

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		web.JSONError(w, errors.New("unauthorized"), http.StatusUnauthorized)
		return
	}

	var req struct {
		Label                 string                      `json:"label"`
		Algorithm             string                      `json:"algorithm"`
		IdentityKey           string                      `json:"identity_key"`
		SignedPrekeyID        int64                       `json:"signed_prekey_id"`
		SignedPrekey          string                      `json:"signed_prekey"`
		SignedPrekeySignature string                      `json:"signed_prekey_signature"`
		Prekeys               []coreid.DevicePrekeyUpload `json:"prekeys"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.JSONError(w, errors.New("invalid request body"), http.StatusBadRequest)
		return
	}

	device, err := h.Devices.RegisterDeviceIdentity(r.Context(), coreid.UserID(userID), currentSessionID(r), coreid.RegisterDeviceIdentityRequest{
		Label:                 req.Label,
		Algorithm:             req.Algorithm,
		IdentityKey:           req.IdentityKey,
		SignedPrekeyID:        req.SignedPrekeyID,
		SignedPrekey:          req.SignedPrekey,
		SignedPrekeySignature: req.SignedPrekeySignature,
		Prekeys:               req.Prekeys,
	})
	if err != nil {
		web.JSONError(w, err, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(deviceIdentityToJSON(device))
}

func (h *DeviceKeysHandler) RotateDevice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		web.JSONError(w, errors.New("method not allowed"), http.StatusMethodNotAllowed)
		return
	}
	if h.Devices == nil {
		web.JSONError(w, errors.New("device identity service unavailable"), http.StatusServiceUnavailable)
		return
	}

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		web.JSONError(w, errors.New("unauthorized"), http.StatusUnauthorized)
		return
	}

	var req struct {
		DeviceID              int64                       `json:"device_id"`
		SignedPrekeyID        int64                       `json:"signed_prekey_id"`
		SignedPrekey          string                      `json:"signed_prekey"`
		SignedPrekeySignature string                      `json:"signed_prekey_signature"`
		Prekeys               []coreid.DevicePrekeyUpload `json:"prekeys"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.JSONError(w, errors.New("invalid request body"), http.StatusBadRequest)
		return
	}

	device, err := h.Devices.RotateDeviceIdentity(r.Context(), coreid.UserID(userID), currentSessionID(r), coreid.RotateDeviceIdentityRequest{
		DeviceID:              req.DeviceID,
		SignedPrekeyID:        req.SignedPrekeyID,
		SignedPrekey:          req.SignedPrekey,
		SignedPrekeySignature: req.SignedPrekeySignature,
		Prekeys:               req.Prekeys,
	})
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, coreid.ErrDeviceIdentityNotFound) {
			status = http.StatusNotFound
		}
		web.JSONError(w, err, status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(deviceIdentityToJSON(device))
}

func (h *DeviceKeysHandler) PublishPrekeys(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		web.JSONError(w, errors.New("method not allowed"), http.StatusMethodNotAllowed)
		return
	}
	if h.Devices == nil {
		web.JSONError(w, errors.New("device identity service unavailable"), http.StatusServiceUnavailable)
		return
	}

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		web.JSONError(w, errors.New("unauthorized"), http.StatusUnauthorized)
		return
	}

	var req struct {
		DeviceID int64                       `json:"device_id"`
		Prekeys  []coreid.DevicePrekeyUpload `json:"prekeys"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.JSONError(w, errors.New("invalid request body"), http.StatusBadRequest)
		return
	}

	prekeys, err := h.Devices.PublishPrekeys(r.Context(), coreid.UserID(userID), coreid.PublishPrekeysRequest{
		DeviceID: req.DeviceID,
		Prekeys:  req.Prekeys,
	})
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, coreid.ErrDeviceIdentityNotFound) {
			status = http.StatusNotFound
		}
		web.JSONError(w, err, status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(devicePrekeysToJSON(prekeys))
}

func (h *DeviceKeysHandler) RevokeDevice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		web.JSONError(w, errors.New("method not allowed"), http.StatusMethodNotAllowed)
		return
	}
	if h.Devices == nil {
		web.JSONError(w, errors.New("device identity service unavailable"), http.StatusServiceUnavailable)
		return
	}

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		web.JSONError(w, errors.New("unauthorized"), http.StatusUnauthorized)
		return
	}

	var req struct {
		DeviceID int64 `json:"device_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.JSONError(w, errors.New("invalid request body"), http.StatusBadRequest)
		return
	}

	if err := h.Devices.RevokeDeviceIdentity(r.Context(), coreid.UserID(userID), req.DeviceID); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, coreid.ErrDeviceIdentityNotFound) {
			status = http.StatusNotFound
		}
		web.JSONError(w, err, status)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *DeviceKeysHandler) GetDirectory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		web.JSONError(w, errors.New("method not allowed"), http.StatusMethodNotAllowed)
		return
	}
	if h.Devices == nil {
		web.JSONError(w, errors.New("device identity service unavailable"), http.StatusServiceUnavailable)
		return
	}
	if _, ok := auth.UserIDFromContext(r.Context()); !ok {
		web.JSONError(w, errors.New("unauthorized"), http.StatusUnauthorized)
		return
	}

	directory, err := h.Devices.GetDeviceDirectory(r.Context(), r.URL.Query().Get("username"))
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, store.ErrNotFound) {
			status = http.StatusNotFound
		}
		web.JSONError(w, err, status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"user_id":  directory.UserID,
		"username": directory.Username,
		"devices":  deviceDirectoryToJSON(directory.Devices),
	})
}

func currentSessionID(r *http.Request) int64 {
	sessionID, _ := auth.SessionIDFromContext(r.Context())
	return sessionID
}

func deviceIdentityToJSON(device coreid.DeviceIdentity) map[string]any {
	item := map[string]any{
		"id":                      device.ID,
		"user_id":                 device.UserID,
		"label":                   device.Label,
		"algorithm":               device.Algorithm,
		"identity_key":            device.IdentityKey,
		"signed_prekey_id":        device.SignedPrekeyID,
		"signed_prekey":           device.SignedPrekey,
		"signed_prekey_signature": device.SignedPrekeySignature,
		"state":                   device.State,
		"prekey_count":            device.PrekeyCount,
		"current_session":         device.CurrentSession,
		"created_at":              device.CreatedAt,
		"published_at":            device.PublishedAt,
		"rotated_at":              device.RotatedAt,
	}
	if device.RevokedAt != nil {
		item["revoked_at"] = *device.RevokedAt
	}
	return item
}

func deviceIdentitiesToJSON(devices []coreid.DeviceIdentity) []map[string]any {
	resp := make([]map[string]any, 0, len(devices))
	for _, device := range devices {
		resp = append(resp, deviceIdentityToJSON(device))
	}
	return resp
}

func devicePrekeysToJSON(prekeys []coreid.DevicePrekey) []map[string]any {
	resp := make([]map[string]any, 0, len(prekeys))
	for _, prekey := range prekeys {
		item := map[string]any{
			"id":                 prekey.ID,
			"device_identity_id": prekey.DeviceIdentityID,
			"prekey_id":          prekey.PrekeyID,
			"public_key":         prekey.PublicKey,
			"state":              prekey.State,
			"created_at":         prekey.CreatedAt,
		}
		if prekey.RevokedAt != nil {
			item["revoked_at"] = *prekey.RevokedAt
		}
		resp = append(resp, item)
	}
	return resp
}

func deviceDirectoryToJSON(entries []coreid.DeviceDirectoryEntry) []map[string]any {
	resp := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		item := deviceIdentityToJSON(entry.DeviceIdentity)
		item["prekeys"] = devicePrekeysToJSON(entry.Prekeys)
		resp = append(resp, item)
	}
	return resp
}
