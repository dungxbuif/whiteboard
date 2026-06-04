package handler

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"devtool/internal/domain"
)

type AuthHandler struct {
	authUsecase domain.AuthUsecase
}

func NewAuthHandler(authUsecase domain.AuthUsecase) *AuthHandler {
	return &AuthHandler{
		authUsecase: authUsecase,
	}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	state, err := generateRandomState()
	if err != nil {
		http.Error(w, "failed to generate state", http.StatusInternalServerError)
		return
	}

	cookie := &http.Cookie{
		Name:     domain.AuthCookieName,
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		MaxAge:   domain.OAuthStateMaxAge,
	}
	http.SetCookie(w, cookie)

	url := h.authUsecase.GetGoogleLoginURL(state)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (h *AuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Code is missing", http.StatusBadRequest)
		return
	}

	state := r.URL.Query().Get("state")
	cookie, err := r.Cookie(domain.AuthCookieName)
	if err != nil || cookie.Value != state {
		http.Error(w, "Invalid state", http.StatusBadRequest)
		return
	}

	token, err := h.authUsecase.HandleGoogleCallback(r.Context(), code)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"token":"` + token + `"}`))
}

func generateRandomState() (string, error) {
	const stateLength = 32
	b := make([]byte, stateLength)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
