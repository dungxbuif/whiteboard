package handler

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"net/http"

	"devtool/internal/delivery/http/middleware"
	"devtool/internal/domain"
)

const oauthStateBytes = 32

type AuthHandler struct {
	authUsecase domain.AuthUsecase
	frontendURL string
}

func NewAuthHandler(authUsecase domain.AuthUsecase, frontendURLs ...string) *AuthHandler {
	frontendURL := "/"
	if len(frontendURLs) > 0 && frontendURLs[0] != "" {
		frontendURL = frontendURLs[0]
	}
	return &AuthHandler{authUsecase: authUsecase, frontendURL: frontendURL}
}

func (h *AuthHandler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	stateBytes := make([]byte, oauthStateBytes)
	if _, err := rand.Read(stateBytes); err != nil {
		http.Error(w, "Could not start Google login", http.StatusInternalServerError)
		return
	}
	state := base64.RawURLEncoding.EncodeToString(stateBytes)
	http.SetCookie(w, &http.Cookie{
		Name:     domain.OAuthStateCookie,
		Value:    state,
		Path:     "/api/v1/auth/google",
		MaxAge:   600,
		HttpOnly: true,
		Secure:   requestIsSecure(r),
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, h.authUsecase.OAuthLoginURL(state), http.StatusFound)
}

func (h *AuthHandler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	if providerError := r.URL.Query().Get("error"); providerError != "" {
		http.Error(w, "Google login was cancelled", http.StatusUnauthorized)
		return
	}
	stateCookie, err := r.Cookie(domain.OAuthStateCookie)
	if err != nil || stateCookie.Value == "" || !constantTimeEqual(stateCookie.Value, r.URL.Query().Get("state")) {
		http.Error(w, "Invalid OAuth state", http.StatusBadRequest)
		return
	}
	clearOAuthCookie(w, requestIsSecure(r))
	token, err := h.authUsecase.OAuthCallback(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		http.Error(w, "Google login failed", http.StatusUnauthorized)
		return
	}
	http.Redirect(w, r, h.frontendURL+"#token="+token, http.StatusFound)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := h.authUsecase.GetUserByID(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(user)
}

func constantTimeEqual(left, right string) bool {
	if left == "" || right == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}

func clearOAuthCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     domain.OAuthStateCookie,
		Value:    "",
		Path:     "/api/v1/auth/google",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func requestIsSecure(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}
