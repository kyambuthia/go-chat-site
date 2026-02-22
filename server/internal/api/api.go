package api

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/kyambuthia/go-chat-site/server/internal/adapters/httpapi"
	"github.com/kyambuthia/go-chat-site/server/internal/store"
	"github.com/kyambuthia/go-chat-site/server/internal/ws"
)

func NewAPI(dataStore store.APIStore, hub *ws.Hub) http.Handler {
	// Backward-compatible shim while route/handler composition lives in adapters/httpapi.
	return loggingMiddleware(httpapi.NewRouter(dataStore, hub))
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		reqID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if reqID == "" {
			reqID = newRequestID()
		}
		w.Header().Set("X-Request-ID", reqID)

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		log.Printf("request_id=%s method=%s path=%s status=%d duration_ms=%d", reqID, r.Method, r.URL.Path, rec.status, time.Since(start).Milliseconds())
	})
}

func newRequestID() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(b[:])
}
