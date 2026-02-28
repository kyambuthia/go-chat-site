package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFixedWindowRateLimiter_AllowAndWindowReset(t *testing.T) {
	limiter := newFixedWindowRateLimiter(2, 15*time.Millisecond)

	if !limiter.allow("1.2.3.4") {
		t.Fatal("first request should pass")
	}
	if !limiter.allow("1.2.3.4") {
		t.Fatal("second request should pass")
	}
	if limiter.allow("1.2.3.4") {
		t.Fatal("third request should be rate limited")
	}

	time.Sleep(20 * time.Millisecond)

	if !limiter.allow("1.2.3.4") {
		t.Fatal("request after window reset should pass")
	}
}

func TestRateLimitMiddleware_ReturnsJSON429(t *testing.T) {
	limiter := newFixedWindowRateLimiter(1, time.Minute)
	called := 0
	h := rateLimitMiddleware(limiter)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called++
		w.WriteHeader(http.StatusNoContent)
	}))

	mkReq := func() *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/api/login", nil)
		req.RemoteAddr = "10.0.0.7:1234"
		return req
	}

	rr1 := httptest.NewRecorder()
	h.ServeHTTP(rr1, mkReq())
	if rr1.Code != http.StatusNoContent {
		t.Fatalf("first status = %d, want 204", rr1.Code)
	}

	rr2 := httptest.NewRecorder()
	h.ServeHTTP(rr2, mkReq())
	if rr2.Code != http.StatusTooManyRequests {
		t.Fatalf("second status = %d, want 429", rr2.Code)
	}
	if got := rr2.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content type = %q, want application/json", got)
	}

	var body map[string]string
	if err := json.Unmarshal(rr2.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if body["error"] != "rate limit exceeded" {
		t.Fatalf("error body = %q, want rate limit exceeded", body["error"])
	}
	if called != 1 {
		t.Fatalf("next handler call count = %d, want 1", called)
	}
}

func TestClientIP_PrefersFirstForwardedFor(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.7, 10.0.0.1")
	req.RemoteAddr = "10.1.1.1:4321"

	if got := clientIP(req); got != "203.0.113.7" {
		t.Fatalf("clientIP = %q, want 203.0.113.7", got)
	}
}

func TestClientIP_FallbacksToRemoteAddrHostThenUnknown(t *testing.T) {
	reqHostPort := httptest.NewRequest(http.MethodGet, "/", nil)
	reqHostPort.RemoteAddr = "192.0.2.8:8080"
	if got := clientIP(reqHostPort); got != "192.0.2.8" {
		t.Fatalf("clientIP(host:port) = %q, want 192.0.2.8", got)
	}

	reqRaw := httptest.NewRequest(http.MethodGet, "/", nil)
	reqRaw.RemoteAddr = "not-a-host-port"
	if got := clientIP(reqRaw); got != "not-a-host-port" {
		t.Fatalf("clientIP(raw) = %q, want not-a-host-port", got)
	}

	reqUnknown := httptest.NewRequest(http.MethodGet, "/", nil)
	reqUnknown.RemoteAddr = ""
	if got := clientIP(reqUnknown); got != "unknown" {
		t.Fatalf("clientIP(empty) = %q, want unknown", got)
	}
}
