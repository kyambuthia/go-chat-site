package httpapi

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kyambuthia/go-chat-site/server/internal/adapters/transport/wsrelay"
	"github.com/kyambuthia/go-chat-site/server/internal/auth"
	"github.com/kyambuthia/go-chat-site/server/internal/migrate"
	"github.com/kyambuthia/go-chat-site/server/internal/store"
)

type noSQLDBStore struct {
	store.APIStore
}

func setupRouterStore(t *testing.T) *store.SqliteStore {
	t.Helper()
	s, err := store.NewSqliteStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.DB.Close() })
	if err := migrate.RunMigrations(s.DB, filepath.Join("..", "..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestNewRouter_HealthEndpoints(t *testing.T) {
	if err := auth.ConfigureJWT("router-health-test-secret"); err != nil {
		t.Fatal(err)
	}
	s := setupRouterStore(t)
	hub := wsrelay.NewHub()
	go hub.Run()
	defer hub.Shutdown()

	r := NewRouter(s, hub)

	t.Run("healthz returns ok", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rr.Code)
		}
		if !strings.Contains(rr.Body.String(), `"status":"ok"`) {
			t.Fatalf("unexpected body: %s", rr.Body.String())
		}
	})

	t.Run("readyz returns ready when db is healthy", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 body=%s", rr.Code, rr.Body.String())
		}
		if !strings.Contains(rr.Body.String(), `"status":"ready"`) {
			t.Fatalf("unexpected body: %s", rr.Body.String())
		}
	})
}

func TestNewRouter_ReadyzFailsWithoutSQLDBProvider(t *testing.T) {
	if err := auth.ConfigureJWT("router-readyz-test-secret"); err != nil {
		t.Fatal(err)
	}
	s := setupRouterStore(t)
	wrapped := noSQLDBStore{APIStore: s}

	hub := wsrelay.NewHub()
	go hub.Run()
	defer hub.Shutdown()

	r := NewRouter(wrapped, hub)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"status":"not_ready"`) {
		t.Fatalf("unexpected body: %s", rr.Body.String())
	}
}

func TestNewRouter_CompatibilityRouteStillPresent(t *testing.T) {
	if err := auth.ConfigureJWT("router-routes-test-secret"); err != nil {
		t.Fatal(err)
	}
	s := setupRouterStore(t)
	hub := wsrelay.NewHub()
	go hub.Run()
	defer hub.Shutdown()

	r := NewRouter(s, hub)

	req := httptest.NewRequest(http.MethodGet, "/api/register", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405 body=%s", rr.Code, rr.Body.String())
	}
}
