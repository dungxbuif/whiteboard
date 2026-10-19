package middleware

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

type CredentialRoute string

const (
	CredentialRouteLogin       CredentialRoute = "login"
	CredentialRouteInvitation  CredentialRoute = "invitation-registration"
	CredentialRoutePublicScene CredentialRoute = "public-scene-session"
)

type CredentialRateLimitPolicy struct {
	PerIP       int
	PerIdentity int
}

type CredentialRateLimitConfig struct {
	Window      time.Duration
	MaxEntries  int
	Login       CredentialRateLimitPolicy
	Invitation  CredentialRateLimitPolicy
	PublicScene CredentialRateLimitPolicy
}

type credentialRateEntry struct {
	count     int
	expiresAt time.Time
}

type CredentialRateLimiter struct {
	mu        sync.Mutex
	config    CredentialRateLimitConfig
	digestKey []byte
	now       func() time.Time
	entries   map[string]credentialRateEntry
}

func NewDefaultCredentialRateLimiter() (*CredentialRateLimiter, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	return newCredentialRateLimiter(CredentialRateLimitConfig{
		Window:      5 * time.Minute,
		MaxEntries:  20_000,
		Login:       CredentialRateLimitPolicy{PerIP: 30, PerIdentity: 10},
		Invitation:  CredentialRateLimitPolicy{PerIP: 20, PerIdentity: 5},
		PublicScene: CredentialRateLimitPolicy{PerIP: 60, PerIdentity: 15},
	}, key, time.Now)
}

func newCredentialRateLimiter(config CredentialRateLimitConfig, digestKey []byte, now func() time.Time) (*CredentialRateLimiter, error) {
	if config.Window <= 0 || config.MaxEntries < 2 || len(digestKey) < 16 || now == nil {
		return nil, errors.New("invalid credential rate-limit configuration")
	}
	return &CredentialRateLimiter{
		config:    config,
		digestKey: append([]byte(nil), digestKey...),
		now:       now,
		entries:   make(map[string]credentialRateEntry),
	}, nil
}

func (l *CredentialRateLimiter) policy(route CredentialRoute) CredentialRateLimitPolicy {
	switch route {
	case CredentialRouteLogin:
		return l.config.Login
	case CredentialRouteInvitation:
		return l.config.Invitation
	case CredentialRoutePublicScene:
		return l.config.PublicScene
	default:
		return CredentialRateLimitPolicy{}
	}
}

func (l *CredentialRateLimiter) digest(route CredentialRoute, axis, value string) string {
	mac := hmac.New(sha256.New, l.digestKey)
	_, _ = mac.Write([]byte(string(route)))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write([]byte(axis))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}

func credentialClientIP(r *http.Request) string {
	value := strings.TrimSpace(r.RemoteAddr)
	if host, _, err := net.SplitHostPort(value); err == nil {
		value = host
	}
	if parsed := net.ParseIP(strings.Trim(value, "[]")); parsed != nil {
		return parsed.String()
	}
	return "unknown"
}

func normalizeCredentialIdentity(route CredentialRoute, identity string) string {
	identity = strings.TrimSpace(identity)
	if route == CredentialRouteLogin {
		identity = strings.ToLower(identity)
	}
	if identity == "" {
		return "unknown"
	}
	return identity
}

func (l *CredentialRateLimiter) Allow(route CredentialRoute, r *http.Request, identity string) (bool, time.Duration) {
	if l == nil {
		return false, time.Minute
	}
	policy := l.policy(route)
	if policy.PerIP < 1 || policy.PerIdentity < 1 {
		return false, l.config.Window
	}
	now := l.now()
	keys := []struct {
		key   string
		limit int
	}{
		{l.digest(route, "ip", credentialClientIP(r)), policy.PerIP},
		{l.digest(route, "identity", normalizeCredentialIdentity(route, identity)), policy.PerIdentity},
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	missing := 0
	for _, item := range keys {
		entry, exists := l.entries[item.key]
		if exists && !now.Before(entry.expiresAt) {
			delete(l.entries, item.key)
			exists = false
		}
		if !exists {
			missing++
		}
	}
	if len(l.entries)+missing > l.config.MaxEntries {
		for key, entry := range l.entries {
			if !now.Before(entry.expiresAt) {
				delete(l.entries, key)
			}
		}
		if len(l.entries)+missing > l.config.MaxEntries {
			return false, l.config.Window
		}
	}

	var retry time.Duration
	for _, item := range keys {
		entry, exists := l.entries[item.key]
		if exists && entry.count >= item.limit {
			remaining := entry.expiresAt.Sub(now)
			if remaining > retry {
				retry = remaining
			}
		}
	}
	if retry > 0 {
		return false, retry
	}
	for _, item := range keys {
		entry, exists := l.entries[item.key]
		if !exists {
			entry.expiresAt = now.Add(l.config.Window)
		}
		entry.count++
		l.entries[item.key] = entry
	}
	return true, 0
}

func (l *CredentialRateLimiter) entryCount() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.entries)
}

func EnforceCredentialRateLimit(w http.ResponseWriter, r *http.Request, limiter *CredentialRateLimiter, route CredentialRoute, identity string) bool {
	allowed, retry := limiter.Allow(route, r, identity)
	if allowed {
		return true
	}
	seconds := int(math.Ceil(retry.Seconds()))
	if seconds < 1 {
		seconds = 1
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", strconv.Itoa(seconds))
	w.WriteHeader(http.StatusTooManyRequests)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"code": "RATE_LIMITED", "message": "Too many attempts; try again later", "requestId": chimiddleware.GetReqID(r.Context()),
	})
	return false
}
