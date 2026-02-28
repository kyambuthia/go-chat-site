package config

import (
	"net/http/httptest"
	"testing"
)

func TestWSAllowedOriginsFromEnv_DefaultsWhenUnsetOrEmpty(t *testing.T) {
	t.Setenv(EnvWSAllowedOrigins, "")
	got := WSAllowedOriginsFromEnv()
	want := DefaultWSAllowedOrigins()
	if len(got) != len(want) {
		t.Fatalf("len(default origins) = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("default origin[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestWSAllowedOriginsFromEnv_NormalizesAndTrims(t *testing.T) {
	t.Setenv(EnvWSAllowedOrigins, " HTTPS://Example.COM , http://LOCALHOST:8080 ,,  ")
	got := WSAllowedOriginsFromEnv()

	if len(got) != 2 {
		t.Fatalf("len(origins) = %d, want 2", len(got))
	}
	if got[0] != "https://example.com" {
		t.Fatalf("origin[0] = %q, want https://example.com", got[0])
	}
	if got[1] != "http://localhost:8080" {
		t.Fatalf("origin[1] = %q, want http://localhost:8080", got[1])
	}
}

func TestWSOriginAllowed(t *testing.T) {
	allowed := []string{"https://example.com", "http://localhost:3000"}

	tests := []struct {
		name   string
		origin string
		ok     bool
	}{
		{name: "exact match", origin: "https://example.com", ok: true},
		{name: "case normalized", origin: "HTTPS://EXAMPLE.COM", ok: true},
		{name: "path ignored by host normalization", origin: "https://example.com/path", ok: true},
		{name: "port mismatch", origin: "https://example.com:443", ok: false},
		{name: "not in allowlist", origin: "https://evil.example", ok: false},
		{name: "malformed", origin: "://bad", ok: false},
		{name: "empty allowed for non-browser", origin: "", ok: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := WSOriginAllowed(tc.origin, allowed); got != tc.ok {
				t.Fatalf("WSOriginAllowed(%q) = %v, want %v", tc.origin, got, tc.ok)
			}
		})
	}
}

func TestWSOriginCheckFunc_UsesSnapshotOfEnv(t *testing.T) {
	t.Setenv(EnvWSAllowedOrigins, "https://one.example")
	check := WSOriginCheckFunc()

	t.Setenv(EnvWSAllowedOrigins, "https://two.example")

	reqOne := httptest.NewRequest("GET", "http://localhost/ws", nil)
	reqOne.Header.Set("Origin", "https://one.example")
	if !check(reqOne) {
		t.Fatal("expected one.example to be allowed by captured checker")
	}

	reqTwo := httptest.NewRequest("GET", "http://localhost/ws", nil)
	reqTwo.Header.Set("Origin", "https://two.example")
	if check(reqTwo) {
		t.Fatal("expected two.example to be denied by captured checker")
	}
}

func TestIntFromEnv_UsesFallbackForInvalidValues(t *testing.T) {
	t.Setenv(EnvLoginRateLimitPerMinute, "not-a-number")
	if got := LoginRateLimitPerMinute(); got != 60 {
		t.Fatalf("LoginRateLimitPerMinute invalid = %d, want 60", got)
	}

	t.Setenv(EnvWSRateLimitPerMinute, "0")
	if got := WSHandshakeRateLimitPerMinute(); got != 120 {
		t.Fatalf("WSHandshakeRateLimitPerMinute zero = %d, want 120", got)
	}

	t.Setenv(EnvWSRateLimitPerMinute, "240")
	if got := WSHandshakeRateLimitPerMinute(); got != 240 {
		t.Fatalf("WSHandshakeRateLimitPerMinute = %d, want 240", got)
	}
}
