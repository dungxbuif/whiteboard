package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"devtool/internal/domain"
)

func TestInviteHandlerValidation(t *testing.T) {
	h := NewInvitationHandler(nil)
	for _, tc := range []struct {
		name, body string
		handler    http.HandlerFunc
		status     int
	}{
		{"unknown field", `{"email":"a@example.test","role":"viewer","extra":true}`, h.Create, 400},
		{"owner role", `{"email":"a@example.test","role":"owner"}`, h.Create, 400},
		{"trailing JSON", `{"email":"a@example.test","role":"viewer"}{}`, h.Create, 400},
		{"oversized", `{"email":"` + strings.Repeat("a", 4096) + `","role":"viewer"}`, h.Create, 413},
		{"invalid registration", `{"token":"` + strings.Repeat("a", 43) + `","email":"bad","password":"short","name":""}`, h.Register, 400},
		{"unauthenticated", `{"token":"` + strings.Repeat("a", 43) + `"}`, h.Accept, 401},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			tc.handler(rr, httptest.NewRequest("POST", "/", strings.NewReader(tc.body)))
			if rr.Code != tc.status {
				t.Fatalf("status %d want %d: %s", rr.Code, tc.status, rr.Body.String())
			}
			var response map[string]string
			if e := json.Unmarshal(rr.Body.Bytes(), &response); e != nil {
				t.Fatal(e)
			}
			if _, ok := response["requestId"]; !ok || response["code"] == "" || response["message"] == "" {
				t.Fatal("missing error contract")
			}
		})
	}
}
func TestInviteHandlerErrorContract(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
		code   string
	}{
		{domain.ErrInvitationExpired, 410, "INVITATION_EXPIRED"},
		{domain.ErrInvitationEmail, 403, "INVITATION_EMAIL_MISMATCH"},
		{domain.ErrInvitationInvalid, 400, "INVITATION_INVALID"},
		{domain.ErrRegistrationRejected, 400, "REGISTRATION_REJECTED"},
		{domain.ErrAccessDenied, 403, "FORBIDDEN"},
		{domain.ErrInaccessible, 404, "NOT_FOUND"},
	} {
		rr := httptest.NewRecorder()
		invitationError(rr, httptest.NewRequest("POST", "/", nil), tc.err)
		if rr.Code != tc.status || !strings.Contains(rr.Body.String(), tc.code) {
			t.Fatalf("%v: %d %s", tc.err, rr.Code, rr.Body.String())
		}
	}
}
