package config

import (
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const (
	EnvWSAllowedOrigins        = "WS_ALLOWED_ORIGINS"
	EnvLoginRateLimitPerMinute = "LOGIN_RATE_LIMIT_PER_MINUTE"
	EnvWSRateLimitPerMinute    = "WS_HANDSHAKE_RATE_LIMIT_PER_MINUTE"
)

func DefaultWSAllowedOrigins() []string {
	return []string{
		"http://localhost",
		"https://localhost",
		"http://127.0.0.1",
		"https://127.0.0.1",
		"http://[::1]",
		"https://[::1]",
		"http://localhost:3000",
		"http://localhost:5173",
		"http://127.0.0.1:3000",
		"http://127.0.0.1:5173",
	}
}

func WSAllowedOriginsFromEnv() []string {
	raw := strings.TrimSpace(os.Getenv(EnvWSAllowedOrigins))
	if raw == "" {
		return DefaultWSAllowedOrigins()
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, strings.ToLower(v))
		}
	}
	if len(out) == 0 {
		return DefaultWSAllowedOrigins()
	}
	return out
}

func WSOriginAllowed(origin string, allowed []string) bool {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		// Allow non-browser clients and local tooling that omit Origin.
		return true
	}
	u, err := url.Parse(origin)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}
	normalized := strings.ToLower(u.Scheme + "://" + u.Host)
	for _, candidate := range allowed {
		if normalized == strings.ToLower(strings.TrimSpace(candidate)) {
			return true
		}
	}
	return false
}

func WSOriginCheckFunc() func(r *http.Request) bool {
	allowed := WSAllowedOriginsFromEnv()
	return func(r *http.Request) bool {
		return WSOriginAllowed(r.Header.Get("Origin"), allowed)
	}
}

func loginRateLimitPerMinute() int {
	return intFromEnv(EnvLoginRateLimitPerMinute, 60)
}

func wsHandshakeRateLimitPerMinute() int {
	return intFromEnv(EnvWSRateLimitPerMinute, 120)
}

func LoginRateLimitPerMinute() int       { return loginRateLimitPerMinute() }
func WSHandshakeRateLimitPerMinute() int { return wsHandshakeRateLimitPerMinute() }

func intFromEnv(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}
