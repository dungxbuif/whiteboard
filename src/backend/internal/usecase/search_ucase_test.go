package usecase

import (
	"context"
	"testing"

	"devtool/internal/domain"

	"github.com/google/uuid"
)

type mockSearchRepository struct {
	items []domain.SearchItem
	query string
	view  string
	limit int
}

func (m *mockSearchRepository) Search(_ context.Context, _ uuid.UUID, query string, view string, limit int) ([]domain.SearchItem, error) {
	m.query = query
	m.view = view
	m.limit = limit
	return m.items, nil
}

func TestSearchUsecaseDefaultsToAllViewAndBoundsResults(t *testing.T) {
	repo := &mockSearchRepository{items: []domain.SearchItem{{ID: uuid.New(), Name: "Board"}}}
	usecase := NewSearchUsecase(repo)

	items, err := usecase.Search(context.Background(), uuid.New(), "  board  ", "")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(items) != 1 || repo.query != "board" || repo.view != domain.SearchViewAll {
		t.Fatalf("unexpected search call: items=%v query=%q view=%q", items, repo.query, repo.view)
	}
	if repo.limit != domain.MaxSearchResults {
		t.Fatalf("expected limit %d, got %d", domain.MaxSearchResults, repo.limit)
	}
}

func TestSearchUsecaseRejectsInvalidViewAndLongQuery(t *testing.T) {
	usecase := NewSearchUsecase(&mockSearchRepository{})
	workspaceID := uuid.New()

	if _, err := usecase.Search(context.Background(), workspaceID, "board", "unknown"); err == nil {
		t.Fatal("expected invalid view error")
	}
	longQuery := "0123456789"
	for len(longQuery) <= domain.MaxSearchQueryLen {
		longQuery += "0123456789"
	}
	if _, err := usecase.Search(context.Background(), workspaceID, longQuery, domain.SearchViewAll); err == nil {
		t.Fatal("expected long query error")
	}
}
