package config

import (
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	EnvWSAllowedOrigins         = "WS_ALLOWED_ORIGINS"
	EnvLoginRateLimitPerMinute  = "LOGIN_RATE_LIMIT_PER_MINUTE"
	EnvLoginUserRateLimit       = "LOGIN_USER_RATE_LIMIT_PER_MINUTE"
	EnvRefreshRateLimitPerMin   = "REFRESH_RATE_LIMIT_PER_MINUTE"
	EnvWSRateLimitPerMinute     = "WS_HANDSHAKE_RATE_LIMIT_PER_MINUTE"
	EnvAccessTokenTTLMinutes    = "ACCESS_TOKEN_TTL_MINUTES"
	EnvRefreshTokenTTLHours     = "REFRESH_TOKEN_TTL_HOURS"
	EnvLoginLockoutThreshold    = "LOGIN_LOCKOUT_THRESHOLD"
	EnvLoginLockoutWindowMins   = "LOGIN_LOCKOUT_WINDOW_MINUTES"
	EnvLoginLockoutDurationMins = "LOGIN_LOCKOUT_DURATION_MINUTES"
	EnvMessagingStorePlaintext  = "MESSAGING_STORE_PLAINTEXT_WHEN_ENCRYPTED"
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

func loginUserRateLimitPerMinute() int {
	return intFromEnv(EnvLoginUserRateLimit, 20)
}

func refreshRateLimitPerMinute() int {
	return intFromEnv(EnvRefreshRateLimitPerMin, 60)
}

func wsHandshakeRateLimitPerMinute() int {
	return intFromEnv(EnvWSRateLimitPerMinute, 120)
}

func LoginRateLimitPerMinute() int       { return loginRateLimitPerMinute() }
func LoginUserRateLimitPerMinute() int   { return loginUserRateLimitPerMinute() }
func RefreshRateLimitPerMinute() int     { return refreshRateLimitPerMinute() }
func WSHandshakeRateLimitPerMinute() int { return wsHandshakeRateLimitPerMinute() }
func AccessTokenTTL() time.Duration {
	return time.Duration(intFromEnv(EnvAccessTokenTTLMinutes, 15)) * time.Minute
}
func RefreshTokenTTL() time.Duration {
	return time.Duration(intFromEnv(EnvRefreshTokenTTLHours, 24*30)) * time.Hour
}
func LoginLockoutThreshold() int { return intFromEnv(EnvLoginLockoutThreshold, 5) }
func LoginLockoutWindow() time.Duration {
	return time.Duration(intFromEnv(EnvLoginLockoutWindowMins, 15)) * time.Minute
}
func LoginLockoutDuration() time.Duration {
	return time.Duration(intFromEnv(EnvLoginLockoutDurationMins, 15)) * time.Minute
}
func MessagingStorePlaintextWhenEncrypted() bool {
	return boolFromEnv(EnvMessagingStorePlaintext, true)
}

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

func boolFromEnv(key string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return v
}
