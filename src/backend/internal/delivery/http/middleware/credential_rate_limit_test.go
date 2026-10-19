package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

func testCredentialLimiter(t *testing.T, cfg CredentialRateLimitConfig) *CredentialRateLimiter {
	t.Helper()
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	limiter, err := newCredentialRateLimiter(cfg, []byte("test-rate-limit-key-012345678901"), func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	return limiter
}

func observedClientIP(t *testing.T, remoteAddr, realIP string) string {
	t.Helper()
	observed := ""
	handler := CapturePeerIP(chimiddleware.RealIP(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		observed = credentialClientIP(r)
	})))
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.RemoteAddr = remoteAddr
	if realIP != "" {
		req.Header.Set("X-Real-IP", realIP)
	}
	handler.ServeHTTP(httptest.NewRecorder(), req)
	return observed
}

func TestCredentialClientIPTrustsSingleRealIPFromPrivateOrLoopbackPeer(t *testing.T) {
	for _, tc := range []struct {
		peer, header, want string
	}{
		{"172.18.0.4:54321", "203.0.113.42", "203.0.113.42"},
		{"127.0.0.1:54321", "2001:db8::42", "2001:db8::42"},
		{"[fd00::5]:54321", "198.51.100.7", "198.51.100.7"},
	} {
		if got := observedClientIP(t, tc.peer, tc.header); got != tc.want {
			t.Errorf("peer=%q header=%q: got %q, want %q", tc.peer, tc.header, got, tc.want)
		}
	}
}

func TestCredentialClientIPIgnoresSpoofedRealIPFromPublicPeer(t *testing.T) {
	if got := observedClientIP(t, "198.51.100.9:54321", "203.0.113.99"); got != "198.51.100.9" {
		t.Fatalf("spoofed public-peer header resolved to %q", got)
	}
}

func TestCredentialClientIPRejectsMalformedOrMultipleRealIPValues(t *testing.T) {
	for _, header := range []string{"not-an-ip", "203.0.113.1, 203.0.113.2", "203.0.113.1 203.0.113.2"} {
		if got := observedClientIP(t, "172.18.0.4:54321", header); got != "172.18.0.4" {
			t.Errorf("header=%q resolved to %q", header, got)
		}
	}
	observed := ""
	handler := CapturePeerIP(chimiddleware.RealIP(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		observed = credentialClientIP(r)
	})))
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.RemoteAddr = "172.18.0.4:54321"
	req.Header.Add("X-Real-IP", "203.0.113.1")
	req.Header.Add("X-Real-IP", "203.0.113.2")
	handler.ServeHTTP(httptest.NewRecorder(), req)
	if observed != "172.18.0.4" {
		t.Fatalf("duplicate header lines resolved to %q", observed)
	}
}

func requestFrom(ip string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`))
	r.RemoteAddr = ip + ":43210"
	return r
}

func TestCredentialRateLimiterEnforcesBothIPAndIdentityLimits(t *testing.T) {
	limiter := testCredentialLimiter(t, CredentialRateLimitConfig{
		Window: time.Minute, MaxEntries: 20,
		Login: CredentialRateLimitPolicy{PerIP: 2, PerIdentity: 1},
	})

	if allowed, _ := limiter.Allow(CredentialRouteLogin, requestFrom("192.0.2.1"), " User@Example.Test "); !allowed {
		t.Fatal("first attempt rejected")
	}
	if allowed, _ := limiter.Allow(CredentialRouteLogin, requestFrom("192.0.2.2"), "user@example.test"); allowed {
		t.Fatal("normalized account limit was bypassed from a second IP")
	}
	if allowed, _ := limiter.Allow(CredentialRouteLogin, requestFrom("192.0.2.1"), "other@example.test"); !allowed {
		t.Fatal("second attempt from IP rejected too early")
	}
	if allowed, retry := limiter.Allow(CredentialRouteLogin, requestFrom("192.0.2.1"), "third@example.test"); allowed || retry <= 0 {
		t.Fatalf("IP limit result = (%v, %s), want rejection with retry", allowed, retry)
	}
}

func TestCredentialRateLimiterSeparatesRoutesAndBoundsState(t *testing.T) {
	limiter := testCredentialLimiter(t, CredentialRateLimitConfig{
		Window: time.Minute, MaxEntries: 4,
		Login:       CredentialRateLimitPolicy{PerIP: 10, PerIdentity: 10},
		Invitation:  CredentialRateLimitPolicy{PerIP: 10, PerIdentity: 10},
		PublicScene: CredentialRateLimitPolicy{PerIP: 10, PerIdentity: 10},
	})
	if allowed, _ := limiter.Allow(CredentialRouteLogin, requestFrom("192.0.2.1"), "one@example.test"); !allowed {
		t.Fatal("first login rejected")
	}
	if allowed, _ := limiter.Allow(CredentialRouteInvitation, requestFrom("192.0.2.2"), "invite-token"); !allowed {
		t.Fatal("invitation route was not independent")
	}
	if allowed, _ := limiter.Allow(CredentialRoutePublicScene, requestFrom("192.0.2.3"), "share-token"); allowed {
		t.Fatal("new attacker-controlled keys were accepted after the hard cap")
	}
	if got := limiter.entryCount(); got > 4 {
		t.Fatalf("entry count = %d, exceeds cap", got)
	}
}

func TestEnforceCredentialRateLimitReturnsControlled429(t *testing.T) {
	limiter := testCredentialLimiter(t, CredentialRateLimitConfig{
		Window: time.Minute, MaxEntries: 20,
		Login: CredentialRateLimitPolicy{PerIP: 1, PerIdentity: 1},
	})
	req := requestFrom("192.0.2.10")
	if !EnforceCredentialRateLimit(httptest.NewRecorder(), req, limiter, CredentialRouteLogin, "user@example.test") {
		t.Fatal("first request rejected")
	}
	rr := httptest.NewRecorder()
	if EnforceCredentialRateLimit(rr, req, limiter, CredentialRouteLogin, "user@example.test") {
		t.Fatal("limited request was allowed")
	}
	if rr.Code != http.StatusTooManyRequests || rr.Header().Get("Retry-After") == "" {
		t.Fatalf("status=%d retry-after=%q", rr.Code, rr.Header().Get("Retry-After"))
	}
	if !strings.Contains(rr.Body.String(), `"code":"RATE_LIMITED"`) {
		t.Fatalf("unexpected response: %s", rr.Body.String())
	}
}

func TestCredentialRateLimiterExpiresEntriesBeforeReusingCapacity(t *testing.T) {
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	limiter, err := newCredentialRateLimiter(CredentialRateLimitConfig{
		Window: time.Minute, MaxEntries: 2,
		Login: CredentialRateLimitPolicy{PerIP: 1, PerIdentity: 1},
	}, []byte("test-rate-limit-key-012345678901"), func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	if allowed, _ := limiter.Allow(CredentialRouteLogin, requestFrom("192.0.2.1"), "one@example.test"); !allowed {
		t.Fatal("initial request rejected")
	}
	now = now.Add(time.Minute + time.Second)
	if allowed, _ := limiter.Allow(CredentialRouteLogin, requestFrom("192.0.2.2"), "two@example.test"); !allowed {
		t.Fatal("expired entries did not release bounded capacity")
	}
	if got := limiter.entryCount(); got != 2 {
		t.Fatalf("entry count=%d, want 2", got)
	}
}
