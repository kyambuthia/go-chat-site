package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kyambuthia/go-chat-site/server/internal/auth"
	coreid "github.com/kyambuthia/go-chat-site/server/internal/core/identity"
)

type fakeIdentityProfileService struct {
	profile        coreid.Profile
	updatedProfile coreid.Profile
	err            error
	lastUserID     coreid.UserID
	lastUpdate     coreid.ProfileUpdate
}

func (f *fakeIdentityProfileService) GetProfile(ctx context.Context, userID coreid.UserID) (coreid.Profile, error) {
	_ = ctx
	f.lastUserID = userID
	return f.profile, f.err
}

func (f *fakeIdentityProfileService) UpdateProfile(ctx context.Context, userID coreid.UserID, update coreid.ProfileUpdate) (coreid.Profile, error) {
	_ = ctx
	f.lastUserID = userID
	f.lastUpdate = update
	if f.updatedProfile.Username != "" {
		return f.updatedProfile, f.err
	}
	return f.profile, f.err
}

func TestMeHandler_GetMe_UsesIdentityServiceAndPreservesResponseShape(t *testing.T) {
	svc := &fakeIdentityProfileService{profile: coreid.Profile{
		UserID:      4,
		Username:    "alice",
		DisplayName: "Alice",
		AvatarURL:   "https://example.com/a.png",
	}}
	h := &MeHandler{Identity: svc}

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req = req.WithContext(auth.WithUserID(req.Context(), 4))
	rr := httptest.NewRecorder()

	h.GetMe(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if svc.lastUserID != 4 {
		t.Fatalf("GetProfile userID = %d, want 4", svc.lastUserID)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got := int(resp["id"].(float64)); got != 4 {
		t.Fatalf("id = %d, want 4", got)
	}
	if got := resp["username"].(string); got != "alice" {
		t.Fatalf("username = %q, want alice", got)
	}
}

func TestMeHandler_GetMe_MapsProfileErrorToNotFound(t *testing.T) {
	svc := &fakeIdentityProfileService{err: errors.New("not found")}
	h := &MeHandler{Identity: svc}

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req = req.WithContext(auth.WithUserID(req.Context(), 4))
	rr := httptest.NewRecorder()

	h.GetMe(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
}

func TestMeHandler_UpdateMe_PartiallyUpdatesProfileAndPreservesResponseShape(t *testing.T) {
	svc := &fakeIdentityProfileService{profile: coreid.Profile{
		UserID:      4,
		Username:    "alice",
		DisplayName: "Alice",
		AvatarURL:   "https://example.com/a.png",
	}, updatedProfile: coreid.Profile{
		UserID:      4,
		Username:    "alice",
		DisplayName: "Alice Doe",
		AvatarURL:   "https://example.com/alice.png",
	}}
	h := &MeHandler{Identity: svc}

	req := httptest.NewRequest(http.MethodPatch, "/api/me", strings.NewReader(`{"display_name":"Alice Doe"}`))
	req = req.WithContext(auth.WithUserID(req.Context(), 4))
	rr := httptest.NewRecorder()

	h.UpdateMe(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", rr.Code, rr.Body.String())
	}
	if svc.lastUpdate.DisplayName != "Alice Doe" {
		t.Fatalf("display_name = %q, want Alice Doe", svc.lastUpdate.DisplayName)
	}
	if svc.lastUpdate.AvatarURL != "https://example.com/a.png" {
		t.Fatalf("avatar_url = %q, want existing avatar", svc.lastUpdate.AvatarURL)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got := resp["display_name"].(string); got != "Alice Doe" {
		t.Fatalf("display_name = %q, want Alice Doe", got)
	}
	if got := resp["avatar_url"].(string); got != "https://example.com/alice.png" {
		t.Fatalf("avatar_url = %q, want updated avatar", got)
	}
}

func TestMeHandler_UpdateMe_RequiresAtLeastOneField(t *testing.T) {
	svc := &fakeIdentityProfileService{}
	h := &MeHandler{Identity: svc}

	req := httptest.NewRequest(http.MethodPatch, "/api/me", strings.NewReader(`{}`))
	req = req.WithContext(auth.WithUserID(req.Context(), 4))
	rr := httptest.NewRecorder()

	h.UpdateMe(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}
