package handler

import (
	"net/http"

	"devtool/internal/delivery/http/middleware"
	"devtool/internal/domain"
)

type SearchHandler struct {
	usecase domain.SearchUsecase
}

func NewSearchHandler(usecase domain.SearchUsecase) *SearchHandler {
	return &SearchHandler{usecase: usecase}
}

func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := middleware.WorkspaceIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	items, err := h.usecase.Search(r.Context(), workspaceID, r.URL.Query().Get("q"), r.URL.Query().Get("view"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSONResponse(w, http.StatusOK, items)
}
