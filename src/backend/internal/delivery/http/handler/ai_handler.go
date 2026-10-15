package handler

import (
	"encoding/json"
	"net/http"

	"devtool/internal/domain"
	"github.com/go-chi/chi/v5"
)

type AIHandler struct{ usecase domain.AIUsecase }

func NewAIHandler(usecase domain.AIUsecase) *AIHandler { return &AIHandler{usecase: usecase} }

func (h *AIHandler) Handler() http.Handler {
	r := chi.NewRouter()
	r.Post("/chat", h.Chat)
	r.Post("/generate-diagram", h.GenerateDiagram)
	r.Post("/to-code", h.ToCode)
	return r
}

func (h *AIHandler) Chat(w http.ResponseWriter, r *http.Request) { h.handle(w, r, h.usecase.Chat) }
func (h *AIHandler) GenerateDiagram(w http.ResponseWriter, r *http.Request) {
	h.handle(w, r, h.usecase.GenerateDiagram)
}
func (h *AIHandler) ToCode(w http.ResponseWriter, r *http.Request) { h.handle(w, r, h.usecase.ToCode) }

func (h *AIHandler) handle(w http.ResponseWriter, r *http.Request, action func(domain.Context, domain.AIRequest) (*domain.AIResponse, error)) {
	var request domain.AIRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20)).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	response, err := action(r.Context(), request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSONResponse(w, http.StatusOK, response)
}
