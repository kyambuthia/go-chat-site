package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kyambuthia/go-chat-site/server/internal/auth"
	coreid "github.com/kyambuthia/go-chat-site/server/internal/core/identity"
)

type fakeIdentityProfileService struct {
	profile    coreid.Profile
	err        error
	lastUserID coreid.UserID
}

func (f *fakeIdentityProfileService) GetProfile(ctx context.Context, userID coreid.UserID) (coreid.Profile, error) {
	_ = ctx
	f.lastUserID = userID
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
