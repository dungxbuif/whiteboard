package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"devtool/internal/domain"

	"github.com/google/uuid"
)

type ssoHandlerService struct{ binding string }

func (s *ssoHandlerService) Begin(ctx context.Context, _ uuid.UUID, _ string) (domain.AuthorizationRedirect, error) {
	s.binding, _ = ctx.Value(domain.ContextKeyBrowserBinding).(string)
	return domain.AuthorizationRedirect{URL: "https://idp.example/authorize", ExpiresAt: time.Now().Add(time.Minute)}, nil
}
func (s *ssoHandlerService) BeginLink(ctx context.Context, _ domain.Principal, _ uuid.UUID, _ string, _ uuid.UUID) (domain.AuthorizationRedirect, error) {
	s.binding, _ = ctx.Value(domain.ContextKeyBrowserBinding).(string)
	return domain.AuthorizationRedirect{URL: "https://idp.example/authorize", ExpiresAt: time.Now().Add(time.Minute)}, nil
}
func (s *ssoHandlerService) Callback(_ context.Context, _, _, binding string) (domain.OneTimeExchange, error) {
	s.binding = binding
	return domain.OneTimeExchange{Code: "one-time-code", ReturnPath: "/w/demo", ExpiresAt: time.Now().Add(time.Minute)}, nil
}
func (s *ssoHandlerService) Exchange(_ context.Context, _, binding string) (domain.LoginResult, error) {
	s.binding = binding
	return domain.LoginResult{Token: "login-token", User: domain.User{ID: uuid.New(), Email: "user@example.test"}}, nil
}
func (s *ssoHandlerService) Link(context.Context, uuid.UUID, domain.OIDCIdentity, uuid.UUID) error {
	return nil
}
func (s *ssoHandlerService) Configure(context.Context, domain.Principal, uuid.UUID, domain.SSOConnectionInput) (domain.SSOConnection, error) {
	return domain.SSOConnection{}, nil
}
func (s *ssoHandlerService) GetConnection(context.Context, domain.Principal, uuid.UUID) (domain.SSOConnection, error) {
	return domain.SSOConnection{}, nil
}

type ssoHandlerAccess struct{}

func (ssoHandlerAccess) ResolveWorkspaceID(context.Context, string) (uuid.UUID, error) {
	return uuid.New(), nil
}

func TestSSOHandlerStrictBodyAndBrowserBinding(t *testing.T) {
	service := &ssoHandlerService{}
	h := NewSSOHandler(service, ssoHandlerAccess{}, "https://app.example")
	bad := httptest.NewRequest(http.MethodPost, "/api/v1/sso/begin", strings.NewReader(`{"connectionId":"`+uuid.NewString()+`","returnPath":"/","extra":true}`))
	bad.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	h.Begin(response, bad)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("strict body status=%d", response.Code)
	}
	good := httptest.NewRequest(http.MethodPost, "/api/v1/sso/begin", strings.NewReader(`{"connectionId":"`+uuid.NewString()+`","returnPath":"/w/demo"}`))
	response = httptest.NewRecorder()
	h.Begin(response, good)
	if response.Code != http.StatusOK || service.binding == "" {
		t.Fatalf("begin status=%d binding=%q", response.Code, service.binding)
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly {
		t.Fatalf("cookies=%+v", cookies)
	}
}

func TestSSOCallbackReturnsOnlyOpaqueExchangeCode(t *testing.T) {
	service := &ssoHandlerService{}
	h := NewSSOHandler(service, ssoHandlerAccess{}, "https://app.example")
	request := httptest.NewRequest(http.MethodGet, "/api/v1/sso/callback?state=state&code=provider-code", nil)
	request.AddCookie(&http.Cookie{Name: ssoBrowserCookie, Value: "browser-binding-01234567890123456789"})
	response := httptest.NewRecorder()
	h.Callback(response, request)
	if response.Code != http.StatusFound {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	location := response.Header().Get("Location")
	if !strings.Contains(location, "code=one-time-code") || strings.Contains(location, "login-token") || strings.Contains(location, "provider-code") {
		t.Fatalf("unsafe callback location=%s", location)
	}
}

func TestSSOExchangeRejectsUnknownJSON(t *testing.T) {
	h := NewSSOHandler(&ssoHandlerService{}, ssoHandlerAccess{}, "https://app.example")
	request := httptest.NewRequest(http.MethodPost, "/api/v1/sso/exchange", strings.NewReader(`{"code":"one-time-code","token":"leak"}`))
	request.AddCookie(&http.Cookie{Name: ssoBrowserCookie, Value: "browser-binding-01234567890123456789"})
	response := httptest.NewRecorder()
	h.Exchange(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", response.Code)
	}
}

func TestSSOHandlerIdentityPreservesRecoveryWorkspaceScope(t *testing.T) {
	h := NewSSOHandler(&ssoHandlerService{}, ssoHandlerAccess{}, "https://app.example")
	user, session, recoveryWorkspace := uuid.New(), uuid.New(), uuid.New()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/demo/sso", nil)
	ctx := context.WithValue(request.Context(), domain.ContextKeyUserID, user)
	ctx = context.WithValue(ctx, domain.ContextKeySessionID, session)
	ctx = context.WithValue(ctx, domain.ContextKeyAuthMethod, domain.AuthMethodBreakGlass)
	ctx = context.WithValue(ctx, domain.ContextKeyRecoveryWorkspace, recoveryWorkspace)
	principal, _, err := h.identity(request.WithContext(ctx))
	if err != nil {
		t.Fatal(err)
	}
	if principal.SessionID != session || principal.AuthMethod != domain.AuthMethodBreakGlass || principal.RecoveryWorkspaceID != recoveryWorkspace {
		t.Fatalf("principal=%+v", principal)
	}
}
