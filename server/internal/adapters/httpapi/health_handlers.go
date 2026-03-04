package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/kyambuthia/go-chat-site/server/internal/health"
	"github.com/kyambuthia/go-chat-site/server/internal/store"
)

func healthzHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

func readyzHandler(check func(context.Context) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := check(ctx); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "not_ready",
				"error":  err.Error(),
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
	}
}

func readinessCheck(dataStore store.APIStore) func(context.Context) error {
	dbProvider, ok := dataStore.(interface{ SQLDB() *sql.DB })
	if !ok || dbProvider.SQLDB() == nil {
		return func(context.Context) error { return health.ErrNilDB }
	}

	db := dbProvider.SQLDB()
	return func(ctx context.Context) error {
		return health.CheckDBReady(ctx, db)
	}
}
