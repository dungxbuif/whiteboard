package main

import (
	"strings"
	"testing"
)

func TestLoginJWTSecretMustContainAtLeast32Bytes(t *testing.T) {
	setValidConfig(t)
	for _, value := range []string{"", "short", strings.Repeat("x", 31)} {
		t.Setenv("JWT_SECRET", value)
		if _, err := loadConfig(); err == nil {
			t.Errorf("JWT_SECRET with %d bytes was accepted", len(value))
		}
	}
	t.Setenv("JWT_SECRET", strings.Repeat("x", 32))
	if _, err := loadConfig(); err != nil {
		t.Fatalf("32-byte JWT_SECRET rejected: %v", err)
	}
}

func setValidConfig(t *testing.T) {
	t.Helper()
	for _, key := range []string{"DATABASE_URL", "CASBIN_MODEL_PATH", "S3_ENDPOINT", "S3_ACCESS_KEY", "S3_SECRET_KEY", "S3_BUCKET"} {
		t.Setenv(key, "fixture")
	}
	t.Setenv("FRONTEND_URL", "https://example.test")
	t.Setenv("JWT_SECRET", "login-0123456789abcdef0123456789abcdef")
	t.Setenv("COLLAB_JWT_SECRET", "collab-0123456789abcdef0123456789abcdef")
	t.Setenv("AUTHORITY_SERVICE_SECRET", "service-0123456789abcdef0123456789abcdef")
	t.Setenv("MUTATION_GRANT_SECRET", "grant-0123456789abcdef0123456789abcdef")
}

func TestInvitationFrontendOriginConfig(t *testing.T) {
	t.Setenv("AUTHORITY_SERVICE_SECRET", "service-0123456789abcdef0123456789abcdef")
	t.Setenv("MUTATION_GRANT_SECRET", "grant-0123456789abcdef0123456789abcdef")
	t.Setenv("COLLAB_JWT_SECRET", "collab-0123456789abcdef0123456789abcdef")
	for _, key := range []string{"DATABASE_URL", "CASBIN_MODEL_PATH", "S3_ENDPOINT", "S3_ACCESS_KEY", "S3_SECRET_KEY", "S3_BUCKET"} {
		t.Setenv(key, "fixture")
	}
	t.Setenv("JWT_SECRET", "login-0123456789abcdef0123456789abcdef")
	for _, value := range []string{"/relative", "ftp://example.test", "https://example.test/path", "https://example.test?next=x", "https://example.test#fragment", "https://user:pass@example.test"} {
		t.Setenv("FRONTEND_URL", value)
		if _, err := loadConfig(); err == nil {
			t.Errorf("unsafe frontend origin accepted: %s", value)
		}
	}
	for _, value := range []string{"https://example.test", "http://localhost:5173/"} {
		t.Setenv("FRONTEND_URL", value)
		if _, err := loadConfig(); err != nil {
			t.Errorf("valid origin rejected: %v", err)
		}
	}
}

func TestRoomTokenSecretConfig(t *testing.T) {
	t.Setenv("AUTHORITY_SERVICE_SECRET", "service-0123456789abcdef0123456789abcdef")
	t.Setenv("MUTATION_GRANT_SECRET", "grant-0123456789abcdef0123456789abcdef")
	for _, key := range []string{"DATABASE_URL", "CASBIN_MODEL_PATH", "S3_ENDPOINT", "S3_ACCESS_KEY", "S3_SECRET_KEY", "S3_BUCKET"} {
		t.Setenv(key, "fixture")
	}
	t.Setenv("FRONTEND_URL", "https://example.test")
	t.Setenv("JWT_SECRET", "login-0123456789abcdef0123456789abcdef")
	for _, value := range []string{"", "short", "login-0123456789abcdef0123456789abcdef"} {
		t.Setenv("COLLAB_JWT_SECRET", value)
		if _, e := loadConfig(); e == nil {
			t.Errorf("invalid room signing configuration accepted")
		}
	}
	t.Setenv("COLLAB_JWT_SECRET", "collab-0123456789abcdef0123456789abcdef")
	if _, e := loadConfig(); e != nil {
		t.Fatal(e)
	}
}

func TestAuthorityConfigFailClosed(t *testing.T) {
	for _, key := range []string{"DATABASE_URL", "CASBIN_MODEL_PATH", "S3_ENDPOINT", "S3_ACCESS_KEY", "S3_SECRET_KEY", "S3_BUCKET"} {
		t.Setenv(key, "fixture")
	}
	t.Setenv("FRONTEND_URL", "https://example.test")
	t.Setenv("JWT_SECRET", "login-0123456789abcdef0123456789abcdef")
	t.Setenv("COLLAB_JWT_SECRET", "collab-0123456789abcdef0123456789abcdef")
	t.Setenv("AUTHORITY_SERVICE_SECRET", "")
	t.Setenv("MUTATION_GRANT_SECRET", "")
	if _, e := loadConfig(); e == nil {
		t.Fatal("missing authority secrets accepted")
	}
}

func TestAuthorityListenerAndSecretIsolation(t *testing.T) {
	for _, key := range []string{"DATABASE_URL", "CASBIN_MODEL_PATH", "S3_ENDPOINT", "S3_ACCESS_KEY", "S3_SECRET_KEY", "S3_BUCKET"} {
		t.Setenv(key, "fixture")
	}
	t.Setenv("FRONTEND_URL", "https://example.test")
	t.Setenv("JWT_SECRET", "login-0123456789abcdef0123456789abcdef")
	t.Setenv("COLLAB_JWT_SECRET", "collab-0123456789abcdef0123456789abcdef")
	t.Setenv("AUTHORITY_SERVICE_SECRET", "service-0123456789abcdef0123456789abcdef")
	t.Setenv("MUTATION_GRANT_SECRET", "grant-0123456789abcdef0123456789abcdef")
	for _, addr := range []string{":8081", "0.0.0.0:8081", "8.8.8.8:8081", "localhost:8081", "127.0.0.1:0"} {
		t.Setenv("AUTHORITY_LISTEN_ADDR", addr)
		if _, e := loadConfig(); e == nil {
			t.Errorf("unsafe bind %s", addr)
		}
	}
	t.Setenv("AUTHORITY_LISTEN_ADDR", "127.0.0.1:8081")
	if _, e := loadConfig(); e != nil {
		t.Fatal(e)
	}
	t.Setenv("MUTATION_GRANT_SECRET", "service-0123456789abcdef0123456789abcdef")
	if _, e := loadConfig(); e == nil {
		t.Fatal("reused service secret accepted")
	}
}
