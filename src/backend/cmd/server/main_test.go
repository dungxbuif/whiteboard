package main

import (
	"testing"
)

func TestInvitationFrontendOriginConfig(t *testing.T) {
	t.Setenv("COLLAB_JWT_SECRET", "collab-0123456789abcdef0123456789abcdef")
	for _, key := range []string{"DATABASE_URL", "JWT_SECRET", "CASBIN_MODEL_PATH", "S3_ENDPOINT", "S3_ACCESS_KEY", "S3_SECRET_KEY", "S3_BUCKET"} {
		t.Setenv(key, "fixture")
	}
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
