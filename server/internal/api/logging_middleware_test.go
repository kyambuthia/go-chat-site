package api

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoggingMiddleware_AssignsAndLogsRequestID(t *testing.T) {
	var logBuf bytes.Buffer
	originalOut := log.Writer()
	log.SetOutput(&logBuf)
	t.Cleanup(func() {
		log.SetOutput(originalOut)
	})

	handler := loggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := w.Header().Get("X-Request-ID"); got == "" {
			t.Fatal("expected X-Request-ID header to be set")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	if got := rr.Header().Get("X-Request-ID"); got == "" {
		t.Fatal("expected response X-Request-ID header")
	}

	if !strings.Contains(logBuf.String(), "request_id=") {
		t.Fatalf("expected request_id in logs, got %q", logBuf.String())
	}
}

func TestLoggingMiddleware_PreservesIncomingRequestID(t *testing.T) {
	handler := loggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := w.Header().Get("X-Request-ID"); got != "req-123" {
			t.Fatalf("X-Request-ID = %q, want req-123", got)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("X-Request-ID", "req-123")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
	}
	if got := rr.Header().Get("X-Request-ID"); got != "req-123" {
		t.Fatalf("response X-Request-ID = %q, want req-123", got)
	}
}
